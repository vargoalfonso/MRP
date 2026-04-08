// Package logger sets up the application-wide structured logger (log/slog)
// and provides context-aware helpers used throughout the codebase.
package logger

import (
	"context"
	"log/slog"
	"os"
)

// contextKey is an unexported type to prevent key collision across packages.
type contextKey string

const (
	// RequestIDKey is the context key for the inbound request ID (UUIDv4).
	RequestIDKey contextKey = "request_id"
	// LoggerKey is the context key that carries the per-request *slog.Logger.
	LoggerKey contextKey = "logger"
)

// New creates and registers a global slog.Logger.
//
//	level:  "debug" | "info" | "warn" | "error"
//	format: "json"  | "text"  (anything else defaults to text)
func New(level, format string) *slog.Logger {
	var lvl slog.Level
	if err := lvl.UnmarshalText([]byte(level)); err != nil {
		lvl = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: lvl}

	var handler slog.Handler
	if format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}

// WithRequestID injects a request ID into the context and returns a logger
// pre-populated with the request_id attribute.
func WithRequestID(ctx context.Context, requestID string) (context.Context, *slog.Logger) {
	ctx = context.WithValue(ctx, RequestIDKey, requestID)
	logger := slog.Default().With(slog.String("request_id", requestID))
	ctx = context.WithValue(ctx, LoggerKey, logger)
	return ctx, logger
}

// FromContext retrieves the per-request logger stored in the context.
// Falls back to the global default logger so callers never receive nil.
func FromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(LoggerKey).(*slog.Logger); ok && l != nil {
		return l
	}
	return slog.Default()
}

// RequestIDFromContext returns the request ID stored in the context, or empty string.
func RequestIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return ""
}
