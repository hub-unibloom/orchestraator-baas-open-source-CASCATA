package logger

import (
	"context"
	"log/slog"
	"os"
)

var Log *slog.Logger

// Init initializes the structured logger with default fields.
func Init(level string) {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: logLevel,
	}

	handler := slog.NewJSONHandler(os.Stdout, opts)
	Log = slog.New(handler)
	slog.SetDefault(Log)
}

// WithContext adds tracing fields to the logger from the context.
func WithContext(ctx context.Context) *slog.Logger {
	// Future implementation: Extract RequestID, TraceID, etc.
	return Log
}
