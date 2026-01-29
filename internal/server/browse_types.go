package server

import (
	"encoding/json"
	"time"
)

type SessionFilter struct {
	Tool       string
	UploadedBy string
	Tags       []Tag
	Query      string
	Page       int
	Limit      int
	Sort       string
	Order      string
}

type SessionsResponse struct {
	Sessions   []SessionSummary `json:"sessions"`
	Total      int              `json:"total"`
	Pagination *Pagination      `json:"pagination,omitempty"`
}

type Pagination struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type SessionSummary struct {
	ID              string     `json:"id"`
	Tool            string     `json:"tool"`
	SessionID       string     `json:"session_id"`
	CreatedAt       time.Time  `json:"created_at"`
	EndedAt         *time.Time `json:"ended_at,omitempty"`
	Cwd             string     `json:"cwd"`
	UploadedBy      string     `json:"uploaded_by"`
	UploadedAt      time.Time  `json:"uploaded_at"`
	DurationSeconds *int       `json:"duration_seconds,omitempty"`
	MessageCount    int        `json:"message_count"`
	ToolUseCount    int        `json:"tool_use_count"`
	Tags            []Tag      `json:"tags"`
	Summary         string     `json:"summary,omitempty"`
}

type SessionDetail struct {
	ID              string          `json:"id"`
	Tool            string          `json:"tool"`
	SessionID       string          `json:"session_id"`
	CreatedAt       time.Time       `json:"created_at"`
	EndedAt         *time.Time      `json:"ended_at,omitempty"`
	Cwd             string          `json:"cwd"`
	UploadedBy      string          `json:"uploaded_by"`
	UploadedAt      time.Time       `json:"uploaded_at"`
	DurationSeconds *int            `json:"duration_seconds,omitempty"`
	MessageCount    int             `json:"message_count"`
	ToolUseCount    int             `json:"tool_use_count"`
	Tags            []Tag           `json:"tags"`
	Messages        []MessageDetail `json:"messages"`
	Tools           []ToolDetail    `json:"tools"`
}

type MessageDetail struct {
	ID        string          `json:"id"`
	Timestamp time.Time       `json:"timestamp"`
	Seq       int             `json:"seq"`
	Role      string          `json:"role"`
	Model     *string         `json:"model,omitempty"`
	Content   json.RawMessage `json:"content"`
}

type ToolDetail struct {
	ID        string          `json:"id"`
	Timestamp time.Time       `json:"timestamp"`
	ToolUseID string          `json:"tool_use_id"`
	ToolName  string          `json:"tool_name"`
	Input     json.RawMessage `json:"input"`
	Output    json.RawMessage `json:"output,omitempty"`
	IsError   bool            `json:"is_error"`
}

type TagCount struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Count int    `json:"count"`
}
