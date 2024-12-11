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
			startTime := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			reqID := middleware.GetReqID(r.Context())

			defer func() {
				duration := time.Since(startTime)
				status := ww.Status()

				// Log at appropriate level
				logFn := logger.Info
				if status >= 500 {
					logFn = logger.Error
				}

				// Log request details
				logFn("http request",
					"requestId", reqID,
					"method", r.Method,
					"path", r.URL.Path,
					"status", status,
					"duration", duration,
					"size", ww.BytesWritten(),
					"remoteIP", r.RemoteAddr,
				)
			}()

			next.ServeHTTP(ww, r)
		})
	}
}

// requestIDHeaderMiddleware ensures request ID is in response headers
func requestIDHeaderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if reqID := middleware.GetReqID(r.Context()); reqID != "" {
			w.Header().Set("Request-ID", reqID)
			w.Header().Set("X-Request-ID", reqID) // Keep X- header for compatibility
		}
		next.ServeHTTP(w, r)
	})
}
