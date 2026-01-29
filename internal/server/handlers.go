package server

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgconn"
)

var errInvalidAPIKey = errors.New("invalid api key")

func (s *Server) handleUploadSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	apiKey, err := parseBearerToken(r.Header.Get("Authorization"))
	if err != nil {
		s.writeError(w, http.StatusUnauthorized, "invalid_api_key", "Invalid or missing API key")
		return
	}

	ctx := r.Context()
	keyRecord, err := s.lookupAPIKey(ctx, apiKey)
	if err != nil {
		s.writeError(w, http.StatusUnauthorized, "invalid_api_key", "Invalid or expired API key")
		return
	}

	var req UploadRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	normalized, err := normalizeUpload(req)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	exists, err := s.sessionExists(ctx, normalized.Tool, normalized.SessionID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "server_error", "Failed to check session")
		return
	}
	if exists {
		s.writeError(w, http.StatusConflict, "duplicate_session", "Session already uploaded")
		return
	}

	remoteID, err := s.storeSession(ctx, normalized, keyRecord)
	if err != nil {
		if errors.Is(err, errDuplicateSession) {
			s.writeError(w, http.StatusConflict, "duplicate_session", "Session already uploaded")
			return
		}
		s.writeError(w, http.StatusInternalServerError, "server_error", "Failed to store session")
		return
	}

	resp := map[string]string{
		"id":  remoteID,
		"url": strings.TrimRight(s.baseURL, "/") + "/sessions/" + remoteID,
	}
	s.writeJSON(w, http.StatusCreated, resp)
}

type apiKeyRecord struct {
	ID     string
	UserID string
}

func (s *Server) lookupAPIKey(ctx context.Context, key string) (apiKeyRecord, error) {
	if !strings.HasPrefix(key, "tabs_") {
		return apiKeyRecord{}, errInvalidAPIKey
	}
	hash := sha256.Sum256([]byte(key))
	hashHex := hex.EncodeToString(hash[:])

	var record apiKeyRecord
	var isActive bool
	var expiresAt sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, is_active, expires_at
		FROM api_keys
		WHERE key_hash = $1
	`, hashHex).Scan(&record.ID, &record.UserID, &isActive, &expiresAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return apiKeyRecord{}, errInvalidAPIKey
		}
		return apiKeyRecord{}, err
	}
	if !isActive {
		return apiKeyRecord{}, errInvalidAPIKey
	}
	if expiresAt.Valid && time.Now().After(expiresAt.Time) {
		return apiKeyRecord{}, errInvalidAPIKey
	}
	return record, nil
}

func (s *Server) sessionExists(ctx context.Context, tool, sessionID string) (bool, error) {
	var id string
	err := s.db.QueryRowContext(ctx, `SELECT id FROM sessions WHERE tool = $1 AND session_id = $2`, tool, sessionID).Scan(&id)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return false, err
}

var errDuplicateSession = errors.New("duplicate session")

func (s *Server) storeSession(ctx context.Context, session NormalizedSession, key apiKeyRecord) (string, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	var endedAt interface{}
	if session.EndedAt != nil {
		endedAt = *session.EndedAt
	}
	var duration interface{}
	if session.DurationSeconds != nil {
		duration = *session.DurationSeconds
	}

	var remoteID string
	err = tx.QueryRowContext(ctx, `
		INSERT INTO sessions (
			tool, session_id, created_at, ended_at, cwd, uploaded_by, api_key_id,
			duration_seconds, message_count, tool_use_count
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING id
	`, session.Tool, session.SessionID, session.CreatedAt, endedAt, session.Cwd, key.UserID, key.ID,
		duration, session.MessageCount, session.ToolUseCount).Scan(&remoteID)
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok {
			if pgErr.Code == "23505" {
				return "", errDuplicateSession
			}
		}
		return "", err
	}

	if err := insertMessages(ctx, tx, remoteID, session.Messages); err != nil {
		return "", err
	}
	if err := insertTools(ctx, tx, remoteID, session.Tools); err != nil {
		return "", err
	}
	if err := insertTags(ctx, tx, remoteID, session.Tags); err != nil {
		return "", err
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE api_keys
		SET last_used_at = NOW(), usage_count = usage_count + 1
		WHERE id = $1
	`, key.ID); err != nil {
		return "", err
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}
	return remoteID, nil
}

func insertMessages(ctx context.Context, tx *sql.Tx, sessionID string, messages []MessageRecord) error {
	if len(messages) == 0 {
		return nil
	}
	stmt := `INSERT INTO messages (session_id, timestamp, seq, role, model, content) VALUES ($1,$2,$3,$4,$5,$6)`
	for _, msg := range messages {
		var model interface{}
		if msg.Model != nil && *msg.Model != "" {
			model = *msg.Model
		}
		content := msg.Content
		if len(content) == 0 {
			content = json.RawMessage("[]")
		}
		if _, err := tx.ExecContext(ctx, stmt, sessionID, msg.Timestamp, msg.Seq, msg.Role, model, []byte(content)); err != nil {
			return err
		}
	}
	return nil
}

func insertTools(ctx context.Context, tx *sql.Tx, sessionID string, tools []ToolRecord) error {
	if len(tools) == 0 {
		return nil
	}
	sort.Slice(tools, func(i, j int) bool {
		return tools[i].Timestamp.Before(tools[j].Timestamp)
	})
	stmt := `INSERT INTO tools (session_id, timestamp, tool_use_id, tool_name, input, output, is_error) VALUES ($1,$2,$3,$4,$5,$6,$7)`
	for _, tool := range tools {
		input := tool.Input
		if len(input) == 0 {
			input = json.RawMessage("null")
		}
		var output interface{}
		if len(tool.Output) > 0 {
			output = []byte(tool.Output)
		}
		if _, err := tx.ExecContext(ctx, stmt, sessionID, tool.Timestamp, tool.ToolUseID, tool.ToolName, []byte(input), output, tool.IsError); err != nil {
			return err
		}
	}
	return nil
}

func insertTags(ctx context.Context, tx *sql.Tx, sessionID string, tags []Tag) error {
	if len(tags) == 0 {
		return nil
	}
	stmt := `INSERT INTO tags (session_id, tag_key, tag_value) VALUES ($1,$2,$3)`
	for _, tag := range tags {
		if _, err := tx.ExecContext(ctx, stmt, sessionID, tag.Key, tag.Value); err != nil {
			return err
		}
	}
	return nil
}

func parseBearerToken(header string) (string, error) {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", errInvalidAPIKey
	}
	if parts[1] == "" {
		return "", errInvalidAPIKey
	}
	return parts[1], nil
}

func normalizeUpload(req UploadRequest) (NormalizedSession, error) {
	if req.Session.SessionID == "" {
		return NormalizedSession{}, errors.New("missing session.session_id")
	}
	if _, err := uuid.Parse(req.Session.SessionID); err != nil {
		return NormalizedSession{}, errors.New("session.session_id must be a valid UUID")
	}
	if !isValidTool(req.Session.Tool) {
		return NormalizedSession{}, errors.New("session.tool must be claude-code or cursor")
	}
	if len(req.Session.Events) == 0 {
		return NormalizedSession{}, errors.New("session.events must not be empty")
	}

	normalized := NormalizedSession{
		Tool:      req.Session.Tool,
		SessionID: req.Session.SessionID,
		Cwd:       strings.TrimSpace(req.Session.Cwd),
		Tags:      dedupeTags(req.Tags),
	}

	if normalized.Cwd == "" {
		normalized.Cwd = ""
	}

	createdAt, err := parseOptionalTime(req.Session.CreatedAt)
	if err != nil {
		return NormalizedSession{}, errors.New("session.created_at must be RFC3339")
	}
	endedAt, err := parseOptionalTime(req.Session.EndedAt)
	if err != nil {
		return NormalizedSession{}, errors.New("session.ended_at must be RFC3339")
	}
	if !createdAt.IsZero() {
		normalized.CreatedAt = createdAt
	}
	if !endedAt.IsZero() {
		normalized.EndedAt = &endedAt
	}

	var earliest time.Time
	var latest time.Time
	var messages []MessageRecord
	var toolUseCount int
	toolMap := make(map[string]*ToolRecord)
	seq := 0

	var sessionEndDuration *int
	var sessionEndMsgCount *int
	var sessionEndToolCount *int

	for _, event := range req.Session.Events {
		if event.EventType == "" {
			continue
		}
		ts, err := parseTime(event.Timestamp)
		if err != nil {
			return NormalizedSession{}, errors.New("event.timestamp must be RFC3339")
		}
		if earliest.IsZero() || ts.Before(earliest) {
			earliest = ts
		}
		if latest.IsZero() || ts.After(latest) {
			latest = ts
		}
		if event.SessionID != "" && event.SessionID != req.Session.SessionID {
			return NormalizedSession{}, errors.New("event.session_id must match session.session_id")
		}
		if event.Tool != "" && event.Tool != req.Session.Tool {
			return NormalizedSession{}, errors.New("event.tool must match session.tool")
		}

		switch event.EventType {
		case "session_start":
			var data struct {
				Cwd string `json:"cwd"`
			}
			if len(event.Data) > 0 {
				_ = json.Unmarshal(event.Data, &data)
			}
			if normalized.Cwd == "" && strings.TrimSpace(data.Cwd) != "" {
				normalized.Cwd = strings.TrimSpace(data.Cwd)
			}
			if normalized.CreatedAt.IsZero() {
				normalized.CreatedAt = ts
			}
		case "session_end":
			var data struct {
				DurationSeconds *int `json:"duration_seconds"`
				MessageCount    *int `json:"message_count"`
				ToolUseCount    *int `json:"tool_use_count"`
			}
			if len(event.Data) > 0 {
				_ = json.Unmarshal(event.Data, &data)
			}
			if normalized.EndedAt == nil {
				ended := ts
				normalized.EndedAt = &ended
			}
			if data.DurationSeconds != nil {
				sessionEndDuration = data.DurationSeconds
			}
			if data.MessageCount != nil {
				sessionEndMsgCount = data.MessageCount
			}
			if data.ToolUseCount != nil {
				sessionEndToolCount = data.ToolUseCount
			}
		case "message":
			var data struct {
				Role    string          `json:"role"`
				Content json.RawMessage `json:"content"`
				Model   string          `json:"model"`
			}
			if err := json.Unmarshal(event.Data, &data); err != nil {
				return NormalizedSession{}, errors.New("invalid message event")
			}
			if data.Role != "user" && data.Role != "assistant" {
				return NormalizedSession{}, errors.New("message.role must be user or assistant")
			}
			seq++
			var model *string
			if data.Model != "" {
				model = &data.Model
			}
			messages = append(messages, MessageRecord{
				Timestamp: ts,
				Seq:       seq,
				Role:      data.Role,
				Model:     model,
				Content:   data.Content,
			})
		case "tool_use":
			var data struct {
				ToolUseID string          `json:"tool_use_id"`
				ToolName  string          `json:"tool_name"`
				Input     json.RawMessage `json:"input"`
			}
			if err := json.Unmarshal(event.Data, &data); err != nil {
				return NormalizedSession{}, errors.New("invalid tool_use event")
			}
			if data.ToolUseID == "" || data.ToolName == "" {
				return NormalizedSession{}, errors.New("tool_use requires tool_use_id and tool_name")
			}
			rec, ok := toolMap[data.ToolUseID]
			if !ok {
				rec = &ToolRecord{ToolUseID: data.ToolUseID}
				toolMap[data.ToolUseID] = rec
			}
			rec.Timestamp = ts
			rec.ToolName = data.ToolName
			rec.Input = data.Input
			toolUseCount++
		case "tool_result":
			var data struct {
				ToolUseID string `json:"tool_use_id"`
				Content   string `json:"content"`
				IsError   bool   `json:"is_error"`
			}
			if err := json.Unmarshal(event.Data, &data); err != nil {
				return NormalizedSession{}, errors.New("invalid tool_result event")
			}
			if data.ToolUseID == "" {
				return NormalizedSession{}, errors.New("tool_result requires tool_use_id")
			}
			rec, ok := toolMap[data.ToolUseID]
			if !ok {
				rec = &ToolRecord{ToolUseID: data.ToolUseID}
				toolMap[data.ToolUseID] = rec
			}
			if rec.Timestamp.IsZero() {
				rec.Timestamp = ts
			}
			rec.Output = marshalOutput(data.Content)
			rec.IsError = data.IsError
		case "schema_version":
			// ignore
		default:
			// ignore unknown event types
		}
	}

	if normalized.CreatedAt.IsZero() {
		if !earliest.IsZero() {
			normalized.CreatedAt = earliest
		} else {
			return NormalizedSession{}, errors.New("missing session.created_at")
		}
	}
	if normalized.EndedAt == nil && !latest.IsZero() {
		ended := latest
		normalized.EndedAt = &ended
	}
	if normalized.Cwd == "" {
		return NormalizedSession{}, errors.New("missing session.cwd")
	}

	normalized.Messages = messages
	if sessionEndMsgCount != nil {
		normalized.MessageCount = *sessionEndMsgCount
	} else {
		normalized.MessageCount = len(messages)
	}
	if sessionEndToolCount != nil {
		normalized.ToolUseCount = *sessionEndToolCount
	} else {
		normalized.ToolUseCount = toolUseCount
	}
	if sessionEndDuration != nil {
		normalized.DurationSeconds = sessionEndDuration
	} else if normalized.EndedAt != nil {
		dur := int(normalized.EndedAt.Sub(normalized.CreatedAt).Seconds())
		if dur < 0 {
			dur = 0
		}
		normalized.DurationSeconds = &dur
	}

	for _, rec := range toolMap {
		if rec.ToolName == "" {
			return NormalizedSession{}, errors.New("tool_result without tool_name")
		}
		normalized.Tools = append(normalized.Tools, *rec)
	}

	return normalized, nil
}

func parseTime(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, errors.New("empty timestamp")
	}
	if ts, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return ts, nil
	}
	return time.Parse(time.RFC3339, value)
}

func parseOptionalTime(value string) (time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return time.Time{}, nil
	}
	return parseTime(value)
}

func isValidTool(tool string) bool {
	switch tool {
	case "claude-code", "cursor":
		return true
	default:
		return false
	}
}

func marshalOutput(content string) json.RawMessage {
	if content == "" {
		return json.RawMessage("null")
	}
	payload, err := json.Marshal(map[string]string{"content": content})
	if err != nil {
		return json.RawMessage("null")
	}
	return payload
}

func dedupeTags(tags []Tag) []Tag {
	if len(tags) == 0 {
		return nil
	}
	seen := make(map[string]struct{})
	out := make([]Tag, 0, len(tags))
	for _, tag := range tags {
		key := strings.TrimSpace(tag.Key)
		value := strings.TrimSpace(tag.Value)
		if key == "" || value == "" {
			continue
		}
		compound := key + ":" + value
		if _, ok := seen[compound]; ok {
			continue
		}
		seen[compound] = struct{}{}
		out = append(out, Tag{Key: key, Value: value})
	}
	return out
}

// insertDefaultAPIKey intentionally omitted here; API key creation is handled via UI/API in later phases.
