package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"log/slog"

	"github.com/wrale/wrale-signage/internal/wsignd/auth"
	werrors "github.com/wrale/wrale-signage/internal/wsignd/errors"
	"github.com/wrale/wrale-signage/internal/wsignd/ratelimit"
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
					stack := string(debug.Stack())

					logger.Error("panic recovery",
						"error", rvr,
						"stack", stack,
						"path", r.URL.Path,
						"requestId", reqID,
					)

					// Create error with panic details
					err := werrors.NewError(
						"INTERNAL",
						fmt.Sprintf("internal error: %v", rvr),
						"panic",
						fmt.Errorf("panic: %v", rvr),
					)

					// Write error response
					writeJSONError(w, err, http.StatusInternalServerError, logger.With(
						"requestId", reqID,
						"path", r.URL.Path,
					))
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// rateLimitMiddleware wraps the ratelimit.Middleware to provide consistent error responses
func rateLimitMiddleware(service ratelimit.Service, logger *slog.Logger, options ratelimit.RateLimitOptions) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		rateLimitNext := ratelimit.Middleware(service, logger, options)(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID := middleware.GetReqID(r.Context())

			// Handle rate limit errors with consistent format
			defer func() {
				if rvr := recover(); rvr != nil {
					if limitErr, ok := rvr.(error); ok && errors.Is(limitErr, ratelimit.ErrLimitExceeded) {
						err := werrors.NewError(
							"RATE_LIMITED",
							"too many requests",
							"ratelimit",
							limitErr,
						)
						w.Header().Set("Retry-After", "60") // Default 1 minute retry
						writeJSONError(w, err, http.StatusTooManyRequests, logger.With(
							"requestId", reqID,
							"path", r.URL.Path,
						))
						return
					}
					panic(rvr) // Re-panic if not a rate limit error
				}
			}()

			rateLimitNext.ServeHTTP(w, r)
		})
	}
}

// authMiddleware validates bearer tokens and adds display ID to context
func authMiddleware(authService auth.Service, logger *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID := middleware.GetReqID(r.Context())
			reqLogger := logger.With(
				"requestId", reqID,
				"path", r.URL.Path,
			)

			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				err := werrors.NewError("UNAUTHORIZED", "missing authorization header", "auth", nil)
				writeJSONError(w, err, http.StatusUnauthorized, reqLogger)
				return
			}

			// Validate bearer token format
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				err := werrors.NewError("UNAUTHORIZED", "invalid authorization header", "auth", nil)
				writeJSONError(w, err, http.StatusUnauthorized, reqLogger)
				return
			}

			// Validate token with auth service
			displayID, err := authService.ValidateAccessToken(r.Context(), parts[1])
			if err != nil {
				var errCode, errMsg string
				if err == auth.ErrTokenExpired {
					errCode = "TOKEN_EXPIRED"
					errMsg = "access token has expired"
				} else {
					errCode = "UNAUTHORIZED"
					errMsg = "invalid access token"
				}

				writeJSONError(w, werrors.NewError(errCode, errMsg, "auth", err), http.StatusUnauthorized, reqLogger)
				reqLogger.Error("auth failed",
					"error", err,
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
func writeJSONError(w http.ResponseWriter, err error, status int, logger *slog.Logger) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	// Convert to werrors.Error if not already
	var werr *werrors.Error
	if !errors.As(err, &werr) {
		werr = werrors.NewError("INTERNAL", err.Error(), "", err)
	}

	// Build error response
	resp := map[string]string{
		"code":    werr.Code,
		"message": werr.Message,
	}

	if encodeErr := json.NewEncoder(w).Encode(resp); encodeErr != nil {
		logger.Error("failed to write error response",
			"error", encodeErr,
			"originalError", err,
		)
	}
}
