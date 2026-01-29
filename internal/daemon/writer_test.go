package daemon

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppendJSONLAddsNewline(t *testing.T) {
	baseDir := t.TempDir()
	path := filepath.Join(baseDir, "events.jsonl")
	_, err := appendJSONL(path, []byte(`{"event":"one"}`))
	if err != nil {
		t.Fatalf("append failed: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if !strings.HasSuffix(string(data), "\n") {
		t.Fatalf("expected newline suffix")
	}
}

func TestEnsureNewline(t *testing.T) {
	if got := string(ensureNewline([]byte(""))); got != "\n" {
		t.Fatalf("expected newline for empty, got %q", got)
	}
	if got := string(ensureNewline([]byte("hi"))); got != "hi\n" {
		t.Fatalf("expected newline appended, got %q", got)
	}
	if got := string(ensureNewline([]byte("hi\n"))); got != "hi\n" {
		t.Fatalf("expected unchanged, got %q", got)
	}
}
