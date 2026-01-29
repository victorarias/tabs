package logging

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

func New(level string, out io.Writer) *slog.Logger {
	if out == nil {
		out = os.Stdout
	}
	handler := slog.NewTextHandler(out, &slog.HandlerOptions{
		Level: parseLevel(level),
	})
	return slog.New(handler)
}

func parseLevel(raw string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
