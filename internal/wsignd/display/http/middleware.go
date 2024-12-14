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

// logMiddleware provides structured logging for all HTTP requests
func logMiddleware(logger *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startTime := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			reqID := middleware.GetReqID(r.Context())

			// Create request-scoped logger with ID and path
			reqLogger := logger.With(
				"requestId", reqID,
				"path", r.URL.Path,
				"method", r.Method,
				"remoteIP", r.RemoteAddr,
			)

			// Store logger in context for handlers
			ctx := context.WithValue(r.Context(), struct{}{}, reqLogger)
			r = r.WithContext(ctx)

			defer func() {
				duration := time.Since(startTime)
				status := ww.Status()

				// Log at appropriate level
				logFn := reqLogger.Info
				if status >= 500 {
					logFn = reqLogger.Error
				}

				// Log request outcome
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

// corsMiddleware implements secure CORS headers following best practices
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set secure CORS headers with restrictive defaults
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", strings.Join([]string{
			"Authorization",
			"Content-Type",
			"Request-ID",
			"X-Request-ID",
		}, ", "))
		w.Header().Set("Access-Control-Max-Age", "3600") // Cache preflight for 1 hour
		w.Header().Set("Access-Control-Expose-Headers", strings.Join([]string{
			"Request-ID",
			"X-Request-ID",
			"Retry-After",
			"RateLimit-Limit",
			"RateLimit-Remaining",
			"RateLimit-Reset",
		}, ", "))

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// requestIDHeaderMiddleware ensures consistent request ID propagation
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

// recoverMiddleware handles panics and returns standardized error responses
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

					// Write OAuth-compliant error response
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)

					response := struct {
						Error            string `json:"error"`
						ErrorDescription string `json:"error_description"`
					}{
						Error:            "server_error",
						ErrorDescription: "An internal error occurred",
					}

					if err := json.NewEncoder(w).Encode(response); err != nil {
						logger.Error("failed to write error response",
							"error", err,
							"originalError", rvr,
						)
					}
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// rateLimitMiddleware provides OAuth-compliant rate limiting responses
func rateLimitMiddleware(service ratelimit.Service, logger *slog.Logger, options ratelimit.RateLimitOptions) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		rateLimitNext := ratelimit.Middleware(service, logger, options)(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID := middleware.GetReqID(r.Context())
			reqLogger := logger.With("requestId", reqID)

			// Apply rate limiting
			defer func() {
				if rvr := recover(); rvr != nil {
					if limitErr, ok := rvr.(error); ok && errors.Is(limitErr, ratelimit.ErrLimitExceeded) {
						// Get rate limit details
						limit := service.GetLimit(options.LimitType)
						remaining, reset := service.Status(ratelimit.LimitKey{
							Type:     options.LimitType,
							RemoteIP: r.RemoteAddr,
						})

						// Set standard rate limit headers
						w.Header().Set("RateLimit-Limit", fmt.Sprintf("%d", limit.Rate))
						w.Header().Set("RateLimit-Remaining", fmt.Sprintf("%d", remaining))
						w.Header().Set("RateLimit-Reset", fmt.Sprintf("%d", reset.Unix()))
						w.Header().Set("Retry-After", fmt.Sprintf("%d", int(time.Until(reset).Seconds())))

						// Write OAuth error response
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusTooManyRequests)

						response := struct {
							Error            string `json:"error"`
							ErrorDescription string `json:"error_description"`
						}{
							Error:            "slow_down",
							ErrorDescription: "Too many requests, please try again later",
						}

						if err := json.NewEncoder(w).Encode(response); err != nil {
							reqLogger.Error("failed to write rate limit response",
								"error", err,
							)
						}
						return
					}
					panic(rvr) // Re-panic if not a rate limit error
				}
			}()

			rateLimitNext.ServeHTTP(w, r)
		})
	}
}

// authMiddleware validates OAuth tokens and manages display authentication
func authMiddleware(authService auth.Service, logger *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID := middleware.GetReqID(r.Context())
			reqLogger := logger.With("requestId", reqID)

			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeOAuthError(w, "invalid_request", "Missing authorization header", http.StatusUnauthorized, reqLogger)
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				writeOAuthError(w, "invalid_request", "Invalid authorization format", http.StatusUnauthorized, reqLogger)
				return
			}

			// Validate access token
			displayID, err := authService.ValidateAccessToken(r.Context(), parts[1])
			if err != nil {
				var errCode, errMsg string
				if err == auth.ErrTokenExpired {
					errCode = "invalid_token"
					errMsg = "Access token has expired"
				} else {
					errCode = "invalid_token"
					errMsg = "Invalid access token"
				}

				writeOAuthError(w, errCode, errMsg, http.StatusUnauthorized, reqLogger)
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

// writeOAuthError writes a standardized OAuth 2.0 error response
func writeOAuthError(w http.ResponseWriter, code string, description string, status int, logger *slog.Logger) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.WriteHeader(status)

	response := struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description,omitempty"`
	}{
		Error:            code,
		ErrorDescription: description,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Error("failed to write error response",
			"error", err,
			"originalError", code,
		)
	}
}
