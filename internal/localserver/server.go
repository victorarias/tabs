package localserver

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/victorarias/tabs/internal/config"
	"github.com/victorarias/tabs/internal/daemon"
)

const protocolVersion = "1.0"

type Server struct {
	baseDir    string
	configPath string
	uiPort     int
}

func NewServer(baseDir string, cfg config.Config) *Server {
	cfgPath, err := config.Path()
	if err != nil {
		cfgPath = filepath.Join(baseDir, "config.toml")
	}
	return &Server{
		baseDir:    baseDir,
		configPath: cfgPath,
		uiPort:     cfg.Local.UIPort,
	}
}

func (s *Server) ListenAndServe(ctx context.Context) error {
	addr := net.JoinHostPort("127.0.0.1", intToString(s.uiPort))
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	httpServer := &http.Server{
		Handler:      s.routes(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- httpServer.Serve(listener)
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return httpServer.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/sessions", s.handleSessions)
	mux.HandleFunc("/api/sessions/", s.handleSessionDetail)
	mux.HandleFunc("/api/config", s.handleConfig)
	mux.HandleFunc("/api/daemon/status", s.handleDaemonStatus)
	mux.HandleFunc("/api/sessions/push", s.handlePushSession)
	mux.HandleFunc("/app.js", s.handleStatic)
	mux.HandleFunc("/styles.css", s.handleStatic)
	mux.HandleFunc("/", s.handleRoot)

	return csrfGuard(s.uiPort, mux)
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" && !strings.HasPrefix(r.URL.Path, "/sessions/") && r.URL.Path != "/settings" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	s.serveUI(w, r, "index.html")
}

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	name := strings.TrimPrefix(r.URL.Path, "/")
	s.serveUI(w, r, name)
}

func (s *Server) serveUI(w http.ResponseWriter, r *http.Request, name string) {
	data, err := uiFS.ReadFile("ui/" + name)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if strings.HasSuffix(name, ".css") {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	} else if strings.HasSuffix(name, ".js") {
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	} else {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	filter := SessionFilter{
		Tool: r.URL.Query().Get("tool"),
		Date: r.URL.Query().Get("date"),
		Cwd:  r.URL.Query().Get("cwd"),
		Q:    r.URL.Query().Get("q"),
	}

	sessions, err := ListSessions(s.baseDir, filter)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "server_error", "Failed to load sessions")
		return
	}

	resp := SessionsResponse{
		Sessions: sessions,
		Total:    len(sessions),
	}
	s.writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleSessionDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	sessionID := strings.TrimPrefix(r.URL.Path, "/api/sessions/")
	if sessionID == "" || sessionID == "/" {
		s.writeError(w, http.StatusBadRequest, "invalid_request", "Missing session id")
		return
	}

	session, err := GetSession(s.baseDir, sessionID)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			s.writeError(w, http.StatusNotFound, "session_not_found", "Session not found")
			return
		}
		s.writeError(w, http.StatusInternalServerError, "server_error", "Failed to load session")
		return
	}

	resp := map[string]interface{}{
		"session": session,
	}
	s.writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		cfg, err := s.loadConfig()
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "server_error", "Failed to load config")
			return
		}
		apiKeyConfigured := cfg.Remote.APIKey != ""
		apiKeyPrefix := ""
		if apiKeyConfigured {
			apiKeyPrefix = cfg.Remote.APIKey
			if len(apiKeyPrefix) > 12 {
				apiKeyPrefix = apiKeyPrefix[:12]
			}
		}
		resp := map[string]interface{}{
			"local": map[string]interface{}{
				"ui_port":   cfg.Local.UIPort,
				"log_level": cfg.Local.LogLevel,
			},
			"remote": map[string]interface{}{
				"server_url":         cfg.Remote.ServerURL,
				"api_key_configured": apiKeyConfigured,
				"api_key_prefix":     apiKeyPrefix,
				"default_tags":       cfg.Remote.DefaultTags,
			},
		}
		s.writeJSON(w, http.StatusOK, resp)
	case http.MethodPut:
		var payload struct {
			Remote struct {
				ServerURL *string `json:"server_url"`
				APIKey    *string `json:"api_key"`
			} `json:"remote"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			s.writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
			return
		}

		cfg, err := s.loadConfig()
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "server_error", "Failed to load config")
			return
		}

		if payload.Remote.ServerURL != nil {
			if err := config.ApplySet(&cfg, "server_url", *payload.Remote.ServerURL); err != nil {
				s.writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
				return
			}
		}
		if payload.Remote.APIKey != nil {
			if err := config.ApplySet(&cfg, "api_key", *payload.Remote.APIKey); err != nil {
				s.writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
				return
			}
		}

		if err := config.Write(s.configPath, cfg); err != nil {
			s.writeError(w, http.StatusInternalServerError, "server_error", "Failed to update config")
			return
		}

		resp := map[string]interface{}{
			"status":  "ok",
			"message": "Configuration updated",
		}
		s.writeJSON(w, http.StatusOK, resp)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleDaemonStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	status, err := fetchDaemonStatus(s.baseDir)
	if err != nil {
		s.writeJSON(w, http.StatusOK, map[string]interface{}{"running": false})
		return
	}

	resp := map[string]interface{}{
		"running":           true,
		"pid":               status.PID,
		"uptime_seconds":    status.UptimeSeconds,
		"sessions_captured": status.SessionsCaptured,
		"events_processed":  status.EventsProcessed,
	}
	s.writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handlePushSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var payload struct {
		SessionID string    `json:"session_id"`
		Tool      string    `json:"tool"`
		Tags      []pushTag `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}
	if strings.TrimSpace(payload.SessionID) == "" || strings.TrimSpace(payload.Tool) == "" {
		s.writeError(w, http.StatusBadRequest, "invalid_request", "session_id and tool are required")
		return
	}

	result, err := pushSessionToDaemon(s.baseDir, payload.SessionID, payload.Tool, payload.Tags)
	if err != nil {
		if resp, ok := err.(daemonResponseError); ok {
			s.writeError(w, http.StatusBadRequest, resp.Code, resp.Message)
			return
		}
		s.writeError(w, http.StatusInternalServerError, "server_error", "Failed to push session")
		return
	}

	resp := map[string]interface{}{
		"status":    "ok",
		"remote_id": result.RemoteID,
		"url":       result.URL,
	}
	s.writeJSON(w, http.StatusOK, resp)
}

type daemonResponseError struct {
	Code    string
	Message string
}

func (e daemonResponseError) Error() string {
	return e.Message
}

type pushResult struct {
	RemoteID string
	URL      string
}

type pushTag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func pushSessionToDaemon(baseDir, sessionID, tool string, tags []pushTag) (pushResult, error) {
	socketPath := daemon.SocketPath(baseDir)
	conn, err := net.DialTimeout("unix", socketPath, 2*time.Second)
	if err != nil {
		return pushResult{}, err
	}
	defer conn.Close()

	req := map[string]interface{}{
		"version": protocolVersion,
		"type":    "push_session",
		"payload": map[string]interface{}{
			"session_id": sessionID,
			"tool":       tool,
			"tags":       tags,
		},
	}
	if err := conn.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return pushResult{}, err
	}
	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return pushResult{}, err
	}

	var resp struct {
		Version string          `json:"version"`
		Status  string          `json:"status"`
		Data    json.RawMessage `json:"data"`
		Error   *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return pushResult{}, err
	}
	if resp.Status != "ok" {
		if resp.Error != nil {
			return pushResult{}, daemonResponseError{Code: resp.Error.Code, Message: resp.Error.Message}
		}
		return pushResult{}, daemonResponseError{Code: "server_error", Message: "push failed"}
	}

	var data struct {
		RemoteID string `json:"remote_id"`
		URL      string `json:"url"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return pushResult{}, err
	}
	return pushResult{RemoteID: data.RemoteID, URL: data.URL}, nil
}

func (s *Server) loadConfig() (config.Config, error) {
	cfg, err := config.Load(s.configPath)
	if err == nil {
		return cfg, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return config.Default(), nil
	}
	return config.Config{}, err
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func (s *Server) writeError(w http.ResponseWriter, status int, code, message string) {
	resp := ErrorResponse{
		Status: "error",
		Error: ErrorPayload{
			Code:    code,
			Message: message,
		},
	}
	s.writeJSON(w, status, resp)
}

func fetchDaemonStatus(baseDir string) (daemon.Status, error) {
	socketPath := daemon.SocketPath(baseDir)
	conn, err := net.DialTimeout("unix", socketPath, 2*time.Second)
	if err != nil {
		return daemon.Status{}, err
	}
	defer conn.Close()

	req := map[string]interface{}{
		"version": protocolVersion,
		"type":    "daemon_status",
		"payload": map[string]interface{}{},
	}
	if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return daemon.Status{}, err
	}
	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return daemon.Status{}, err
	}

	var resp struct {
		Version string          `json:"version"`
		Status  string          `json:"status"`
		Data    json.RawMessage `json:"data"`
		Error   *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return daemon.Status{}, err
	}
	if resp.Status != "ok" {
		if resp.Error != nil {
			return daemon.Status{}, errors.New(resp.Error.Message)
		}
		return daemon.Status{}, errors.New("daemon status failed")
	}
	var status daemon.Status
	if err := json.Unmarshal(resp.Data, &status); err != nil {
		return daemon.Status{}, err
	}
	return status, nil
}

func csrfGuard(port int, next http.Handler) http.Handler {
	allowedOrigins := map[string]struct{}{
		"http://localhost:" + intToString(port): {},
		"http://127.0.0.1:" + intToString(port): {},
	}
	allowedHosts := map[string]struct{}{
		"localhost:" + intToString(port): {},
		"127.0.0.1:" + intToString(port): {},
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodDelete:
			if _, ok := allowedHosts[r.Host]; !ok {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			if origin := r.Header.Get("Origin"); origin != "" {
				if _, ok := allowedOrigins[origin]; !ok {
					w.WriteHeader(http.StatusForbidden)
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

func intToString(value int) string {
	return strconv.Itoa(value)
}
