package daemon

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

type SessionCursor struct {
	SessionID      string           `json:"session_id"`
	TranscriptPath string           `json:"transcript_path,omitempty"`
	LastOffset     int64            `json:"last_offset"`
	LastLineHash   string           `json:"last_line_hash"`
	UpdatedAt      string           `json:"updated_at"`
	Metadata       *SessionMetadata `json:"metadata,omitempty"`
}

type SessionMetadata struct {
	SessionID       string `json:"session_id"`
	Tool            string `json:"tool"`
	CreatedAt       string `json:"created_at,omitempty"`
	EndedAt         string `json:"ended_at,omitempty"`
	Cwd             string `json:"cwd,omitempty"`
	DurationSeconds int    `json:"duration_seconds,omitempty"`
	MessageCount    int    `json:"message_count,omitempty"`
	ToolUseCount    int    `json:"tool_use_count,omitempty"`
	FilePath        string `json:"file_path,omitempty"`
}

type eventMetadata struct {
	EventType string
	Tool      string
	SessionID string
	Timestamp time.Time
	Data      map[string]interface{}
}

func normalizeEvent(event map[string]interface{}, sessionID, tool string, ts time.Time) (map[string]interface{}, []byte, error) {
	if event == nil {
		return nil, nil, fmt.Errorf("event payload is nil")
	}
	normalized := make(map[string]interface{}, len(event)+3)
	for k, v := range event {
		normalized[k] = v
	}
	normalized["session_id"] = sessionID
	normalized["tool"] = tool
	normalized["timestamp"] = ts.UTC().Format(time.RFC3339Nano)
	data, err := json.Marshal(normalized)
	if err != nil {
		return nil, nil, err
	}
	return normalized, data, nil
}

func hashLine(line []byte) string {
	sum := sha256.Sum256(line)
	return hex.EncodeToString(sum[:])
}

func appendJSONL(path string, line []byte) (int64, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		return 0, err
	}
	defer func() {
		_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
	}()

	line = ensureNewline(line)
	if _, err := file.Write(line); err != nil {
		return 0, err
	}
	if err := file.Sync(); err != nil {
		return 0, err
	}
	info, err := file.Stat()
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

func ensureNewline(line []byte) []byte {
	if len(line) == 0 {
		return []byte{'\n'}
	}
	if line[len(line)-1] == '\n' {
		return line
	}
	out := make([]byte, len(line)+1)
	copy(out, line)
	out[len(out)-1] = '\n'
	return out
}

func loadCursorState(baseDir, sessionID string) (*SessionCursor, error) {
	path := cursorStatePath(baseDir, sessionID)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &SessionCursor{SessionID: sessionID}, nil
		}
		return nil, err
	}
	var cursor SessionCursor
	if err := json.Unmarshal(data, &cursor); err != nil {
		return &SessionCursor{SessionID: sessionID}, fmt.Errorf("decode cursor state: %w", err)
	}
	if cursor.SessionID == "" {
		cursor.SessionID = sessionID
	}
	return &cursor, nil
}

func saveCursorState(baseDir string, cursor *SessionCursor) error {
	if cursor == nil {
		return fmt.Errorf("cursor state is nil")
	}
	cursor.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	data, err := json.Marshal(cursor)
	if err != nil {
		return err
	}
	return writeFileAtomic(cursorStatePath(baseDir, cursor.SessionID), data, 0o600)
}

func cursorStatePath(baseDir, sessionID string) string {
	return filepath.Join(StateDir(baseDir), sessionID+".json")
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		_ = os.Remove(tmpName)
	}()
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func extractEventMetadata(event map[string]interface{}) eventMetadata {
	var meta eventMetadata
	if value, ok := event["event_type"].(string); ok {
		meta.EventType = value
	}
	if value, ok := event["tool"].(string); ok {
		meta.Tool = value
	}
	if value, ok := event["session_id"].(string); ok {
		meta.SessionID = value
	}
	if value, ok := event["timestamp"].(string); ok {
		if ts, err := time.Parse(time.RFC3339Nano, value); err == nil {
			meta.Timestamp = ts
		}
	}
	if data, ok := event["data"].(map[string]interface{}); ok {
		meta.Data = data
	}
	return meta
}

func updateCursorState(cursor *SessionCursor, meta eventMetadata, event map[string]interface{}, lineHash string, lastOffset int64, filePath string) {
	cursor.SessionID = meta.SessionID
	if cursor.TranscriptPath == "" {
		if path := extractTranscriptPath(event); path != "" {
			cursor.TranscriptPath = path
		}
	}
	cursor.LastLineHash = lineHash
	cursor.LastOffset = lastOffset
	updateCursorMetadata(cursor, meta, filePath)
}

func extractTranscriptPath(event map[string]interface{}) string {
	if value, ok := event["transcript_path"].(string); ok {
		return value
	}
	if data, ok := event["data"].(map[string]interface{}); ok {
		if value, ok := data["transcript_path"].(string); ok {
			return value
		}
	}
	return ""
}

func updateCursorMetadata(cursor *SessionCursor, meta eventMetadata, filePath string) {
	if cursor.Metadata == nil {
		cursor.Metadata = &SessionMetadata{}
	}
	md := cursor.Metadata
	if md.SessionID == "" {
		md.SessionID = meta.SessionID
	}
	if md.Tool == "" {
		md.Tool = meta.Tool
	}
	if md.FilePath == "" {
		md.FilePath = filePath
	}
	if md.CreatedAt == "" && !meta.Timestamp.IsZero() {
		md.CreatedAt = meta.Timestamp.UTC().Format(time.RFC3339Nano)
	}
	if meta.EventType == "session_start" && meta.Data != nil {
		if cwd, ok := meta.Data["cwd"].(string); ok && cwd != "" {
			md.Cwd = cwd
		}
	}
	switch meta.EventType {
	case "message":
		md.MessageCount++
	case "tool_use":
		md.ToolUseCount++
	case "session_end":
		if !meta.Timestamp.IsZero() {
			md.EndedAt = meta.Timestamp.UTC().Format(time.RFC3339Nano)
		}
		if meta.Data != nil {
			if value, ok := toInt(meta.Data["duration_seconds"]); ok {
				md.DurationSeconds = value
			}
			if value, ok := toInt(meta.Data["message_count"]); ok {
				md.MessageCount = value
			}
			if value, ok := toInt(meta.Data["tool_use_count"]); ok {
				md.ToolUseCount = value
			}
		}
	}
}

func toInt(value interface{}) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case json.Number:
		parsed, err := strconv.ParseInt(string(v), 10, 64)
		if err != nil {
			return 0, false
		}
		return int(parsed), true
	case string:
		parsed, err := strconv.Atoi(v)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}
