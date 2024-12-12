package ratelimit

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// RateLimitOptions configures rate limiting behavior
type RateLimitOptions struct {
	// Type of rate limit to apply
	LimitType string

	// GetToken extracts token from request for authenticated limits
	GetToken func(r *http.Request) string

	// WaitOnLimit determines if requests should wait when limited
	WaitOnLimit bool

	// SkipLimitCheck determines if a request should bypass limits
	SkipLimitCheck func(r *http.Request) bool
}

// Middleware creates an HTTP middleware for rate limiting
func Middleware(service Service, logger *slog.Logger, options RateLimitOptions) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if we should skip rate limiting
			if options.SkipLimitCheck != nil && options.SkipLimitCheck(r) {
				next.ServeHTTP(w, r)
				return
			}

			// Build rate limit key
			key := buildKey(r, options)

			// Check rate limit
			if err := checkLimit(r, service, key, options, logger); err != nil {
				// Handle rate limit error
				handleLimitError(w, r, err, logger)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// buildKey creates a rate limit key from the request
func buildKey(r *http.Request, options RateLimitOptions) LimitKey {
	key := LimitKey{
		Type:     options.LimitType,
		RemoteIP: realIP(r),
		Endpoint: r.URL.Path,
	}

	// Add token if authentication extractor provided
	if options.GetToken != nil {
		key.Token = options.GetToken(r)
	}

	return key
}

// checkLimit verifies rate limit compliance
func checkLimit(r *http.Request, service Service, key LimitKey, options RateLimitOptions, logger *slog.Logger) error {
	limit := service.GetLimit(key.Type)

	// No limit configured = allowed
	if limit.Rate == 0 {
		return nil
	}

	// Check limit with retry if waiting is enabled
	startTime := time.Now()
	for {
		err := service.Allow(r.Context(), key)
		if err == nil {
			return nil
		}

		if !options.WaitOnLimit || limit.WaitTimeout == 0 {
			return err
		}

		// Check if we should keep waiting
		if time.Since(startTime) > limit.WaitTimeout {
			return err
		}

		// Wait before retry
		time.Sleep(100 * time.Millisecond)
	}
}

// handleLimitError writes an appropriate error response
func handleLimitError(w http.ResponseWriter, r *http.Request, err error, logger *slog.Logger) {
	if err == ErrLimitExceeded {
		logger.Warn("rate limit exceeded",
			"path", r.URL.Path,
			"method", r.Method,
			"remoteIP", realIP(r),
			"requestID", middleware.GetReqID(r.Context()),
		)

		w.Header().Set("Retry-After", "60") // Default 1 minute retry
		http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	// Handle other errors
	logger.Error("rate limit check failed",
		"error", err,
		"path", r.URL.Path,
		"method", r.Method,
		"remoteIP", realIP(r),
		"requestID", middleware.GetReqID(r.Context()),
	)
	http.Error(w, "internal error", http.StatusInternalServerError)
}

// realIP gets the real client IP, using X-Real-IP when available
func realIP(r *http.Request) string {
	// Check X-Real-IP header
	ip := r.Header.Get("X-Real-IP")
	if ip != "" {
		return ip
	}

	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if parts := strings.Split(xff, ","); len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}

	// Fall back to RemoteAddr
	return strings.Split(r.RemoteAddr, ":")[0]
}

// Helper functions for common rate limit scenarios
type CommonRateLimits struct {
	service Service
	logger  *slog.Logger
}

func NewCommonRateLimits(service Service, logger *slog.Logger) *CommonRateLimits {
	return &CommonRateLimits{
		service: service,
		logger:  logger,
	}
}

// TokenRefreshLimiter creates middleware for token refresh endpoint
func (c *CommonRateLimits) TokenRefreshLimiter() func(http.Handler) http.Handler {
	return Middleware(c.service, c.logger, RateLimitOptions{
		LimitType: "token_refresh",
		GetToken: func(r *http.Request) string {
			auth := r.Header.Get("Authorization")
			if strings.HasPrefix(auth, "Bearer ") {
				return auth[7:]
			}
			return ""
		},
		WaitOnLimit: false,
	})
}

// APIRequestLimiter creates middleware for general API endpoints
func (c *CommonRateLimits) APIRequestLimiter() func(http.Handler) http.Handler {
	return Middleware(c.service, c.logger, RateLimitOptions{
		LimitType: "api_request",
		GetToken: func(r *http.Request) string {
			auth := r.Header.Get("Authorization")
			if strings.HasPrefix(auth, "Bearer ") {
				return auth[7:]
			}
			return ""
		},
		WaitOnLimit: true,
		SkipLimitCheck: func(r *http.Request) bool {
			// Skip health checks and similar endpoints
			return strings.HasPrefix(r.URL.Path, "/healthz") ||
				strings.HasPrefix(r.URL.Path, "/readyz")
		},
	})
}

// DeviceCodeLimiter creates middleware for device code requests
func (c *CommonRateLimits) DeviceCodeLimiter() func(http.Handler) http.Handler {
	return Middleware(c.service, c.logger, RateLimitOptions{
		LimitType:   "device_code",
		WaitOnLimit: false,
	})
}

// WebSocketLimiter creates middleware for WebSocket connections
func (c *CommonRateLimits) WebSocketLimiter() func(http.Handler) http.Handler {
	return Middleware(c.service, c.logger, RateLimitOptions{
		LimitType: "ws_connection",
		GetToken: func(r *http.Request) string {
			auth := r.Header.Get("Authorization")
			if strings.HasPrefix(auth, "Bearer ") {
				return auth[7:]
			}
			return ""
		},
		WaitOnLimit: false,
	})
}
