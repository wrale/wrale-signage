package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"log/slog"

	"github.com/wrale/wrale-signage/internal/wsignd/auth"
	werrors "github.com/wrale/wrale-signage/internal/wsignd/errors"
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

// recoverMiddleware recovers from panics and returns JSON error responses
func recoverMiddleware(logger *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rvr := recover(); rvr != nil {
					reqID := middleware.GetReqID(r.Context())
					logger.Error("panic recovery",
						"error", rvr,
						"stack", string(debug.Stack()),
						"path", r.URL.Path,
						"requestId", reqID,
					)

					// Ensure clean response
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)

					// Write standardized error response
					err := werrors.NewError("INTERNAL", "internal error", "panic", nil)
					json.NewEncoder(w).Encode(map[string]string{
						"code":    err.Code,
						"message": err.Message,
					})
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// authMiddleware validates bearer tokens and adds display ID to context
func authMiddleware(authService auth.Service, logger *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				err := werrors.NewError("UNAUTHORIZED", "missing authorization header", "auth", nil)
				writeJSONError(w, err, http.StatusUnauthorized)
				return
			}

			// Validate bearer token format
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				err := werrors.NewError("UNAUTHORIZED", "invalid authorization header", "auth", nil)
				writeJSONError(w, err, http.StatusUnauthorized)
				return
			}

			// Validate token with auth service
			displayID, err := authService.ValidateAccessToken(r.Context(), parts[1])
			if err != nil {
				if err == auth.ErrTokenExpired {
					writeJSONError(w, werrors.NewError("UNAUTHORIZED", "token expired", "auth", err), http.StatusUnauthorized)
				} else {
					writeJSONError(w, werrors.NewError("UNAUTHORIZED", "invalid token", "auth", err), http.StatusUnauthorized)
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

// writeJSONError writes a standard JSON error response
func writeJSONError(w http.ResponseWriter, err error, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	var werr *werrors.Error
	if !errors.As(err, &werr) {
		werr = werrors.NewError("INTERNAL", err.Error(), "", err)
	}

	json.NewEncoder(w).Encode(map[string]string{
		"code":    werr.Code,
		"message": werr.Message,
	})
}
