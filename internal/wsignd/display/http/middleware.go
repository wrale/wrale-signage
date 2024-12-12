package http

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"log/slog"

	"github.com/wrale/wrale-signage/internal/wsignd/auth"
)

// Key type for context values
type contextKey int

const (
	// displayIDKey is the context key for the authenticated display ID
	displayIDKey contextKey = iota
)

// GetDisplayID retrieves the authenticated display ID from the context
func GetDisplayID(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(displayIDKey).(uuid.UUID)
	return id, ok
}

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

// authMiddleware validates bearer tokens and adds display ID to context
func authMiddleware(authService auth.Service, logger *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "missing authorization header", http.StatusUnauthorized)
				return
			}

			// Validate bearer token format
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "invalid authorization header", http.StatusUnauthorized)
				return
			}

			// Validate token with auth service
			displayID, err := authService.ValidateAccessToken(r.Context(), parts[1])
			if err != nil {
				if err == auth.ErrTokenExpired {
					http.Error(w, "token expired", http.StatusUnauthorized)
				} else {
					http.Error(w, "invalid token", http.StatusUnauthorized)
				}
				logger.Error("auth failed",
					"error", err,
					"path", r.URL.Path,
					"remoteIP", r.RemoteAddr,
				)
				return
			}

			// Add display ID to context
			ctx := context.WithValue(r.Context(), displayIDKey, displayID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
