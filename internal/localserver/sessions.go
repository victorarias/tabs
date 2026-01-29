package localserver

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type SessionFilter struct {
	Tool string
	Date string
	Cwd  string
	Q    string
}

func ListSessions(baseDir string, filter SessionFilter) ([]SessionSummary, error) {
	sessionsDir := filepath.Join(baseDir, "sessions")
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []SessionSummary{}, nil
		}
		return nil, err
	}

	var summaries []SessionSummary
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dayDir := filepath.Join(sessionsDir, entry.Name())
		files, err := os.ReadDir(dayDir)
		if err != nil {
			return nil, err
		}
		for _, file := range files {
			if file.IsDir() || !strings.HasSuffix(file.Name(), ".jsonl") {
				continue
			}
			path := filepath.Join(dayDir, file.Name())
			summary, matched, err := summarizeSession(path, filter)
			if err != nil {
				return nil, err
			}
			if !matched {
				continue
			}
			if filter.Date != "" {
				if summary.CreatedAt != "" {
					if ts, err := time.Parse(time.RFC3339Nano, summary.CreatedAt); err == nil {
						if ts.Format("2006-01-02") != filter.Date {
							continue
						}
					}
				} else if entry.Name() != filter.Date {
					continue
				}
			}
			if filter.Tool != "" && summary.Tool != filter.Tool {
				continue
			}
			if filter.Cwd != "" && !strings.HasPrefix(summary.Cwd, filter.Cwd) {
				continue
			}
			summaries = append(summaries, summary)
		}
	}

	sort.Slice(summaries, func(i, j int) bool {
		return sessionSortTime(summaries[i]).After(sessionSortTime(summaries[j]))
	})

	return summaries, nil
}

func GetSession(baseDir, sessionID string) (SessionDetail, error) {
	if sessionID == "" {
		return SessionDetail{}, errors.New("missing session id")
	}
	path, err := findSessionFile(baseDir, sessionID)
	if err != nil {
		return SessionDetail{}, err
	}
	if path == "" {
		return SessionDetail{}, os.ErrNotExist
	}
	return loadSessionDetail(path)
}

func findSessionFile(baseDir, sessionID string) (string, error) {
	sessionsDir := filepath.Join(baseDir, "sessions")
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	prefix := sessionID + "-"
	var bestPath string
	var bestTS int64 = -1
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dayDir := filepath.Join(sessionsDir, entry.Name())
		files, err := os.ReadDir(dayDir)
		if err != nil {
			return "", err
		}
		for _, file := range files {
			name := file.Name()
			if file.IsDir() || !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, ".jsonl") {
				continue
			}
			ts := extractTimestamp(name)
			if ts > bestTS {
				bestTS = ts
				bestPath = filepath.Join(dayDir, name)
			}
		}
	}
	return bestPath, nil
}

func extractTimestamp(filename string) int64 {
	trimmed := strings.TrimSuffix(filename, ".jsonl")
	parts := strings.Split(trimmed, "-")
	if len(parts) < 3 {
		return -1
	}
	tsPart := parts[len(parts)-1]
	ts, err := parseInt64(tsPart)
	if err != nil {
		return -1
	}
	return ts
}

func parseInt64(value string) (int64, error) {
	var parsed int64
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return 0, errors.New("invalid int")
		}
		parsed = parsed*10 + int64(ch-'0')
	}
	return parsed, nil
}

func summarizeSession(path string, filter SessionFilter) (SessionSummary, bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return SessionSummary{}, false, err
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	summary := SessionSummary{FilePath: path}
	var earliest time.Time
	var latest time.Time
	var hasStart bool
	var overrideCounts bool
	matchedQuery := filter.Q == ""
	query := strings.ToLower(filter.Q)

	for {
		line, err := reader.ReadBytes('\n')
		if len(line) == 0 && errors.Is(err, io.EOF) {
			break
		}
		if errors.Is(err, io.EOF) && len(line) > 0 && !bytes.HasSuffix(line, []byte{'\n'}) {
			break
		}
		if err != nil && !errors.Is(err, io.EOF) {
			return SessionSummary{}, false, err
		}

		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			if errors.Is(err, io.EOF) {
				break
			}
			continue
		}

		var event map[string]interface{}
		if jsonErr := json.Unmarshal(line, &event); jsonErr != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			continue
		}

		if summary.SessionID == "" {
			if value, ok := event["session_id"].(string); ok {
				summary.SessionID = value
			}
		}
		if summary.Tool == "" {
			if value, ok := event["tool"].(string); ok {
				summary.Tool = value
			}
		}

		ts := parseEventTime(event)
		if !ts.IsZero() {
			if earliest.IsZero() || ts.Before(earliest) {
				earliest = ts
			}
			if latest.IsZero() || ts.After(latest) {
				latest = ts
			}
		}

		eventType, _ := event["event_type"].(string)
		switch eventType {
		case "session_start":
			if !ts.IsZero() && !hasStart {
				summary.CreatedAt = ts.UTC().Format(time.RFC3339Nano)
				hasStart = true
			}
			if data, ok := event["data"].(map[string]interface{}); ok {
				if cwd, ok := data["cwd"].(string); ok && cwd != "" {
					summary.Cwd = cwd
				}
			}
		case "session_end":
			if !ts.IsZero() {
				summary.EndedAt = ts.UTC().Format(time.RFC3339Nano)
			}
			if data, ok := event["data"].(map[string]interface{}); ok {
				if value, ok := toInt(data["duration_seconds"]); ok {
					summary.DurationSeconds = value
				}
				if value, ok := toInt(data["message_count"]); ok {
					summary.MessageCount = value
					overrideCounts = true
				}
				if value, ok := toInt(data["tool_use_count"]); ok {
					summary.ToolUseCount = value
					overrideCounts = true
				}
			}
		case "message":
			if !overrideCounts {
				summary.MessageCount++
			}
			if !matchedQuery && query != "" {
				if matchesQuery(event, query) {
					matchedQuery = true
				}
			}
		case "tool_use":
			if !overrideCounts {
				summary.ToolUseCount++
			}
		default:
			if !matchedQuery && query != "" {
				if matchesQuery(event, query) {
					matchedQuery = true
				}
			}
		}

		if errors.Is(err, io.EOF) {
			break
		}
	}

	if summary.CreatedAt == "" && !earliest.IsZero() {
		summary.CreatedAt = earliest.UTC().Format(time.RFC3339Nano)
	}
	if summary.EndedAt == "" && !latest.IsZero() {
		summary.EndedAt = latest.UTC().Format(time.RFC3339Nano)
	}
	if summary.DurationSeconds == 0 && !earliest.IsZero() && !latest.IsZero() && latest.After(earliest) {
		summary.DurationSeconds = int(latest.Sub(earliest).Seconds())
	}

	if filter.Q != "" && !matchedQuery {
		return summary, false, nil
	}
	return summary, true, nil
}

func loadSessionDetail(path string) (SessionDetail, error) {
	file, err := os.Open(path)
	if err != nil {
		return SessionDetail{}, err
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	detail := SessionDetail{}
	var earliest time.Time
	var latest time.Time

	for {
		line, err := reader.ReadBytes('\n')
		if len(line) == 0 && errors.Is(err, io.EOF) {
			break
		}
		if errors.Is(err, io.EOF) && len(line) > 0 && !bytes.HasSuffix(line, []byte{'\n'}) {
			break
		}
		if err != nil && !errors.Is(err, io.EOF) {
			return SessionDetail{}, err
		}

		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			if errors.Is(err, io.EOF) {
				break
			}
			continue
		}

		var event map[string]interface{}
		if jsonErr := json.Unmarshal(line, &event); jsonErr != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			continue
		}

		if detail.SessionID == "" {
			if value, ok := event["session_id"].(string); ok {
				detail.SessionID = value
			}
		}
		if detail.Tool == "" {
			if value, ok := event["tool"].(string); ok {
				detail.Tool = value
			}
		}

		ts := parseEventTime(event)
		if !ts.IsZero() {
			if earliest.IsZero() || ts.Before(earliest) {
				earliest = ts
			}
			if latest.IsZero() || ts.After(latest) {
				latest = ts
			}
		}

		if eventType, ok := event["event_type"].(string); ok {
			switch eventType {
			case "session_start":
				if data, ok := event["data"].(map[string]interface{}); ok {
					if cwd, ok := data["cwd"].(string); ok && cwd != "" {
						detail.Cwd = cwd
					}
				}
				if !ts.IsZero() {
					detail.CreatedAt = ts.UTC().Format(time.RFC3339Nano)
				}
			case "session_end":
				if !ts.IsZero() {
					detail.EndedAt = ts.UTC().Format(time.RFC3339Nano)
				}
			}
		}

		detail.Events = append(detail.Events, event)

		if errors.Is(err, io.EOF) {
			break
		}
	}

	if detail.CreatedAt == "" && !earliest.IsZero() {
		detail.CreatedAt = earliest.UTC().Format(time.RFC3339Nano)
	}
	if detail.EndedAt == "" && !latest.IsZero() {
		detail.EndedAt = latest.UTC().Format(time.RFC3339Nano)
	}

	return detail, nil
}

func parseEventTime(event map[string]interface{}) time.Time {
	value, ok := event["timestamp"].(string)
	if !ok {
		return time.Time{}
	}
	if ts, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return ts
	}
	if ts, err := time.Parse(time.RFC3339, value); err == nil {
		return ts
	}
	return time.Time{}
}

func sessionSortTime(summary SessionSummary) time.Time {
	if summary.CreatedAt != "" {
		if ts, err := time.Parse(time.RFC3339Nano, summary.CreatedAt); err == nil {
			return ts
		}
	}
	if summary.EndedAt != "" {
		if ts, err := time.Parse(time.RFC3339Nano, summary.EndedAt); err == nil {
			return ts
		}
	}
	if ts, err := time.Parse(time.RFC3339, summary.CreatedAt); err == nil {
		return ts
	}
	return time.Time{}
}

func matchesQuery(event map[string]interface{}, query string) bool {
	data, ok := event["data"].(map[string]interface{})
	if !ok {
		return false
	}
	if content, ok := data["content"]; ok {
		switch v := content.(type) {
		case string:
			return strings.Contains(strings.ToLower(v), query)
		case []interface{}:
			for _, item := range v {
				if m, ok := item.(map[string]interface{}); ok {
					if text, ok := m["text"].(string); ok {
						if strings.Contains(strings.ToLower(text), query) {
							return true
						}
					}
				}
			}
		}
	}
	if text, ok := data["text"].(string); ok {
		return strings.Contains(strings.ToLower(text), query)
	}
	return false
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
		parsed, err := parseInt64(string(v))
		if err != nil {
			return 0, false
		}
		return int(parsed), true
	case string:
		parsed, err := parseInt64(v)
		if err != nil {
			return 0, false
		}
		return int(parsed), true
	default:
		return 0, false
	}
}
