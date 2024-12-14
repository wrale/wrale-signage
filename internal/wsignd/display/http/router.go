package http

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter creates a new HTTP router for display endpoints.
// Deprecated: Use Handler.Router() instead for consistent middleware and route configuration
func NewRouter(h *Handler) chi.Router {
	return h.Router()
}

// Router returns the HTTP router for display endpoints
func (h *Handler) Router() chi.Router {
	// Create base router with shared middleware
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(requestIDHeaderMiddleware)
	r.Use(middleware.RealIP)
	r.Use(recoverMiddleware(h.logger))
	r.Use(logMiddleware(h.logger))

	// Mount all display endpoints under /api/v1alpha1/displays
	r.Route("/api/v1alpha1/displays", func(r chi.Router) {
		// Initialize common rate limiters
		rateLimits := h.ratelimit.NewCommonRateLimits()

		// Health check endpoints (no rate limiting or auth)
		r.Get("/healthz", h.handleHealth())
		r.Get("/readyz", h.handleReady())

		// Public routes with request timeout
		r.Group(func(r chi.Router) {
			r.Use(middleware.Timeout(10 * time.Second))

			// Device activation flow with rate limiting
			r.With(rateLimits.DeviceCodeLimiter()).Post("/device/code", h.RequestDeviceCode)
			r.With(rateLimits.DeviceCodeLimiter()).Post("/activate", h.ActivateDeviceCode)
		})

		// Protected routes requiring authentication
		r.Group(func(r chi.Router) {
			r.Use(middleware.Timeout(30 * time.Second))
			r.Use(authMiddleware(h.auth, h.logger))
			r.Use(rateLimits.APIRequestLimiter())

			// Display management endpoints
			r.Get("/{id}", h.GetDisplay)
			r.Put("/{id}/activate", h.ActivateDisplay)
			r.Put("/{id}/last-seen", h.UpdateLastSeen)

			// Content events with dedicated rate limit
			r.With(rateLimits.ContentEventLimiter()).Post("/events", h.HandleContentEvents)

			// WebSocket endpoint with WebSocket-specific rate limit
			r.With(rateLimits.WebSocketLimiter()).Get("/ws", h.ServeWebSocket)
		})
	})

	return r
}

// handleHealth returns basic health check status
func (h *Handler) handleHealth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	}
}

// handleReady checks if the server is ready to accept requests
func (h *Handler) handleReady() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	}
}
