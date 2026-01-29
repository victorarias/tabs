package daemon

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/victorarias/tabs/internal/config"
)

type pushPayload struct {
	SessionID string    `json:"session_id"`
	Tool      string    `json:"tool"`
	Tags      []pushTag `json:"tags"`
}

type pushTag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type uploadRequest struct {
	Session uploadSession `json:"session"`
	Tags    []pushTag     `json:"tags"`
}

type uploadSession struct {
	SessionID string        `json:"session_id"`
	Tool      string        `json:"tool"`
	CreatedAt string        `json:"created_at,omitempty"`
	EndedAt   string        `json:"ended_at,omitempty"`
	Cwd       string        `json:"cwd,omitempty"`
	Events    []uploadEvent `json:"events"`
}

type uploadEvent struct {
	EventType string          `json:"event_type"`
	Timestamp string          `json:"timestamp"`
	Tool      string          `json:"tool"`
	SessionID string          `json:"session_id"`
	Data      json.RawMessage `json:"data"`
}

type pushResult struct {
	RemoteID string `json:"remote_id"`
	URL      string `json:"url"`
}

type pushError struct {
	Code    string
	Message string
}

func (e *pushError) Error() string {
	return e.Message
}

func handlePushSession(baseDir string, payload pushPayload) (pushResult, error) {
	if payload.SessionID == "" || payload.Tool == "" {
		return pushResult{}, &pushError{Code: "invalid_payload", Message: "session_id and tool are required"}
	}
	if payload.Tool != "claude-code" && payload.Tool != "cursor" {
		return pushResult{}, &pushError{Code: "invalid_payload", Message: "tool must be claude-code or cursor"}
	}

	cfgPath, err := config.Path()
	if err != nil {
		return pushResult{}, &pushError{Code: "storage_error", Message: "failed to resolve config path"}
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return pushResult{}, &pushError{Code: "storage_error", Message: "failed to read config"}
		}
		cfg = config.Default()
	}

	if strings.TrimSpace(cfg.Remote.ServerURL) == "" {
		return pushResult{}, &pushError{Code: "invalid_request", Message: "server_url not configured"}
	}
	if strings.TrimSpace(cfg.Remote.APIKey) == "" {
		return pushResult{}, &pushError{Code: "no_api_key", Message: "API key not configured"}
	}

	path, ok, err := findExistingSessionFile(baseDir, payload.SessionID, payload.Tool)
	if err != nil {
		return pushResult{}, &pushError{Code: "storage_error", Message: "failed to locate session file"}
	}
	if !ok || path == "" {
		return pushResult{}, &pushError{Code: "session_not_found", Message: "session not found"}
	}

	events, meta, err := readSessionEvents(path)
	if err != nil {
		return pushResult{}, &pushError{Code: "storage_error", Message: err.Error()}
	}
	if len(events) == 0 {
		return pushResult{}, &pushError{Code: "storage_error", Message: "session contains no events"}
	}

	resolvedTags := mergeTags(cfg.Remote.DefaultTags, payload.Tags)

	req := uploadRequest{
		Session: uploadSession{
			SessionID: payload.SessionID,
			Tool:      payload.Tool,
			CreatedAt: meta.CreatedAt,
			EndedAt:   meta.EndedAt,
			Cwd:       meta.Cwd,
			Events:    events,
		},
		Tags: resolvedTags,
	}

	return pushToRemote(cfg, req)
}

type sessionMeta struct {
	CreatedAt string
	EndedAt   string
	Cwd       string
}

func readSessionEvents(path string) ([]uploadEvent, sessionMeta, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, sessionMeta{}, err
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	var events []uploadEvent
	meta := sessionMeta{}
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
			return nil, sessionMeta{}, err
		}

		trimmed := bytes.TrimSpace(line)
		if len(trimmed) == 0 {
			if errors.Is(err, io.EOF) {
				break
			}
			continue
		}

		var event uploadEvent
		if jsonErr := json.Unmarshal(trimmed, &event); jsonErr != nil {
			return nil, sessionMeta{}, fmt.Errorf("invalid event JSON")
		}
		if event.EventType == "" || event.Timestamp == "" {
			continue
		}
		events = append(events, event)

		if ts, tsErr := time.Parse(time.RFC3339Nano, event.Timestamp); tsErr == nil {
			if earliest.IsZero() || ts.Before(earliest) {
				earliest = ts
			}
			if latest.IsZero() || ts.After(latest) {
				latest = ts
			}
		}

		switch event.EventType {
		case "session_start":
			if meta.Cwd == "" && len(event.Data) > 0 {
				var data struct {
					Cwd string `json:"cwd"`
				}
				_ = json.Unmarshal(event.Data, &data)
				if strings.TrimSpace(data.Cwd) != "" {
					meta.Cwd = strings.TrimSpace(data.Cwd)
				}
			}
		case "session_end":
			if meta.EndedAt == "" && event.Timestamp != "" {
				meta.EndedAt = event.Timestamp
			}
		}

		if errors.Is(err, io.EOF) {
			break
		}
	}

	if meta.CreatedAt == "" && !earliest.IsZero() {
		meta.CreatedAt = earliest.UTC().Format(time.RFC3339Nano)
	}
	if meta.EndedAt == "" && !latest.IsZero() {
		meta.EndedAt = latest.UTC().Format(time.RFC3339Nano)
	}

	return events, meta, nil
}

func mergeTags(defaults []string, tags []pushTag) []pushTag {
	seen := make(map[string]struct{})
	out := make([]pushTag, 0, len(defaults)+len(tags))

	for _, entry := range defaults {
		if tag, ok := parseTagString(entry); ok {
			key := tag.Key + ":" + tag.Value
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, tag)
		}
	}

	for _, tag := range tags {
		key := strings.TrimSpace(tag.Key)
		value := strings.TrimSpace(tag.Value)
		if key == "" || value == "" {
			continue
		}
		compound := key + ":" + value
		if _, exists := seen[compound]; exists {
			continue
		}
		seen[compound] = struct{}{}
		out = append(out, pushTag{Key: key, Value: value})
	}

	return out
}

func parseTagString(raw string) (pushTag, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return pushTag{}, false
	}
	parts := strings.SplitN(trimmed, ":", 2)
	if len(parts) != 2 {
		parts = strings.SplitN(trimmed, "=", 2)
	}
	if len(parts) != 2 {
		return pushTag{}, false
	}
	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])
	if key == "" || value == "" {
		return pushTag{}, false
	}
	return pushTag{Key: key, Value: value}, true
}

func pushToRemote(cfg config.Config, req uploadRequest) (pushResult, error) {
	endpoint := strings.TrimRight(cfg.Remote.ServerURL, "/") + "/api/sessions"
	payload, err := json.Marshal(req)
	if err != nil {
		return pushResult{}, &pushError{Code: "invalid_payload", Message: "failed to encode session"}
	}

	httpReq, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return pushResult{}, &pushError{Code: "network_error", Message: "failed to create request"}
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+strings.TrimSpace(cfg.Remote.APIKey))

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return pushResult{}, &pushError{Code: "network_error", Message: "failed to reach remote server"}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var parsed struct {
			ID  string `json:"id"`
			URL string `json:"url"`
		}
		if err := json.Unmarshal(body, &parsed); err != nil {
			return pushResult{}, &pushError{Code: "storage_error", Message: "invalid response from server"}
		}
		return pushResult{RemoteID: parsed.ID, URL: parsed.URL}, nil
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return pushResult{}, &pushError{Code: "invalid_api_key", Message: "invalid or expired API key"}
	}
	if resp.StatusCode == http.StatusConflict {
		return pushResult{}, &pushError{Code: "duplicate_session", Message: "session already uploaded"}
	}

	var errResp struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &errResp); err == nil {
		if errResp.Error.Code != "" {
			return pushResult{}, &pushError{Code: errResp.Error.Code, Message: errResp.Error.Message}
		}
	}
	return pushResult{}, &pushError{Code: "network_error", Message: "remote server error"}
}
