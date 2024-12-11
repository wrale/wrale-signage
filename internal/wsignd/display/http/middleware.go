package http

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"log/slog"
)

// logMiddleware logs requests with detailed information
func logMiddleware(logger *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract request info before processing
			entry := &logEntry{
				StartTime: time.Now(),
				Method:    r.Method,
				Path:      r.URL.Path,
				Query:     r.URL.RawQuery,
				RemoteIP:  r.RemoteAddr,
				RequestID: middleware.GetReqID(r.Context()),
			}

			// Create wrapped response writer to capture status and size
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			defer func() {
				// Record response details after completion
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
				logFn("http request", attrs...)
			}()

			// Call the next handler
			next.ServeHTTP(ww, r)
		})
	}
}

// requestIDHeaderMiddleware ensures request ID is in response headers
func requestIDHeaderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add request ID header before next handler
		reqID := middleware.GetReqID(r.Context())
		if reqID != "" {
			w.Header().Set("X-Request-ID", reqID)
		}
		next.ServeHTTP(w, r)
	})
}

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
