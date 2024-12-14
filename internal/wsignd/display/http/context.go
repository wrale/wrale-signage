package http

import (
	"context"
	"log/slog"
)

// loggerKey is used to store the request-scoped logger in context
type loggerKey struct{}

// GetLogger retrieves the request-scoped logger from context.
// Falls back to the default logger if no logger is found in context.
func GetLogger(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey{}).(*slog.Logger); ok {
		return logger
	}
	return slog.Default()
}
