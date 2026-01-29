package daemon

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/victorarias/tabs/internal/config"
	_ "modernc.org/sqlite"
)

type cursorMessage struct {
	Role      string
	Content   string
	Timestamp string
}

type cursorConversation struct {
	ID             string
	Messages       []cursorMessage
	WorkspaceRoots []string
}

func StartCursorPoller(ctx context.Context, srv *Server, cfg config.Config) {
	if strings.TrimSpace(cfg.Cursor.DBPath) == "" {
		return
	}
	interval := time.Duration(cfg.Cursor.PollInterval) * time.Second
	if interval <= 0 {
		interval = 2 * time.Second
	}

	srv.mu.Lock()
	srv.state.SetCursorPolling(true)
	srv.mu.Unlock()
	go func() {
		defer func() {
			srv.mu.Lock()
			srv.state.SetCursorPolling(false)
			srv.mu.Unlock()
		}()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := srv.pollCursorDB(cfg.Cursor.DBPath); err != nil {
					srv.logger.Warn("cursor poll error", "error", err)
				}
			}
		}
	}()
}

func (s *Server) captureCursor(req capturePayload, sessionID string, hookTime time.Time) (int, time.Time, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cursor, err := loadCursorState(s.baseDir, sessionID)
	if err != nil {
		s.logger.Warn("cursor state load failed", "session_id", sessionID, "error", err)
		return 0, time.Time{}, err
	}
	if cursor == nil {
		return 0, time.Time{}, errors.New("failed to read cursor state")
	}

	sessionPath, err := s.state.EnsureSessionFile(s.baseDir, sessionID, req.Tool, hookTime)
	if err != nil {
		return 0, time.Time{}, err
	}

	eventsWritten := 0
	lastEventTime := time.Time{}
	hookEvent := extractCursorHookEvent(req.Event)

	if hookEvent == "beforeSubmitPrompt" {
		if needsSessionStart(cursor) {
			start := buildCursorSessionStart(req.Event, sessionID, hookTime)
			if start != nil {
				wroteAt, err := s.appendEvent(sessionPath, cursor, start)
				if err != nil {
					return eventsWritten, lastEventTime, err
				}
				eventsWritten++
				lastEventTime = maxTime(lastEventTime, wroteAt)
			}
		}

		if prompt, ok := req.Event["prompt"].(string); ok && strings.TrimSpace(prompt) != "" {
			msg := buildCursorMessage(sessionID, hookTime, "user", prompt)
			wroteAt, err := s.appendEvent(sessionPath, cursor, msg)
			if err != nil {
				return eventsWritten, lastEventTime, err
			}
			eventsWritten++
			lastEventTime = maxTime(lastEventTime, wroteAt)
		}
	}

	if hookEvent == "stop" {
		if cursor.Metadata == nil || cursor.Metadata.EndedAt == "" {
			end := buildCursorSessionEnd(req.Event, sessionID, hookTime)
			if end != nil {
				wroteAt, err := s.appendEvent(sessionPath, cursor, end)
				if err != nil {
					return eventsWritten, lastEventTime, err
				}
				eventsWritten++
				lastEventTime = maxTime(lastEventTime, wroteAt)
			}
		}
	}

	if err := saveCursorState(s.baseDir, cursor); err != nil {
		return eventsWritten, lastEventTime, err
	}

	return eventsWritten, lastEventTime, nil
}

func extractCursorHookEvent(event map[string]interface{}) string {
	if value, ok := event["hook_event_name"].(string); ok {
		return value
	}
	if value, ok := event["event"].(string); ok {
		return value
	}
	return ""
}

func buildCursorSessionStart(event map[string]interface{}, sessionID string, hookTime time.Time) map[string]interface{} {
	data := map[string]interface{}{}
	if roots, ok := event["workspace_roots"].([]interface{}); ok && len(roots) > 0 {
		if cwd, ok := roots[0].(string); ok && cwd != "" {
			data["cwd"] = cwd
		}
	}
	if len(data) == 0 {
		data["metadata"] = map[string]interface{}{}
	}
	return buildEvent("session_start", sessionID, "cursor", hookTimestamp(event, hookTime), data)
}

func buildCursorSessionEnd(event map[string]interface{}, sessionID string, hookTime time.Time) map[string]interface{} {
	data := map[string]interface{}{}
	if value, ok := event["generation_id"].(string); ok && value != "" {
		data["generation_id"] = value
	}
	if len(data) == 0 {
		data["metadata"] = map[string]interface{}{}
	}
	return buildEvent("session_end", sessionID, "cursor", hookTimestamp(event, hookTime), data)
}

func buildCursorMessage(sessionID string, ts time.Time, role, content string) map[string]interface{} {
	data := map[string]interface{}{
		"role": role,
		"content": []map[string]interface{}{{
			"type": "text",
			"text": content,
		}},
	}
	return buildEvent("message", sessionID, "cursor", ts, data)
}

func (s *Server) pollCursorDB(path string) error {
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=ro", path))
	if err != nil {
		return fmt.Errorf("open cursor db: %w", err)
	}
	defer db.Close()

	rows, err := db.Query(`SELECT value FROM ItemTable WHERE [key] = 'workbench.panel.aichat.view.aichat.chatdata'`)
	if err != nil {
		return fmt.Errorf("query cursor db: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			continue
		}
		conv, ok := parseCursorConversation(raw)
		if !ok || conv.ID == "" {
			continue
		}
		s.processCursorConversation(conv)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("scan cursor db: %w", err)
	}
	return nil
}

func (s *Server) processCursorConversation(conv cursorConversation) {
	if conv.ID == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	cursor, err := loadCursorState(s.baseDir, conv.ID)
	if err != nil {
		s.logger.Warn("cursor state load failed", "session_id", conv.ID, "error", err)
		return
	}
	if cursor == nil {
		return
	}

	if len(conv.Messages) == 0 {
		return
	}

	if needsSessionStart(cursor) {
		data := map[string]interface{}{}
		if len(conv.WorkspaceRoots) > 0 && conv.WorkspaceRoots[0] != "" {
			data["cwd"] = conv.WorkspaceRoots[0]
		} else {
			data["metadata"] = map[string]interface{}{}
		}
		start := buildEvent("session_start", conv.ID, "cursor", time.Now().UTC(), data)
		if sessionPath, err := s.state.EnsureSessionFile(s.baseDir, conv.ID, "cursor", time.Now().UTC()); err == nil {
			if _, err := s.appendEvent(sessionPath, cursor, start); err == nil {
				s.state.RecordEvent(conv.ID, time.Now().UTC(), 1)
			}
		}
	}

	messageCount := 0
	if cursor.Metadata != nil {
		messageCount = cursor.Metadata.MessageCount
	}
	if messageCount < 0 {
		messageCount = 0
	}
	if messageCount >= len(conv.Messages) {
		return
	}

	sessionPath, err := s.state.EnsureSessionFile(s.baseDir, conv.ID, "cursor", time.Now().UTC())
	if err != nil {
		return
	}

	lastEventTime := time.Time{}
	written := 0
	for i := messageCount; i < len(conv.Messages); i++ {
		msg := conv.Messages[i]
		if msg.Role == "" || msg.Content == "" {
			continue
		}
		ts := parseCursorTimestamp(msg.Timestamp)
		event := buildCursorMessage(conv.ID, ts, msg.Role, msg.Content)
		wroteAt, err := s.appendEvent(sessionPath, cursor, event)
		if err != nil {
			break
		}
		written++
		lastEventTime = maxTime(lastEventTime, wroteAt)
	}

	if written > 0 {
		_ = saveCursorState(s.baseDir, cursor)
		s.state.RecordEvent(conv.ID, lastEventTime, written)
	}
}

func parseCursorTimestamp(raw string) time.Time {
	if raw == "" {
		return time.Now().UTC()
	}
	if ts, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return ts
	}
	if ts, err := time.Parse(time.RFC3339, raw); err == nil {
		return ts
	}
	return time.Now().UTC()
}

func parseCursorConversation(raw []byte) (cursorConversation, bool) {
	var payload map[string]interface{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return cursorConversation{}, false
	}
	id := getString(payload, "conversation_id", "conversationId")
	messagesRaw, ok := payload["messages"].([]interface{})
	if !ok {
		return cursorConversation{}, false
	}
	messages := make([]cursorMessage, 0, len(messagesRaw))
	for _, item := range messagesRaw {
		msgMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		role := getString(msgMap, "role")
		content := getString(msgMap, "content")
		if content == "" {
			content = fmt.Sprint(msgMap["content"])
		}
		ts := getString(msgMap, "timestamp", "time", "created_at", "createdAt")
		messages = append(messages, cursorMessage{Role: role, Content: content, Timestamp: ts})
	}

	roots := []string{}
	if rawRoots, ok := payload["workspace_roots"].([]interface{}); ok {
		for _, entry := range rawRoots {
			if text, ok := entry.(string); ok && text != "" {
				roots = append(roots, text)
			}
		}
	}
	if len(roots) == 0 {
		if rawRoots, ok := payload["workspaceRoots"].([]interface{}); ok {
			for _, entry := range rawRoots {
				if text, ok := entry.(string); ok && text != "" {
					roots = append(roots, text)
				}
			}
		}
	}

	return cursorConversation{ID: id, Messages: messages, WorkspaceRoots: roots}, true
}

func getString(values map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if value, ok := values[key]; ok {
			switch v := value.(type) {
			case string:
				return v
			case fmt.Stringer:
				return v.String()
			}
		}
	}
	return ""
}
