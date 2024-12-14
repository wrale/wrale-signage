package http

import (
	"net/http"

	"github.com/wrale/wrale-signage/internal/wsignd/ratelimit"
)

// Rate limit middleware helpers

// rateLimitDeviceCode returns rate limit middleware for device code endpoints
func (h *Handler) rateLimitDeviceCode() func(http.Handler) http.Handler {
	return rateLimitMiddleware(h.ratelimit, h.logger, ratelimit.RateLimitOptions{
		LimitType:   "device_code",
		WaitOnLimit: false,
	})
}

// rateLimitAPIRequest returns rate limit middleware for general API endpoints
func (h *Handler) rateLimitAPIRequest() func(http.Handler) http.Handler {
	return rateLimitMiddleware(h.ratelimit, h.logger, ratelimit.RateLimitOptions{
		LimitType:   "api_request",
		WaitOnLimit: true,
	})
}

// rateLimitContentEvents returns rate limit middleware for content events
func (h *Handler) rateLimitContentEvents() func(http.Handler) http.Handler {
	return rateLimitMiddleware(h.ratelimit, h.logger, ratelimit.RateLimitOptions{
		LimitType:   "content_events",
		WaitOnLimit: false,
	})
}

// rateLimitWebSocket returns rate limit middleware for WebSocket connections
func (h *Handler) rateLimitWebSocket() func(http.Handler) http.Handler {
	return rateLimitMiddleware(h.ratelimit, h.logger, ratelimit.RateLimitOptions{
		LimitType:   "ws_connection",
		WaitOnLimit: false,
	})
}
