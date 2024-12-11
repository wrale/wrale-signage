package http

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"log/slog"
)

// logEntry holds data for logging
type logEntry struct {
	StartTime time.Time
	Method    string
	Path      string
	Query     string
	RemoteIP  string
	RequestID string
	Status    int
	Size      int64
	Duration  time.Duration
}

// logMiddleware logs requests with detailed information
func logMiddleware(logger *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			entry := &logEntry{
				StartTime: time.Now(),
				Method:    r.Method,
				Path:      r.URL.Path,
				Query:     r.URL.RawQuery,
				RemoteIP:  r.RemoteAddr,
				RequestID: middleware.GetReqID(r.Context()),
			}

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			// Call the next handler
			next.ServeHTTP(ww, r)

			// Record response details
			entry.Status = ww.Status()
			entry.Size = int64(ww.BytesWritten())
			entry.Duration = time.Since(entry.StartTime)

			// Log at appropriate level
			logFn := logger.Info
			if entry.Status >= 400 {
				logFn = logger.Error
			}

			// Build log attributes
			attrs := []any{
				"requestId", entry.RequestID,
				"method", entry.Method,
				"path", entry.Path,
				"status", entry.Status,
				"duration", entry.Duration,
				"size", entry.Size,
				"remoteIP", entry.RemoteIP,
			}

			if entry.Query != "" {
				attrs = append(attrs, "query", entry.Query)
			}

			// Log the request
			logFn("http request",
				attrs...,
			)
		})
	}
}

// requestIDHeaderMiddleware ensures request ID is in response headers
func requestIDHeaderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if reqID := middleware.GetReqID(r.Context()); reqID != "" {
			w.Header().Set("X-Request-ID", reqID)
		}
		next.ServeHTTP(w, r)
	})
}
