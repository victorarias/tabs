package daemon

import (
	"time"
)

type State struct {
	start           time.Time
	sessions        map[string]struct{}
	eventsProcessed int
	lastEventAt     time.Time
}

func NewState() *State {
	return &State{
		start:    time.Now().UTC(),
		sessions: make(map[string]struct{}),
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

func (s *State) RecordEvent(sessionID string, ts time.Time) {
	if sessionID == "" {
		return
	}
	s.sessions[sessionID] = struct{}{}
	s.eventsProcessed++
	s.lastEventAt = ts
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
