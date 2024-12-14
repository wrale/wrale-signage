package http

import (
	"net/http"
	"time"

	"github.com/wrale/wrale-signage/internal/wsignd/ratelimit"
)

// All rate limit types used by the display HTTP handlers
const (
	RateLimitTypeDeviceCode    = "device_code"    // Device activation flow
	RateLimitTypeAPIRequest    = "api_request"    // General API requests
	RateLimitTypeContentEvents = "content_events" // Content health monitoring
	RateLimitTypeWebSocket     = "ws_connection"  // WebSocket connections
)

// rateLimitDeviceCode handles rate limiting for device activation endpoints
func (h *Handler) rateLimitDeviceCode() func(http.Handler) http.Handler {
	return rateLimitMiddleware(h.ratelimit, h.logger, ratelimit.RateLimitOptions{
		LimitType:      RateLimitTypeDeviceCode,
		WaitOnLimit:    false,
		WaitTimeout:    0,
		SkipLimitCheck: nil,
	})
}

// rateLimitAPIRequest handles rate limiting for general API endpoints
func (h *Handler) rateLimitAPIRequest() func(http.Handler) http.Handler {
	return rateLimitMiddleware(h.ratelimit, h.logger, ratelimit.RateLimitOptions{
		LimitType:   RateLimitTypeAPIRequest,
		WaitOnLimit: true,
		WaitTimeout: 5 * time.Second,
		SkipLimitCheck: func(r *http.Request) bool {
			// Skip health checks and metrics endpoints
			return r.URL.Path == "/healthz" || r.URL.Path == "/metrics"
		},
	})
}

// rateLimitContentEvents handles rate limiting for content health monitoring
func (h *Handler) rateLimitContentEvents() func(http.Handler) http.Handler {
	return rateLimitMiddleware(h.ratelimit, h.logger, ratelimit.RateLimitOptions{
		LimitType:      RateLimitTypeContentEvents,
		WaitOnLimit:    false,
		WaitTimeout:    0,
		SkipLimitCheck: nil,
	})
}

// rateLimitWebSocket handles rate limiting for WebSocket connections
func (h *Handler) rateLimitWebSocket() func(http.Handler) http.Handler {
	return rateLimitMiddleware(h.ratelimit, h.logger, ratelimit.RateLimitOptions{
		LimitType:      RateLimitTypeWebSocket,
		WaitOnLimit:    false,
		WaitTimeout:    0,
		SkipLimitCheck: nil,
		GetToken: func(r *http.Request) string {
			auth := r.Header.Get("Authorization")
			if len(auth) > 7 && auth[:7] == "Bearer " {
				return auth[7:]
			}
			return ""
		},
	})
}
