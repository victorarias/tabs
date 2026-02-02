package daemon

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"
)

func (s *Server) captureClaude(req capturePayload, sessionID string, hookTime time.Time) (int, time.Time, error) {
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

	transcriptPath := extractTranscriptPath(req.Event)
	if transcriptPath == "" {
		transcriptPath = cursor.TranscriptPath
	}
	if transcriptPath == "" {
		return 0, time.Time{}, errors.New("missing transcript_path")
	}
	cursor.TranscriptPath = transcriptPath

	sessionPath, err := s.state.EnsureSessionFile(s.baseDir, sessionID, req.Tool, hookTime)
	if err != nil {
		return 0, time.Time{}, err
	}

	eventsWritten := 0
	lastEventTime := time.Time{}

	if needsSessionStart(cursor) {
		startEvent := buildSessionStartEvent(req.Event, sessionID, req.Tool, hookTime)
		if startEvent != nil {
			wroteAt, err := s.appendEvent(sessionPath, cursor, startEvent)
			if err != nil {
				return 0, time.Time{}, err
			}
			eventsWritten++
			lastEventTime = maxTime(lastEventTime, wroteAt)
		}
	}

	written, latest, newOffset, lastHash, err := s.appendClaudeTranscript(sessionPath, sessionID, cursor, hookTime)
	if err != nil {
		return 0, time.Time{}, err
	}
	eventsWritten += written
	lastEventTime = maxTime(lastEventTime, latest)
	if newOffset >= 0 {
		cursor.LastOffset = newOffset
	}
	if lastHash != "" {
		cursor.LastLineHash = lastHash
	}

	if endEvent := buildSessionEndEvent(req.Event, sessionID, req.Tool, hookTime, cursor); endEvent != nil {
		wroteAt, err := s.appendEvent(sessionPath, cursor, endEvent)
		if err != nil {
			return 0, time.Time{}, err
		}
		eventsWritten++
		lastEventTime = maxTime(lastEventTime, wroteAt)
	}

	if err := saveCursorState(s.baseDir, cursor); err != nil {
		return 0, time.Time{}, err
	}

	return eventsWritten, lastEventTime, nil
}

func needsSessionStart(cursor *SessionCursor) bool {
	if cursor == nil || cursor.Metadata == nil {
		return true
	}
	return cursor.Metadata.CreatedAt == ""
}

func buildSessionStartEvent(event map[string]interface{}, sessionID, tool string, hookTime time.Time) map[string]interface{} {
	data := map[string]interface{}{}
	if value, ok := event["cwd"].(string); ok && value != "" {
		data["cwd"] = value
	}
	if value, ok := event["permission_mode"].(string); ok && value != "" {
		data["permission_mode"] = value
	}
	if value, ok := event["model"].(string); ok && value != "" {
		data["model"] = value
	}
	if len(data) == 0 {
		data["metadata"] = map[string]interface{}{}
	}
	return buildEvent("session_start", sessionID, tool, hookTimestamp(event, hookTime), data)
}

func buildSessionEndEvent(event map[string]interface{}, sessionID, tool string, hookTime time.Time, cursor *SessionCursor) map[string]interface{} {
	if cursor != nil && cursor.Metadata != nil && cursor.Metadata.EndedAt != "" {
		return nil
	}
	data := map[string]interface{}{}
	if fileContext := extractFileContext(event); fileContext != nil {
		data["file_context"] = fileContext
	}
	if value, ok := toInt(event["duration_seconds"]); ok {
		data["duration_seconds"] = value
	}
	if value, ok := toInt(event["message_count"]); ok {
		data["message_count"] = value
	}
	if value, ok := toInt(event["tool_use_count"]); ok {
		data["tool_use_count"] = value
	}
	if len(data) == 0 {
		return nil
	}
	return buildEvent("session_end", sessionID, tool, hookTimestamp(event, hookTime), data)
}

func extractFileContext(event map[string]interface{}) map[string]interface{} {
	if ctx, ok := event["file_context"].(map[string]interface{}); ok {
		return ctx
	}
	if ctx, ok := event["fileContext"].(map[string]interface{}); ok {
		return ctx
	}
	return nil
}

func hookTimestamp(event map[string]interface{}, fallback time.Time) time.Time {
	if value, ok := event["timestamp"].(string); ok {
		if ts, err := time.Parse(time.RFC3339Nano, value); err == nil {
			return ts
		}
		if ts, err := time.Parse(time.RFC3339, value); err == nil {
			return ts
		}
	}
	return fallback
}

func (s *Server) appendClaudeTranscript(sessionPath, sessionID string, cursor *SessionCursor, hookTime time.Time) (int, time.Time, int64, string, error) {
	if cursor == nil || cursor.TranscriptPath == "" {
		return 0, time.Time{}, cursor.LastOffset, cursor.LastLineHash, nil
	}

	file, err := os.Open(cursor.TranscriptPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Transcript file doesn't exist yet (e.g., during SessionStart)
			// Return success - cursor will be saved with transcript_path for later
			return 0, time.Time{}, cursor.LastOffset, cursor.LastLineHash, nil
		}
		return 0, time.Time{}, cursor.LastOffset, cursor.LastLineHash, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return 0, time.Time{}, cursor.LastOffset, cursor.LastLineHash, err
	}

	offset := cursor.LastOffset
	lastHash := cursor.LastLineHash
	if offset > info.Size() {
		offset = 0
		lastHash = ""
	}
	if offset > 0 {
		if _, err := file.Seek(offset, io.SeekStart); err != nil {
			return 0, time.Time{}, cursor.LastOffset, cursor.LastLineHash, err
		}
	}

	reader := bufio.NewReader(file)
	eventsWritten := 0
	lastEventTime := time.Time{}
	for {
		line, err := reader.ReadBytes('\n')
		if len(line) == 0 && errors.Is(err, io.EOF) {
			break
		}
		if errors.Is(err, io.EOF) && len(line) > 0 && !bytes.HasSuffix(line, []byte{'\n'}) {
			break
		}
		if err != nil && !errors.Is(err, io.EOF) {
			return eventsWritten, lastEventTime, offset, lastHash, err
		}

		if len(line) == 0 {
			if errors.Is(err, io.EOF) {
				break
			}
			continue
		}
		offset += int64(len(line))
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) == 0 {
			if errors.Is(err, io.EOF) {
				break
			}
			continue
		}

		lineHash := hashLine(trimmed)
		lastHash = lineHash
		if lineHash == cursor.LastLineHash {
			if errors.Is(err, io.EOF) {
				break
			}
			continue
		}

		events, lineTime, parseErr := claudeEventsFromLine(trimmed, sessionID, hookTime)
		if parseErr != nil {
			s.logger.Warn("failed to parse transcript line", "session_id", sessionID, "error", parseErr)
			if errors.Is(err, io.EOF) {
				break
			}
			continue
		}
		lastEventTime = maxTime(lastEventTime, lineTime)

		for _, event := range events {
			wroteAt, err := s.appendEvent(sessionPath, cursor, event)
			if err != nil {
				return eventsWritten, lastEventTime, offset, lastHash, err
			}
			eventsWritten++
			lastEventTime = maxTime(lastEventTime, wroteAt)
		}
		if errors.Is(err, io.EOF) {
			break
		}
	}

	return eventsWritten, lastEventTime, offset, lastHash, nil
}

func claudeEventsFromLine(line []byte, sessionID string, fallback time.Time) ([]map[string]interface{}, time.Time, error) {
	var record map[string]interface{}
	if err := json.Unmarshal(line, &record); err != nil {
		return nil, time.Time{}, err
	}

	ts := hookTimestamp(record, fallback)
	events := []map[string]interface{}{}

	if role := claudeRole(record); role != "" {
		content := normalizeContent(extractMessageContent(record))
		if len(content) > 0 {
			data := map[string]interface{}{
				"role":    role,
				"content": content,
			}
			if role == "assistant" {
				if model := extractMessageModel(record); model != "" {
					data["model"] = model
				}
			}
			events = append(events, buildEvent("message", sessionID, "claude-code", ts, data))
		}
	}

	for _, toolUse := range extractToolUse(record) {
		events = append(events, buildEvent("tool_use", sessionID, "claude-code", ts, toolUse))
	}

	for _, toolResult := range extractToolResult(record) {
		events = append(events, buildEvent("tool_result", sessionID, "claude-code", ts, toolResult))
	}

	return events, ts, nil
}

func claudeRole(record map[string]interface{}) string {
	if value, ok := record["type"].(string); ok {
		switch value {
		case "user", "assistant":
			return value
		}
	}
	if value, ok := record["role"].(string); ok {
		switch value {
		case "user", "assistant":
			return value
		}
	}
	return ""
}

func extractMessageContent(record map[string]interface{}) interface{} {
	// Try nested message.content first (actual Claude Code transcript format)
	if message, ok := record["message"].(map[string]interface{}); ok {
		if content := message["content"]; content != nil {
			return content
		}
	}
	// Fallback to top-level content
	return record["content"]
}

func extractMessageModel(record map[string]interface{}) string {
	// Try nested message.model first
	if message, ok := record["message"].(map[string]interface{}); ok {
		if model, ok := message["model"].(string); ok && model != "" {
			return model
		}
	}
	// Fallback to top-level model
	if model, ok := record["model"].(string); ok {
		return model
	}
	return ""
}

func normalizeContent(raw interface{}) []map[string]interface{} {
	switch value := raw.(type) {
	case []interface{}:
		content := make([]map[string]interface{}, 0, len(value))
		for _, item := range value {
			switch part := item.(type) {
			case map[string]interface{}:
				partType, _ := part["type"].(string)
				if partType == "" {
					continue
				}
				// Extract text based on content type
				var partText string
				switch partType {
				case "text":
					partText, _ = part["text"].(string)
				case "thinking":
					partText, _ = part["thinking"].(string)
				default:
					// Skip tool_use and other types (handled separately)
					continue
				}
				if partText == "" {
					continue
				}
				content = append(content, map[string]interface{}{
					"type": partType,
					"text": partText,
				})
			case string:
				if part != "" {
					content = append(content, map[string]interface{}{
						"type": "text",
						"text": part,
					})
				}
			}
		}
		return content
	case string:
		if value == "" {
			return nil
		}
		return []map[string]interface{}{{"type": "text", "text": value}}
	default:
		return nil
	}
}

func extractToolUse(record map[string]interface{}) []map[string]interface{} {
	content := extractMessageContent(record)
	contentArray, ok := content.([]interface{})
	if !ok {
		return nil
	}

	var toolUses []map[string]interface{}
	for _, item := range contentArray {
		block, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		blockType, _ := block["type"].(string)
		if blockType != "tool_use" {
			continue
		}

		toolUseID := toStringValue(block["id"])
		if toolUseID == "" {
			toolUseID = toStringValue(block["tool_use_id"])
		}
		toolName := toStringValue(block["name"])
		if toolName == "" {
			toolName = toStringValue(block["tool_name"])
		}
		input, _ := block["input"].(map[string]interface{})
		if toolUseID == "" || toolName == "" {
			continue
		}
		toolUses = append(toolUses, map[string]interface{}{
			"tool_use_id": toolUseID,
			"tool_name":   toolName,
			"input":       input,
		})
	}
	return toolUses
}

func extractToolResult(record map[string]interface{}) []map[string]interface{} {
	content := extractMessageContent(record)
	contentArray, ok := content.([]interface{})
	if !ok {
		return nil
	}

	var toolResults []map[string]interface{}
	for _, item := range contentArray {
		block, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		blockType, _ := block["type"].(string)
		if blockType != "tool_result" {
			continue
		}

		toolUseID := toStringValue(block["tool_use_id"])
		if toolUseID == "" {
			toolUseID = toStringValue(block["id"])
		}
		resultContent := block["content"]
		isError, _ := block["is_error"].(bool)
		if toolUseID == "" {
			continue
		}
		toolResults = append(toolResults, map[string]interface{}{
			"tool_use_id": toolUseID,
			"content":     resultContent,
			"is_error":    isError,
		})
	}
	return toolResults
}

func toStringValue(value interface{}) string {
	if value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return text
	}
	return fmt.Sprint(value)
}

func buildEvent(eventType, sessionID, tool string, ts time.Time, data map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"event_type": eventType,
		"timestamp":  ts.UTC().Format(time.RFC3339Nano),
		"tool":       tool,
		"session_id": sessionID,
		"data":       data,
	}
}

func (s *Server) appendEvent(sessionPath string, cursor *SessionCursor, event map[string]interface{}) (time.Time, error) {
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return time.Time{}, err
	}
	if _, err := appendJSONL(sessionPath, eventJSON); err != nil {
		return time.Time{}, err
	}
	meta := extractEventMetadata(event)
	updateCursorMetadata(cursor, meta, sessionPath)
	return meta.Timestamp, nil
}

func maxTime(a, b time.Time) time.Time {
	if b.After(a) {
		return b
	}
	return a
}
