package http

import (
	"net/http"

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
		LimitType:   RateLimitTypeDeviceCode,
		WaitOnLimit: false,
		BurstSize:   1,
		Rate:        5,    // 5 requests
		Period:      3600, // per hour
	})
}

// rateLimitAPIRequest handles rate limiting for general API endpoints
func (h *Handler) rateLimitAPIRequest() func(http.Handler) http.Handler {
	return rateLimitMiddleware(h.ratelimit, h.logger, ratelimit.RateLimitOptions{
		LimitType:   RateLimitTypeAPIRequest,
		WaitOnLimit: true, // Queue requests during bursts
		BurstSize:   5,    // Allow small bursts
		Rate:        300,  // 300 requests
		Period:      3600, // per hour
		WaitTimeout: 5000, // Wait up to 5 seconds
	})
}

// rateLimitContentEvents handles rate limiting for content health monitoring
func (h *Handler) rateLimitContentEvents() func(http.Handler) http.Handler {
	return rateLimitMiddleware(h.ratelimit, h.logger, ratelimit.RateLimitOptions{
		LimitType:   RateLimitTypeContentEvents,
		WaitOnLimit: false,
		BurstSize:   10,   // Allow event bursts
		Rate:        3600, // 3600 events
		Period:      3600, // per hour (1/second average)
	})
}

// rateLimitWebSocket handles rate limiting for WebSocket connections
func (h *Handler) rateLimitWebSocket() func(http.Handler) http.Handler {
	return rateLimitMiddleware(h.ratelimit, h.logger, ratelimit.RateLimitOptions{
		LimitType:   RateLimitTypeWebSocket,
		WaitOnLimit: false,
		BurstSize:   1,    // No connection bursts
		Rate:        60,   // 60 connections
		Period:      3600, // per hour
	})
}
