package http

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/wrale/wrale-signage/internal/wsignd/auth"
	werrors "github.com/wrale/wrale-signage/internal/wsignd/errors"
	"github.com/wrale/wrale-signage/internal/wsignd/ratelimit"
)

// contextKey type for context values
type contextKey int

const (
	displayIDKey contextKey = iota
)

// GetDisplayID retrieves the authenticated display ID from context
func GetDisplayID(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(displayIDKey).(uuid.UUID)
	return id, ok
}

// logMiddleware ensures consistent structured logging across all requests.
// It creates a request-scoped logger with common fields and timing information.
func logMiddleware(logger *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startTime := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			reqID := middleware.GetReqID(r.Context())

			// Create request-scoped logger
			reqLogger := logger.With(
				"requestId", reqID,
				"path", r.URL.Path,
				"method", r.Method,
				"remoteIP", r.RemoteAddr,
			)

			// Store logger in context for handlers
			ctx := context.WithValue(r.Context(), loggerKey{}, reqLogger)
			r = r.WithContext(ctx)

			defer func() {
				duration := time.Since(startTime)
				status := ww.Status()

				logFn := reqLogger.Info
				if status >= 500 {
					logFn = reqLogger.Error
				}

				logFn("http request",
					"status", status,
					"duration", duration,
					"size", ww.BytesWritten(),
				)
			}()

			next.ServeHTTP(ww, r)
		})
	}
}

// corsMiddleware implements secure CORS headers for enterprise environments.
// It follows security best practices while allowing necessary cross-origin access.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set secure CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", strings.Join([]string{
			"Authorization",
			"Content-Type",
			"Request-ID",
			"X-Request-ID",
		}, ", "))
		w.Header().Set("Access-Control-Max-Age", "3600")
		w.Header().Set("Access-Control-Expose-Headers", strings.Join([]string{
			"Request-ID",
			"X-Request-ID",
			"Retry-After",
			"RateLimit-Limit",
			"RateLimit-Remaining",
			"RateLimit-Reset",
		}, ", "))

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// requestIDHeaderMiddleware ensures request tracing through consistent ID propagation
func requestIDHeaderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetReqID(r.Context())
		if reqID != "" {
			w.Header().Set("Request-ID", reqID)
			w.Header().Set("X-Request-ID", reqID)
		}
		next.ServeHTTP(w, r)
	})
}

// recoverMiddleware provides graceful panic recovery with proper OAuth error responses
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

					// Return OAuth-compliant error for panics
					writeOAuthError(w, errServerError, "An unexpected error occurred", http.StatusInternalServerError, logger)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// rateLimitMiddleware enforces rate limits following OAuth 2.0 best practices.
// It provides standardized headers and error responses for rate limiting.
func rateLimitMiddleware(service ratelimit.Service, logger *slog.Logger, options ratelimit.RateLimitOptions) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		rateLimitNext := ratelimit.Middleware(service, logger, options)(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID := middleware.GetReqID(r.Context())
			reqLogger := logger.With("requestId", reqID)

			// Apply rate limiting with proper error handling
			defer func() {
				if rvr := recover(); rvr != nil {
					if limitErr, ok := rvr.(error); ok && errors.Is(limitErr, ratelimit.ErrLimitExceeded) {
						// Get limit details
						limit := service.GetLimit(options.LimitType)

						// Calculate remaining requests and reset time
						limitKey := ratelimit.LimitKey{
							Type:     options.LimitType,
							RemoteIP: r.RemoteAddr,
						}

						// Retrieve current status
						remaining, resetTime := 0, time.Now().Add(limit.Period)
						if counts, err := service.Status(limitKey); err == nil {
							remaining = counts.Remaining
							resetTime = counts.Reset
						}

						// Set standard rate limit headers
						w.Header().Set("RateLimit-Limit", fmt.Sprintf("%d", limit.Rate))
						w.Header().Set("RateLimit-Remaining", fmt.Sprintf("%d", remaining))
						w.Header().Set("RateLimit-Reset", fmt.Sprintf("%d", resetTime.Unix()))
						w.Header().Set("Retry-After", fmt.Sprintf("%d", int(time.Until(resetTime).Seconds())))

						// Return OAuth-compliant rate limit error
						writeOAuthError(w, errSlowDown, "Rate limit exceeded, please reduce request rate", http.StatusTooManyRequests, reqLogger)
						return
					}
					panic(rvr) // Re-panic if not a rate limit error
				}
			}()

			rateLimitNext.ServeHTTP(w, r)
		})
	}
}

// authMiddleware validates OAuth tokens and manages display authentication.
// It ensures proper error responses following OAuth 2.0 specifications.
func authMiddleware(authService auth.Service, logger *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID := middleware.GetReqID(r.Context())
			reqLogger := logger.With("requestId", reqID)

			// Extract and validate bearer token
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeOAuthError(w, errInvalidRequest, "Missing authorization header", http.StatusUnauthorized, reqLogger)
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				writeOAuthError(w, errInvalidRequest, "Invalid authorization format", http.StatusUnauthorized, reqLogger)
				return
			}

			// Validate access token
			displayID, err := authService.ValidateAccessToken(r.Context(), parts[1])
			if err != nil {
				var errCode, errDesc string
				if err == auth.ErrTokenExpired {
					errCode = errExpiredToken
					errDesc = "Access token has expired"
				} else {
					errCode = errInvalidRequest
					errDesc = "Invalid access token"
				}

				writeOAuthError(w, errCode, errDesc, http.StatusUnauthorized, reqLogger)
				reqLogger.Error("auth failed",
					"error", err,
					"remoteIP", r.RemoteAddr,
				)
				return
			}

			// Add display ID to request context
			ctx := context.WithValue(r.Context(), displayIDKey, displayID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
