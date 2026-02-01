package daemon

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

var updateGolden = flag.Bool("update", false, "update golden files")

// =============================================================================
// Unit Tests: Core extraction logic (the bug fix)
// =============================================================================

func TestExtractMessageContent(t *testing.T) {
	tests := []struct {
		name    string
		record  map[string]interface{}
		wantNil bool
		want    interface{}
	}{
		{
			name: "nested message.content",
			record: map[string]interface{}{
				"type": "user",
				"message": map[string]interface{}{
					"role":    "user",
					"content": "hello world",
				},
			},
			want: "hello world",
		},
		{
			name: "top-level content fallback",
			record: map[string]interface{}{
				"type":    "user",
				"content": "fallback content",
			},
			want: "fallback content",
		},
		{
			name: "prefers nested over top-level",
			record: map[string]interface{}{
				"type":    "user",
				"content": "should not use",
				"message": map[string]interface{}{
					"content": "should use",
				},
			},
			want: "should use",
		},
		{
			name: "message exists but no content returns nil",
			record: map[string]interface{}{
				"type":    "user",
				"message": map[string]interface{}{"role": "user"},
			},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractMessageContent(tt.record)
			if tt.wantNil {
				if got != nil {
					t.Errorf("expected nil, got %v", got)
				}
				return
			}
			if got != tt.want {
				t.Errorf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

func TestExtractMessageModel(t *testing.T) {
	tests := []struct {
		name   string
		record map[string]interface{}
		want   string
	}{
		{
			name: "nested message.model",
			record: map[string]interface{}{
				"message": map[string]interface{}{"model": "claude-opus-4-5-20251101"},
			},
			want: "claude-opus-4-5-20251101",
		},
		{
			name: "prefers nested over top-level",
			record: map[string]interface{}{
				"model":   "should-not-use",
				"message": map[string]interface{}{"model": "should-use"},
			},
			want: "should-use",
		},
		{
			name:   "top-level fallback",
			record: map[string]interface{}{"model": "fallback-model"},
			want:   "fallback-model",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractMessageModel(tt.record)
			if got != tt.want {
				t.Errorf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

// =============================================================================
// Unit Tests: Content normalization (transformation logic)
// =============================================================================

func TestNormalizeContent(t *testing.T) {
	tests := []struct {
		name string
		raw  interface{}
		want []map[string]interface{}
	}{
		{
			name: "string to text block",
			raw:  "hello world",
			want: []map[string]interface{}{{"type": "text", "text": "hello world"}},
		},
		{
			name: "array of objects preserves type",
			raw: []interface{}{
				map[string]interface{}{"type": "thinking", "text": "hmm"},
				map[string]interface{}{"type": "text", "text": "response"},
			},
			want: []map[string]interface{}{
				{"type": "thinking", "text": "hmm"},
				{"type": "text", "text": "response"},
			},
		},
		{
			name: "skips objects missing required fields",
			raw: []interface{}{
				map[string]interface{}{"type": "text"},           // missing text
				map[string]interface{}{"text": "foo"},            // missing type
				map[string]interface{}{"type": "text", "text": "valid"},
			},
			want: []map[string]interface{}{{"type": "text", "text": "valid"}},
		},
		{
			name: "mixed string and object",
			raw: []interface{}{
				"plain",
				map[string]interface{}{"type": "text", "text": "object"},
			},
			want: []map[string]interface{}{
				{"type": "text", "text": "plain"},
				{"type": "text", "text": "object"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeContent(tt.raw)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

// =============================================================================
// Unit Tests: Role detection (priority logic only)
// =============================================================================

func TestClaudeRole(t *testing.T) {
	// Only test the non-obvious behavior: type takes precedence over role
	record := map[string]interface{}{"type": "user", "role": "assistant"}
	got := claudeRole(record)
	if got != "user" {
		t.Errorf("type should take precedence: expected 'user', got %q", got)
	}
}

// =============================================================================
// Unit Tests: Tool extraction (alternate key fallback logic)
// =============================================================================

func TestExtractToolUse(t *testing.T) {
	tests := []struct {
		name   string
		record map[string]interface{}
		want   map[string]interface{}
	}{
		{
			name: "standard keys",
			record: map[string]interface{}{
				"tool_use": map[string]interface{}{
					"id":    "toolu_123",
					"name":  "Read",
					"input": map[string]interface{}{"file_path": "/tmp/test.txt"},
				},
			},
			want: map[string]interface{}{
				"tool_use_id": "toolu_123",
				"tool_name":   "Read",
				"input":       map[string]interface{}{"file_path": "/tmp/test.txt"},
			},
		},
		{
			name: "alternate keys (tool_use_id, tool_name)",
			record: map[string]interface{}{
				"tool_use": map[string]interface{}{
					"tool_use_id": "toolu_456",
					"tool_name":   "Write",
					"input":       map[string]interface{}{},
				},
			},
			want: map[string]interface{}{
				"tool_use_id": "toolu_456",
				"tool_name":   "Write",
				"input":       map[string]interface{}{},
			},
		},
		{
			name: "missing required fields returns nil",
			record: map[string]interface{}{
				"tool_use": map[string]interface{}{"name": "Read"}, // missing id
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractToolUse(tt.record)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

func TestExtractToolResult(t *testing.T) {
	tests := []struct {
		name   string
		record map[string]interface{}
		want   map[string]interface{}
	}{
		{
			name: "standard key (tool_use_id)",
			record: map[string]interface{}{
				"tool_result": map[string]interface{}{
					"tool_use_id": "toolu_123",
					"content":     "result",
					"is_error":    false,
				},
			},
			want: map[string]interface{}{
				"tool_use_id": "toolu_123",
				"content":     "result",
				"is_error":    false,
			},
		},
		{
			name: "alternate key (id)",
			record: map[string]interface{}{
				"tool_result": map[string]interface{}{
					"id":       "toolu_789",
					"content":  "result",
					"is_error": true,
				},
			},
			want: map[string]interface{}{
				"tool_use_id": "toolu_789",
				"content":     "result",
				"is_error":    true,
			},
		},
		{
			name: "missing id returns nil",
			record: map[string]interface{}{
				"tool_result": map[string]interface{}{"content": "result"},
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractToolResult(tt.record)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

// =============================================================================
// Unit Tests: Line parsing (error handling only)
// =============================================================================

func TestClaudeEventsFromLineInvalidJSON(t *testing.T) {
	fallback := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	_, _, err := claudeEventsFromLine([]byte(`{not valid json`), "test-session", fallback)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

// =============================================================================
// Golden File Tests (real behavior against realistic inputs)
// =============================================================================

func TestClaudeEventsFromLineGolden(t *testing.T) {
	testdataDir := filepath.Join("testdata", "transcripts")
	if _, err := os.Stat(testdataDir); os.IsNotExist(err) {
		t.Skip("testdata/transcripts directory not found")
	}

	entries, err := os.ReadDir(testdataDir)
	if err != nil {
		t.Fatalf("failed to read testdata: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}

		t.Run(entry.Name(), func(t *testing.T) {
			transcriptPath := filepath.Join(testdataDir, entry.Name())
			expectedPath := filepath.Join("testdata", "expected", strings.TrimSuffix(entry.Name(), ".jsonl")+".json")

			file, err := os.Open(transcriptPath)
			if err != nil {
				t.Fatalf("failed to open transcript: %v", err)
			}
			defer file.Close()

			fallback := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
			var allEvents []map[string]interface{}
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := bytes.TrimSpace(scanner.Bytes())
				if len(line) == 0 {
					continue
				}
				events, _, err := claudeEventsFromLine(line, "test-session", fallback)
				if err != nil {
					continue
				}
				allEvents = append(allEvents, events...)
			}
			if err := scanner.Err(); err != nil {
				t.Fatalf("scanner error: %v", err)
			}

			if *updateGolden {
				out, err := json.MarshalIndent(allEvents, "", "  ")
				if err != nil {
					t.Fatalf("failed to marshal events: %v", err)
				}
				if err := os.MkdirAll(filepath.Dir(expectedPath), 0o755); err != nil {
					t.Fatalf("failed to create expected dir: %v", err)
				}
				if err := os.WriteFile(expectedPath, out, 0o644); err != nil {
					t.Fatalf("failed to write expected file: %v", err)
				}
				t.Logf("updated golden file: %s", expectedPath)
				return
			}

			expectedData, err := os.ReadFile(expectedPath)
			if err != nil {
				t.Skipf("expected file not found: %s (run with -update to create)", expectedPath)
			}

			var expected []map[string]interface{}
			if err := json.Unmarshal(expectedData, &expected); err != nil {
				t.Fatalf("failed to unmarshal expected: %v", err)
			}

			if len(allEvents) != len(expected) {
				t.Errorf("event count mismatch: got %d, expected %d", len(allEvents), len(expected))
				return
			}

			for i := range expected {
				gotType, _ := allEvents[i]["event_type"].(string)
				wantType, _ := expected[i]["event_type"].(string)
				if gotType != wantType {
					t.Errorf("event %d: type mismatch: got %q, want %q", i, gotType, wantType)
				}
			}
		})
	}
}

// =============================================================================
// Integration Tests (system behavior, file I/O edge cases)
// =============================================================================

func TestIntegrationAppendClaudeTranscript(t *testing.T) {
	baseDir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := NewServer(baseDir, logger)

	transcriptPath := filepath.Join(t.TempDir(), "transcript.jsonl")
	lines := []string{
		`{"type":"user","message":{"role":"user","content":"hello"},"timestamp":"2026-01-01T12:00:00Z"}`,
		`{"type":"assistant","message":{"role":"assistant","model":"claude-opus-4-5-20251101","content":[{"type":"text","text":"hi"}]},"timestamp":"2026-01-01T12:00:01Z"}`,
	}
	if err := os.WriteFile(transcriptPath, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("failed to write transcript: %v", err)
	}

	sessionID := "test-integration"
	cursor := &SessionCursor{SessionID: sessionID, TranscriptPath: transcriptPath}
	sessionPath, err := srv.state.EnsureSessionFile(baseDir, sessionID, "claude-code", time.Now())
	if err != nil {
		t.Fatalf("failed to create session file: %v", err)
	}

	written, _, _, _, err := srv.appendClaudeTranscript(sessionPath, sessionID, cursor, time.Now())
	if err != nil {
		t.Fatalf("appendClaudeTranscript failed: %v", err)
	}
	if written != 2 {
		t.Errorf("expected 2 events, got %d", written)
	}
}

func TestIntegrationCursorResume(t *testing.T) {
	baseDir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := NewServer(baseDir, logger)

	transcriptPath := filepath.Join(t.TempDir(), "transcript.jsonl")
	lines := []string{
		`{"type":"user","message":{"role":"user","content":"msg1"},"timestamp":"2026-01-01T12:00:00Z"}`,
		`{"type":"user","message":{"role":"user","content":"msg2"},"timestamp":"2026-01-01T12:00:01Z"}`,
	}
	if err := os.WriteFile(transcriptPath, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("failed to write transcript: %v", err)
	}

	sessionID := "test-resume"
	sessionPath, _ := srv.state.EnsureSessionFile(baseDir, sessionID, "claude-code", time.Now())

	// First capture
	cursor1 := &SessionCursor{SessionID: sessionID, TranscriptPath: transcriptPath}
	written1, _, offset1, hash1, _ := srv.appendClaudeTranscript(sessionPath, sessionID, cursor1, time.Now())
	if written1 != 2 {
		t.Fatalf("first capture: expected 2, got %d", written1)
	}

	// Second capture with same cursor state - should get 0
	cursor2 := &SessionCursor{SessionID: sessionID, TranscriptPath: transcriptPath, LastOffset: offset1, LastLineHash: hash1}
	written2, _, _, _, _ := srv.appendClaudeTranscript(sessionPath, sessionID, cursor2, time.Now())
	if written2 != 0 {
		t.Errorf("resume capture: expected 0 new events, got %d", written2)
	}

	// Append new line
	f, _ := os.OpenFile(transcriptPath, os.O_APPEND|os.O_WRONLY, 0o644)
	f.WriteString(`{"type":"user","message":{"role":"user","content":"msg3"},"timestamp":"2026-01-01T12:00:02Z"}` + "\n")
	f.Close()

	// Third capture - should get only the new one
	cursor3 := &SessionCursor{SessionID: sessionID, TranscriptPath: transcriptPath, LastOffset: offset1, LastLineHash: hash1}
	written3, _, _, _, _ := srv.appendClaudeTranscript(sessionPath, sessionID, cursor3, time.Now())
	if written3 != 1 {
		t.Errorf("incremental capture: expected 1 new event, got %d", written3)
	}
}

func TestIntegrationFileTruncation(t *testing.T) {
	baseDir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := NewServer(baseDir, logger)

	transcriptPath := filepath.Join(t.TempDir(), "transcript.jsonl")
	initialContent := `{"type":"user","message":{"role":"user","content":"original long message here"},"timestamp":"2026-01-01T12:00:00Z"}` + "\n"
	if err := os.WriteFile(transcriptPath, []byte(initialContent), 0o644); err != nil {
		t.Fatalf("failed to write transcript: %v", err)
	}

	sessionID := "test-truncation"
	sessionPath, _ := srv.state.EnsureSessionFile(baseDir, sessionID, "claude-code", time.Now())

	// First capture
	cursor1 := &SessionCursor{SessionID: sessionID, TranscriptPath: transcriptPath}
	_, _, offset1, _, _ := srv.appendClaudeTranscript(sessionPath, sessionID, cursor1, time.Now())

	// Truncate file (smaller content)
	smallerContent := `{"type":"user","message":{"role":"user","content":"new"}}` + "\n"
	os.WriteFile(transcriptPath, []byte(smallerContent), 0o644)

	// Capture with old offset past new EOF - should reset and read
	cursor2 := &SessionCursor{SessionID: sessionID, TranscriptPath: transcriptPath, LastOffset: offset1}
	written, _, _, _, err := srv.appendClaudeTranscript(sessionPath, sessionID, cursor2, time.Now())
	if err != nil {
		t.Fatalf("capture after truncation failed: %v", err)
	}
	if written != 1 {
		t.Errorf("expected 1 event after truncation reset, got %d", written)
	}
}
