package daemon

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CleanupEmptySessions removes empty session files older than the given retention period.
// Empty sessions are those with message_count == 0.
// Returns the number of sessions deleted and any error encountered.
func CleanupEmptySessions(baseDir string, retentionHours int) (int, error) {
	if retentionHours <= 0 {
		return 0, nil // Cleanup disabled
	}

	sessionsDir := filepath.Join(baseDir, "sessions")
	cutoff := time.Now().Add(-time.Duration(retentionHours) * time.Hour)

	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	deleted := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dayDir := filepath.Join(sessionsDir, entry.Name())
		files, err := os.ReadDir(dayDir)
		if err != nil {
			continue
		}

		for _, file := range files {
			if file.IsDir() || !strings.HasSuffix(file.Name(), ".jsonl") {
				continue
			}
			filePath := filepath.Join(dayDir, file.Name())

			// Check if session is empty and old enough
			isEmpty, createdAt, err := isEmptySession(filePath)
			if err != nil {
				continue
			}

			if isEmpty && !createdAt.IsZero() && createdAt.Before(cutoff) {
				if err := os.Remove(filePath); err == nil {
					deleted++
				}
			}
		}

		// Try to remove empty day directories
		remaining, _ := os.ReadDir(dayDir)
		if len(remaining) == 0 {
			_ = os.Remove(dayDir)
		}
	}

	return deleted, nil
}

// isEmptySession checks if a session file has zero messages.
// Returns (isEmpty, createdAt, error).
func isEmptySession(path string) (bool, time.Time, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, time.Time{}, err
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	messageCount := 0
	var createdAt time.Time

	for {
		line, err := reader.ReadBytes('\n')
		if len(line) == 0 && errors.Is(err, io.EOF) {
			break
		}
		if errors.Is(err, io.EOF) && len(line) > 0 && !bytes.HasSuffix(line, []byte{'\n'}) {
			// Last line without newline
		} else if err != nil && !errors.Is(err, io.EOF) {
			return false, time.Time{}, err
		}

		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			if errors.Is(err, io.EOF) {
				break
			}
			continue
		}

		var event map[string]interface{}
		if jsonErr := json.Unmarshal(line, &event); jsonErr != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			continue
		}

		// Track earliest timestamp as creation time
		if ts, ok := event["timestamp"].(string); ok && createdAt.IsZero() {
			if parsed, parseErr := time.Parse(time.RFC3339Nano, ts); parseErr == nil {
				createdAt = parsed
			} else if parsed, parseErr := time.Parse(time.RFC3339, ts); parseErr == nil {
				createdAt = parsed
			}
		}

		eventType, _ := event["event_type"].(string)
		if eventType == "message" {
			messageCount++
		}

		// Check session_end for override counts
		if eventType == "session_end" {
			if data, ok := event["data"].(map[string]interface{}); ok {
				if count, ok := data["message_count"].(float64); ok {
					messageCount = int(count)
				}
			}
		}

		if errors.Is(err, io.EOF) {
			break
		}
	}

	return messageCount == 0, createdAt, nil
}

// StartCleanupRoutine starts a background goroutine that periodically cleans up
// empty sessions. It runs once on startup and then every hour.
func StartCleanupRoutine(ctx context.Context, baseDir string, retentionHours int, logger *slog.Logger) {
	if retentionHours <= 0 {
		if logger != nil {
			logger.Info("empty session cleanup disabled")
		}
		return
	}

	if logger != nil {
		logger.Info("starting empty session cleanup routine", "retention_hours", retentionHours)
	}

	// Run cleanup in a goroutine
	go func() {
		// Run immediately on startup
		runCleanup(baseDir, retentionHours, logger)

		// Then run every hour
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				runCleanup(baseDir, retentionHours, logger)
			}
		}
	}()
}

func runCleanup(baseDir string, retentionHours int, logger *slog.Logger) {
	deleted, err := CleanupEmptySessions(baseDir, retentionHours)
	if err != nil {
		if logger != nil {
			logger.Error("cleanup failed", "error", err)
		}
		return
	}
	if deleted > 0 && logger != nil {
		logger.Info("cleaned up empty sessions", "deleted", deleted)
	}
}
