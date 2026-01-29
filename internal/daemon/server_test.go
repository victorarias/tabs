package daemon

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"os"
	"testing"
	"time"
)

func TestServerDaemonStatus(t *testing.T) {
	baseDir := t.TempDir()
	if err := os.MkdirAll(baseDir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := NewServer(baseDir, logger)
	if err := srv.Listen(); err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = srv.Serve(ctx)
	}()

	conn, err := net.DialTimeout("unix", SocketPath(baseDir), 2*time.Second)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()

	req := map[string]interface{}{
		"version": "1.0",
		"type":    "daemon_status",
		"payload": map[string]interface{}{},
	}
	if err := json.NewEncoder(conn).Encode(req); err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	reader := bufio.NewReader(conn)
	line, err := reader.ReadBytes('\n')
	if err != nil && err != io.EOF {
		t.Fatalf("read failed: %v", err)
	}
	var resp response
	if err := json.Unmarshal(bytesTrimSpace(line), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if resp.Status != "ok" {
		t.Fatalf("expected ok response")
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelShutdown()
	_ = srv.Shutdown(shutdownCtx)
}

func TestServerRejectsBadVersion(t *testing.T) {
	baseDir := t.TempDir()
	if err := os.MkdirAll(baseDir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	srv := NewServer(baseDir, nil)
	if err := srv.Listen(); err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = srv.Serve(ctx)
	}()

	conn, err := net.DialTimeout("unix", SocketPath(baseDir), 2*time.Second)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()

	req := map[string]interface{}{
		"version": "0.9",
		"type":    "daemon_status",
		"payload": map[string]interface{}{},
	}
	if err := json.NewEncoder(conn).Encode(req); err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	reader := bufio.NewReader(conn)
	line, err := reader.ReadBytes('\n')
	if err != nil && err != io.EOF {
		t.Fatalf("read failed: %v", err)
	}
	var resp response
	if err := json.Unmarshal(bytesTrimSpace(line), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if resp.Status != "error" || resp.Error == nil {
		t.Fatalf("expected error response")
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelShutdown()
	_ = srv.Shutdown(shutdownCtx)
}
