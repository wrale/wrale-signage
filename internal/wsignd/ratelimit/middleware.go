package ratelimit

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// Middleware creates an HTTP middleware for rate limiting.
// It enforces rate limits while providing standard rate limit headers
// and proper error responses following RFC 6585 and RFC 7231.
func Middleware(service Service, logger *slog.Logger, options RateLimitOptions) func(http.Handler) http.Handler {
	// Initialize a secure random source for jitter calculations
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract request context for logging
			reqID := middleware.GetReqID(r.Context())
			reqLogger := logger.With("requestId", reqID)

			// Check if we should bypass rate limiting
			if options.SkipLimitCheck != nil && options.SkipLimitCheck(r) {
				next.ServeHTTP(w, r)
				return
			}

			// Build the rate limit key that uniquely identifies this request
			key := buildKey(r, options)

			// Get current rate limit status
			status, err := service.Status(key)
			if err != nil {
				reqLogger.Error("failed to get rate limit status",
					"error", err,
					"type", options.LimitType,
					"path", r.URL.Path,
				)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			// Set standard rate limit headers (RFC 6585)
			setRateLimitHeaders(w, status)

			// Check if we're already over the limit
			if status.Remaining <= 0 {
				// If waiting is enabled and we have a timeout, attempt to wait
				if shouldWait(options, status.Limit) {
					if err := waitForCapacity(r.Context(), service, key, options, reqLogger, rnd); err != nil {
						handleLimitExceeded(w, r, status, reqLogger)
						return
					}
					// Successfully waited for capacity
					next.ServeHTTP(w, r)
					return
				}

				// No waiting or wait failed - return rate limit exceeded
				handleLimitExceeded(w, r, status, reqLogger)
				return
			}

			// We have capacity - allow the request
			next.ServeHTTP(w, r)
		})
	}
}

// buildKey creates a rate limit key from the request, incorporating
// all relevant information for proper limit tracking.
func buildKey(r *http.Request, options RateLimitOptions) LimitKey {
	key := LimitKey{
		Type:     options.LimitType,
		RemoteIP: realIP(r),
		Endpoint: r.URL.Path,
	}

	// Add authentication token if available
	if options.GetToken != nil {
		key.Token = options.GetToken(r)
	}

	return key
}

// setRateLimitHeaders adds standard rate limit headers to the response
// following RFC 6585 (Too Many Requests) and RFC 7231 (HTTP/1.1).
func setRateLimitHeaders(w http.ResponseWriter, status *LimitStatus) {
	w.Header().Set("RateLimit-Limit", strconv.Itoa(status.Limit.Rate))
	w.Header().Set("RateLimit-Remaining", strconv.Itoa(status.Remaining))
	w.Header().Set("RateLimit-Reset", strconv.FormatInt(status.Reset.Unix(), 10))

	// Add informational headers about burst capacity
	if status.Limit.BurstSize > 0 {
		w.Header().Set("RateLimit-Burst", strconv.Itoa(status.Limit.BurstSize))
	}
}

// handleLimitExceeded sends a proper 429 Too Many Requests response
// with appropriate headers and a helpful error message.
func handleLimitExceeded(w http.ResponseWriter, r *http.Request, status *LimitStatus, logger *slog.Logger) {
	// Calculate retry delay
	retryAfter := int(time.Until(status.Reset).Seconds())
	if retryAfter < 1 {
		retryAfter = 1 // Minimum 1 second retry delay
	}

	// Log the rate limit event
	logger.Warn("rate limit exceeded",
		"path", r.URL.Path,
		"method", r.Method,
		"remoteIP", realIP(r),
		"retryAfter", retryAfter,
	)

	// Set required response headers
	w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusTooManyRequests)

	// Send helpful error response
	fmt.Fprintf(w, `{"error":"rate_limit_exceeded","message":"Too many requests, please retry after %d seconds"}`, retryAfter)
}

// shouldWait determines if the middleware should wait for rate limit capacity
// based on configuration and current state.
func shouldWait(options RateLimitOptions, limit Limit) bool {
	if !options.WaitOnLimit {
		return false
	}

	// Use option timeout if specified, otherwise use limit timeout
	timeout := limit.WaitTimeout
	if options.WaitTimeout > 0 {
		timeout = options.WaitTimeout
	}

	return timeout > 0
}

// waitForCapacity attempts to wait for rate limit capacity to become available.
// It uses exponential backoff with jitter to prevent thundering herd problems.
func waitForCapacity(ctx context.Context, service Service, key LimitKey, options RateLimitOptions, logger *slog.Logger, rnd *rand.Rand) error {
	timeout := options.WaitTimeout
	if timeout == 0 {
		timeout = service.GetLimit(key.Type).WaitTimeout
	}

	startTime := time.Now()
	backoff := 100 * time.Millisecond // Start with 100ms backoff
	maxBackoff := 1 * time.Second     // Cap backoff at 1 second

	for {
		// Check if we've exceeded our wait timeout or context was canceled
		select {
		case <-ctx.Done():
			return fmt.Errorf("context canceled while waiting for capacity: %w", ctx.Err())
		default:
			if time.Since(startTime) > timeout {
				return fmt.Errorf("timeout waiting for rate limit capacity")
			}
		}

		// Check if we have capacity
		if err := service.Allow(ctx, key); err == nil {
			return nil // Successfully acquired capacity
		}

		// Add jitter to prevent thundering herd
		jitter := time.Duration(float64(backoff) * (0.5 + rnd.Float64())) // ±50% jitter
		time.Sleep(jitter)

		// Exponential backoff
		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

// realIP extracts the real client IP address using standard headers
// and following best practices for IP extraction.
func realIP(r *http.Request) string {
	// Check X-Real-IP header first
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}

	// Try X-Forwarded-For header (using leftmost value)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if parts := strings.Split(xff, ","); len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}

	// Fall back to RemoteAddr, strip port if present
	host := r.RemoteAddr
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		host = host[:idx]
	}

	return host
}

// CommonRateLimiters provides pre-configured rate limit middleware
// for standard API endpoints. This ensures consistent rate limiting
// across the application.
type CommonRateLimiters struct {
	service Service
	logger  *slog.Logger
}

// NewCommonRateLimiters creates a provider of standard rate limiters
func NewCommonRateLimiters(service Service, logger *slog.Logger) *CommonRateLimiters {
	return &CommonRateLimiters{
		service: service,
		logger:  logger,
	}
}

// TokenRefreshLimiter creates middleware for token refresh endpoints.
// This uses stricter limits to prevent token grinding attacks.
func (c *CommonRateLimiters) TokenRefreshLimiter() func(http.Handler) http.Handler {
	return Middleware(c.service, c.logger, RateLimitOptions{
		LimitType: "token_refresh",
		GetToken: func(r *http.Request) string {
			auth := r.Header.Get("Authorization")
			if strings.HasPrefix(auth, "Bearer ") {
				return auth[7:]
			}
			return ""
		},
		WaitOnLimit: false, // Don't wait for token operations
	})
}

// APIRequestLimiter creates middleware for general API endpoints.
// This provides reasonable limits with burst capacity for normal operations.
func (c *CommonRateLimiters) APIRequestLimiter() func(http.Handler) http.Handler {
	return Middleware(c.service, c.logger, RateLimitOptions{
		LimitType: "api_request",
		GetToken: func(r *http.Request) string {
			auth := r.Header.Get("Authorization")
			if strings.HasPrefix(auth, "Bearer ") {
				return auth[7:]
			}
			return ""
		},
		WaitOnLimit: true, // Allow waiting for API requests
		SkipLimitCheck: func(r *http.Request) bool {
			// Skip health checks and monitoring endpoints
			return strings.HasPrefix(r.URL.Path, "/healthz") ||
				strings.HasPrefix(r.URL.Path, "/readyz") ||
				strings.HasPrefix(r.URL.Path, "/metrics")
		},
	})
}

// DeviceCodeLimiter creates middleware for device activation endpoints.
// This enforces strict limits to prevent device code enumeration attacks.
func (c *CommonRateLimiters) DeviceCodeLimiter() func(http.Handler) http.Handler {
	return Middleware(c.service, c.logger, RateLimitOptions{
		LimitType:   "device_code",
		WaitOnLimit: false, // Don't wait for security-sensitive operations
	})
}

// WebSocketLimiter creates middleware for WebSocket connections.
// This manages connection rate without affecting message throughput.
func (c *CommonRateLimiters) WebSocketLimiter() func(http.Handler) http.Handler {
	return Middleware(c.service, c.logger, RateLimitOptions{
		LimitType: "ws_connection",
		GetToken: func(r *http.Request) string {
			auth := r.Header.Get("Authorization")
			if strings.HasPrefix(auth, "Bearer ") {
				return auth[7:]
			}
			return ""
		},
		WaitOnLimit: false, // Don't wait for WebSocket connections
	})
}
