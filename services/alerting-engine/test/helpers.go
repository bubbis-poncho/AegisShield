package test

import (
	"log/slog"
	"os"
)

// setupTestLogger creates a logger for testing
func setupTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
}