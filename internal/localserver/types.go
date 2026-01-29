package localserver

type SessionSummary struct {
	SessionID       string `json:"session_id"`
	Tool            string `json:"tool"`
	CreatedAt       string `json:"created_at"`
	EndedAt         string `json:"ended_at,omitempty"`
	Cwd             string `json:"cwd,omitempty"`
	Summary         string `json:"summary,omitempty"`
	DurationSeconds int    `json:"duration_seconds,omitempty"`
	MessageCount    int    `json:"message_count"`
	ToolUseCount    int    `json:"tool_use_count"`
	FilePath        string `json:"file_path"`
}

type SessionDetail struct {
	SessionID       string                   `json:"session_id"`
	Tool            string                   `json:"tool"`
	CreatedAt       string                   `json:"created_at"`
	EndedAt         string                   `json:"ended_at,omitempty"`
	Cwd             string                   `json:"cwd,omitempty"`
	DurationSeconds int                      `json:"duration_seconds,omitempty"`
	Events          []map[string]interface{} `json:"events"`
}

type SessionsResponse struct {
	Sessions []SessionSummary `json:"sessions"`
	Total    int              `json:"total"`
}

type ErrorResponse struct {
	Status string       `json:"status"`
	Error  ErrorPayload `json:"error"`
}

type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
