package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type State struct {
	start           time.Time
	sessions        map[string]struct{}
	eventsProcessed int
	lastEventAt     time.Time
	sessionFiles    map[string]string
}

func NewState() *State {
	return &State{
		start:        time.Now().UTC(),
		sessions:     make(map[string]struct{}),
		sessionFiles: make(map[string]string),
	}
}

type Status struct {
	PID              int    `json:"pid"`
	UptimeSeconds    int    `json:"uptime_seconds"`
	SessionsCaptured int    `json:"sessions_captured"`
	EventsProcessed  int    `json:"events_processed"`
	CursorPolling    bool   `json:"cursor_polling"`
	LastEventAt      string `json:"last_event_at"`
}

func (s *State) RecordEvent(sessionID string, ts time.Time, eventsWritten int) {
	if sessionID == "" || eventsWritten <= 0 {
		return
	}
	s.sessions[sessionID] = struct{}{}
	s.eventsProcessed += eventsWritten
	if ts.After(s.lastEventAt) {
		s.lastEventAt = ts
	}
}

func (s *State) Snapshot(pid int) Status {
	status := Status{
		PID:              pid,
		UptimeSeconds:    int(time.Since(s.start).Seconds()),
		SessionsCaptured: len(s.sessions),
		EventsProcessed:  s.eventsProcessed,
		CursorPolling:    false,
	}
	if !s.lastEventAt.IsZero() {
		status.LastEventAt = s.lastEventAt.UTC().Format(time.RFC3339Nano)
	}
	return status
}

func (s *State) EnsureSessionFile(baseDir, sessionID, tool string, eventTime time.Time) (string, error) {
	if sessionID == "" || tool == "" {
		return "", fmt.Errorf("invalid session or tool")
	}
	key := sessionKey(sessionID, tool)
	if path, ok := s.sessionFiles[key]; ok {
		return path, nil
	}
	if existing, ok, err := findExistingSessionFile(baseDir, sessionID, tool); err != nil {
		return "", err
	} else if ok {
		s.sessionFiles[key] = existing
		return existing, nil
	}
	dateDir := filepath.Join(SessionsDir(baseDir), eventTime.UTC().Format("2006-01-02"))
	if err := os.MkdirAll(dateDir, 0o700); err != nil {
		return "", err
	}
	filename := fmt.Sprintf("%s-%s-%d.jsonl", sessionID, tool, eventTime.Unix())
	path := filepath.Join(dateDir, filename)
	s.sessionFiles[key] = path
	return path, nil
}

func sessionKey(sessionID, tool string) string {
	return sessionID + "|" + tool
}

func findExistingSessionFile(baseDir, sessionID, tool string) (string, bool, error) {
	sessionsDir := SessionsDir(baseDir)
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, err
	}
	prefix := sessionID + "-" + tool + "-"
	var bestPath string
	var bestTs int64 = -1
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dayDir := filepath.Join(sessionsDir, entry.Name())
		files, err := os.ReadDir(dayDir)
		if err != nil {
			return "", false, err
		}
		for _, file := range files {
			name := file.Name()
			if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, ".jsonl") {
				continue
			}
			tsPart := strings.TrimSuffix(strings.TrimPrefix(name, prefix), ".jsonl")
			ts, err := strconv.ParseInt(tsPart, 10, 64)
			if err != nil {
				continue
			}
			if ts > bestTs {
				bestTs = ts
				bestPath = filepath.Join(dayDir, name)
			}
		}
	}
	if bestPath == "" {
		return "", false, nil
	}
	return bestPath, true, nil
}
