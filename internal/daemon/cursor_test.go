package daemon

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

func TestParseCursorConversation(t *testing.T) {
	raw := []byte(`{
		"conversation_id": "conv-123",
		"workspace_roots": ["/home/victor/projects/tab"],
		"messages": [
			{ "role": "user", "content": "hello", "timestamp": "2026-01-28T12:00:00Z" }
		]
	}`)

	conv, ok := parseCursorConversation(raw)
	if !ok {
		t.Fatal("expected conversation to parse")
	}
	if conv.ID != "conv-123" {
		t.Fatalf("expected id conv-123, got %q", conv.ID)
	}
	if len(conv.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(conv.Messages))
	}
	if conv.Messages[0].Role != "user" {
		t.Fatalf("expected role user, got %q", conv.Messages[0].Role)
	}
	if conv.Messages[0].Content != "hello" {
		t.Fatalf("expected content hello, got %q", conv.Messages[0].Content)
	}
	if len(conv.WorkspaceRoots) != 1 || conv.WorkspaceRoots[0] != "/home/victor/projects/tab" {
		t.Fatalf("expected workspace root to be parsed")
	}
}

func TestParseCursorConversationAlternateKeys(t *testing.T) {
	raw := []byte(`{
		"conversationId": "conv-456",
		"workspaceRoots": ["/tmp/project"],
		"messages": [
			{ "role": "assistant", "content": { "text": "hi" }, "created_at": "2026-01-28T12:01:00Z" }
		]
	}`)

	conv, ok := parseCursorConversation(raw)
	if !ok {
		t.Fatal("expected conversation to parse")
	}
	if conv.ID != "conv-456" {
		t.Fatalf("expected id conv-456, got %q", conv.ID)
	}
	if len(conv.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(conv.Messages))
	}
	if conv.Messages[0].Role != "assistant" {
		t.Fatalf("expected role assistant, got %q", conv.Messages[0].Role)
	}
	if conv.Messages[0].Content == "" {
		t.Fatalf("expected content to be set")
	}
	if len(conv.WorkspaceRoots) != 1 || conv.WorkspaceRoots[0] != "/tmp/project" {
		t.Fatalf("expected workspace roots to be parsed")
	}
}

func TestParseCursorConversationMissingMessages(t *testing.T) {
	raw := []byte(`{"conversation_id": "conv-789"}`)
	if _, ok := parseCursorConversation(raw); ok {
		t.Fatal("expected missing messages to return false")
	}
}

func TestPollCursorDBWritesSession(t *testing.T) {
	baseDir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := NewServer(baseDir, logger)

	dbPath := filepath.Join(t.TempDir(), "state.vscdb")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	_, err = db.Exec(`CREATE TABLE ItemTable ([key] TEXT, value BLOB)`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
	payload := `{"conversation_id":"conv-abc","messages":[{"role":"user","content":"hello","timestamp":"2026-01-28T12:00:00Z"}]}`
	_, err = db.Exec(`INSERT INTO ItemTable ([key], value) VALUES (?, ?)`, "workbench.panel.aichat.view.aichat.chatdata", payload)
	if err != nil {
		t.Fatalf("insert row: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	if err := srv.pollCursorDB(dbPath); err != nil {
		t.Fatalf("poll cursor db: %v", err)
	}

	path, ok, err := findExistingSessionFile(baseDir, "conv-abc", "cursor")
	if err != nil {
		t.Fatalf("find session file: %v", err)
	}
	if !ok || path == "" {
		t.Fatal("expected session file to be created")
	}

	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open session file: %v", err)
	}
	defer file.Close()

	var types []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var event map[string]interface{}
		if err := json.Unmarshal(line, &event); err != nil {
			t.Fatalf("unmarshal event: %v", err)
		}
		if value, ok := event["event_type"].(string); ok {
			types = append(types, value)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan session file: %v", err)
	}

	hasStart := false
	hasMessage := false
	for _, typ := range types {
		if typ == "session_start" {
			hasStart = true
		}
		if typ == "message" {
			hasMessage = true
		}
	}
	if !hasStart || !hasMessage {
		t.Fatalf("expected session_start and message events, got %v", types)
	}
}
