// Package platform provides shared infrastructure (logging, database, redis)
// that every Postal domain reuses. It holds no business logic.
package platform

import (
	"log/slog"
	"os"
)

// NewLogger builds a structured slog.Logger. Production emits JSON; other
// environments emit human-readable text. level is one of debug|info|warn|error.
func NewLogger(level string, production bool) *slog.Logger {
	opts := &slog.HandlerOptions{Level: parseLevel(level)}

	var handler slog.Handler
	if production {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}
	return slog.New(handler)
}

// parseLevel maps a level string to slog.Level, defaulting to Info on unknown input.
func parseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
