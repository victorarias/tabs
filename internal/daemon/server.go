package daemon

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"sync"
	"time"

	"github.com/victorarias/tabs/internal/logging"
)

const protocolVersion = "1.0"

type Server struct {
	baseDir    string
	socketPath string
	logger     *slog.Logger
	listener   net.Listener
	wg         sync.WaitGroup
	mu         sync.Mutex
	state      *State
}

func NewServer(baseDir string, logger *slog.Logger) *Server {
	if logger == nil {
		logger = logging.New("info", os.Stdout)
	}
	return &Server{
		baseDir:    baseDir,
		socketPath: SocketPath(baseDir),
		logger:     logger,
		state:      NewState(),
	}
}

func (s *Server) Listen() error {
	if err := os.Remove(s.socketPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove existing socket: %w", err)
	}
	listener, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("listen on socket: %w", err)
	}
	if err := os.Chmod(s.socketPath, 0o600); err != nil {
		listener.Close()
		return fmt.Errorf("chmod socket: %w", err)
	}
	s.listener = listener
	return nil
}

func (s *Server) Serve(ctx context.Context) error {
	if s.listener == nil {
		return errors.New("listener not initialized")
	}
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
			}
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				continue
			}
			return err
		}
		s.wg.Add(1)
		go s.handleConn(conn)
	}
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.listener != nil {
		_ = s.listener.Close()
	}
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
	}

	if err := os.Remove(s.socketPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (s *Server) handleConn(conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	reader := bufio.NewReader(conn)
	line, err := reader.ReadBytes('\n')
	if err != nil {
		if errors.Is(err, io.EOF) {
			return
		}
		s.logger.Error("read request failed", "error", err)
		return
	}

	var req request
	if err := json.Unmarshal(bytesTrimSpace(line), &req); err != nil {
		s.writeResponse(conn, errorResponse("invalid_json", "Invalid JSON request"))
		return
	}

	if req.Version != protocolVersion {
		s.writeResponse(conn, errorResponse("unsupported_version", "Unsupported protocol version"))
		return
	}

	switch req.Type {
	case "capture_event":
		s.handleCapture(conn, req.Payload)
	case "push_session":
		s.handlePush(conn, req.Payload)
	case "daemon_status":
		s.handleStatus(conn)
	default:
		s.writeResponse(conn, errorResponse("unsupported_type", "Unsupported request type"))
	}
}

func (s *Server) handleCapture(conn net.Conn, payload json.RawMessage) {
	var req capturePayload
	if err := json.Unmarshal(payload, &req); err != nil {
		s.writeResponse(conn, errorResponse("invalid_payload", "Invalid capture payload"))
		return
	}
	if req.Tool != "claude-code" && req.Tool != "cursor" {
		s.writeResponse(conn, errorResponse("unknown_tool", "Unsupported tool"))
		return
	}
	if req.Event == nil {
		s.writeResponse(conn, errorResponse("invalid_payload", "Missing event payload"))
		return
	}

	sessionID, ok := req.Event["session_id"].(string)
	if !ok || sessionID == "" {
		s.writeResponse(conn, errorResponse("invalid_payload", "Missing required field: session_id"))
		return
	}

	eventTime := time.Now().UTC()
	if req.Timestamp != "" {
		if ts, err := time.Parse(time.RFC3339Nano, req.Timestamp); err == nil {
			eventTime = ts
		}
	}

	if req.Tool == "claude-code" {
		eventsWritten, lastEventTime, err := s.captureClaude(req, sessionID, eventTime)
		if err != nil {
			s.writeResponse(conn, errorResponse("storage_error", err.Error()))
			return
		}
		s.mu.Lock()
		if eventsWritten > 0 {
			s.state.RecordEvent(sessionID, lastEventTime, eventsWritten)
		}
		s.mu.Unlock()
		data := map[string]interface{}{
			"session_id":     sessionID,
			"events_written": eventsWritten,
		}
		s.writeResponse(conn, okResponse(data))
		return
	}
	if req.Tool == "cursor" {
		eventsWritten, lastEventTime, err := s.captureCursor(req, sessionID, eventTime)
		if err != nil {
			s.writeResponse(conn, errorResponse("storage_error", err.Error()))
			return
		}
		s.mu.Lock()
		if eventsWritten > 0 {
			s.state.RecordEvent(sessionID, lastEventTime, eventsWritten)
		}
		s.mu.Unlock()
		data := map[string]interface{}{
			"session_id":     sessionID,
			"events_written": eventsWritten,
		}
		s.writeResponse(conn, okResponse(data))
		return
	}

	normalized, eventJSON, err := normalizeEvent(req.Event, sessionID, req.Tool, eventTime)
	if err != nil {
		s.writeResponse(conn, errorResponse("invalid_payload", "Invalid event payload"))
		return
	}

	s.mu.Lock()
	cursor, cursorErr := loadCursorState(s.baseDir, sessionID)
	if cursorErr != nil {
		s.logger.Warn("cursor state load failed", "session_id", sessionID, "error", cursorErr)
	}
	if cursor == nil {
		s.mu.Unlock()
		s.writeResponse(conn, errorResponse("storage_error", "Failed to read cursor state"))
		return
	}

	lineHash := hashLine(eventJSON)
	if cursor.LastLineHash == lineHash {
		s.mu.Unlock()
		data := map[string]interface{}{
			"session_id":     sessionID,
			"events_written": 0,
		}
		s.writeResponse(conn, okResponse(data))
		return
	}

	sessionPath, err := s.state.EnsureSessionFile(s.baseDir, sessionID, req.Tool, eventTime)
	if err != nil {
		s.mu.Unlock()
		s.writeResponse(conn, errorResponse("storage_error", "Failed to resolve session file"))
		return
	}

	lastOffset, err := appendJSONL(sessionPath, eventJSON)
	if err != nil {
		s.mu.Unlock()
		s.writeResponse(conn, errorResponse("storage_error", "Failed to append event"))
		return
	}

	meta := extractEventMetadata(normalized)
	updateCursorState(cursor, meta, normalized, lineHash, lastOffset, sessionPath)
	if err := saveCursorState(s.baseDir, cursor); err != nil {
		s.mu.Unlock()
		s.writeResponse(conn, errorResponse("storage_error", "Failed to update cursor state"))
		return
	}

	s.state.RecordEvent(sessionID, eventTime, 1)
	s.mu.Unlock()

	data := map[string]interface{}{
		"session_id":     sessionID,
		"events_written": 1,
	}
	s.writeResponse(conn, okResponse(data))
}

func (s *Server) handleStatus(conn net.Conn) {
	pid := os.Getpid()
	s.mu.Lock()
	status := s.state.Snapshot(pid)
	s.mu.Unlock()
	s.writeResponse(conn, okResponse(status))
}

func (s *Server) handlePush(conn net.Conn, payload json.RawMessage) {
	var req pushPayload
	if err := json.Unmarshal(payload, &req); err != nil {
		s.writeResponse(conn, errorResponse("invalid_payload", "Invalid push payload"))
		return
	}

	result, err := handlePushSession(s.baseDir, req)
	if err != nil {
		if perr, ok := err.(*pushError); ok {
			s.writeResponse(conn, errorResponse(perr.Code, perr.Message))
			return
		}
		s.writeResponse(conn, errorResponse("storage_error", err.Error()))
		return
	}

	data := map[string]interface{}{
		"remote_id": result.RemoteID,
		"url":       result.URL,
	}
	if result.RemoteID == "" && result.URL == "" {
		s.writeResponse(conn, okResponse(data))
		return
	}
	s.writeResponse(conn, okResponse(data))
}

func (s *Server) writeResponse(conn net.Conn, resp response) {
	payload, err := json.Marshal(resp)
	if err != nil {
		s.logger.Error("marshal response failed", "error", err)
		return
	}
	payload = append(payload, '\n')
	if _, err := conn.Write(payload); err != nil {
		s.logger.Error("write response failed", "error", err)
	}
}

type request struct {
	Version string          `json:"version"`
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type response struct {
	Version string         `json:"version"`
	Status  string         `json:"status"`
	Data    interface{}    `json:"data,omitempty"`
	Error   *responseError `json:"error,omitempty"`
}

type responseError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type capturePayload struct {
	Tool      string                 `json:"tool"`
	Timestamp string                 `json:"timestamp"`
	Event     map[string]interface{} `json:"event"`
}

func okResponse(data interface{}) response {
	return response{Version: protocolVersion, Status: "ok", Data: data}
}

func errorResponse(code, message string) response {
	return response{Version: protocolVersion, Status: "error", Error: &responseError{Code: code, Message: message}}
}

func bytesTrimSpace(input []byte) []byte {
	start := 0
	end := len(input)
	for start < end {
		b := input[start]
		if b != ' ' && b != '\n' && b != '\r' && b != '\t' {
			break
		}
		start++
	}
	for end > start {
		b := input[end-1]
		if b != ' ' && b != '\n' && b != '\r' && b != '\t' {
			break
		}
		end--
	}
	return input[start:end]
}
