package server

import (
	"encoding/json"
	"time"
)

type UploadRequest struct {
	Session UploadSession `json:"session"`
	Tags    []Tag         `json:"tags"`
}

type UploadSession struct {
	SessionID string  `json:"session_id"`
	Tool      string  `json:"tool"`
	CreatedAt string  `json:"created_at"`
	EndedAt   string  `json:"ended_at"`
	Cwd       string  `json:"cwd"`
	Events    []Event `json:"events"`
}

type Event struct {
	EventType string          `json:"event_type"`
	Timestamp string          `json:"timestamp"`
	Tool      string          `json:"tool"`
	SessionID string          `json:"session_id"`
	Data      json.RawMessage `json:"data"`
}

type Tag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type NormalizedSession struct {
	Tool            string
	SessionID       string
	CreatedAt       time.Time
	EndedAt         *time.Time
	Cwd             string
	DurationSeconds *int
	MessageCount    int
	ToolUseCount    int
	Messages        []MessageRecord
	Tools           []ToolRecord
	Tags            []Tag
}

type MessageRecord struct {
	Timestamp time.Time
	Seq       int
	Role      string
	Model     *string
	Content   json.RawMessage
}

type ToolRecord struct {
	Timestamp time.Time
	ToolUseID string
	ToolName  string
	Input     json.RawMessage
	Output    json.RawMessage
	IsError   bool
}

type ErrorResponse struct {
	Error ErrorPayload `json:"error"`
}

type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
