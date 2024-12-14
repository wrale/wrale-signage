package http

import (
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

const (
	// Default timeouts
	publicTimeout  = 10 * time.Second
	privateTimeout = 30 * time.Second
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

	// Add base middleware in correct order
	r.Use(middleware.RequestID)
	r.Use(requestIDHeaderMiddleware)
	r.Use(middleware.RealIP)
	r.Use(recoverMiddleware(h.logger))
	r.Use(logMiddleware(h.logger))

	// Mount all display endpoints under /api/v1alpha1/displays
	r.Route("/api/v1alpha1/displays", func(r chi.Router) {
		// Health check endpoints (no rate limiting or auth)
		r.Group(func(r chi.Router) {
			r.Get("/healthz", h.handleHealth())
			r.Get("/readyz", h.handleReady())
		})

		// Device activation flow (public endpoints with rate limiting)
		r.Group(func(r chi.Router) {
			r.Use(middleware.Timeout(publicTimeout))
			r.Use(h.rateLimitDeviceCode())

			r.Post("/device/code", h.RequestDeviceCode)
			r.Post("/activate", h.ActivateDeviceCode)
		})

		// Protected routes requiring authentication
		r.Group(func(r chi.Router) {
			r.Use(middleware.Timeout(privateTimeout))
			r.Use(authMiddleware(h.auth, h.logger))
			r.Use(h.rateLimitAPIRequest())

			// Display management endpoints
			r.Get("/{id}", h.GetDisplay)
			r.Put("/{id}/activate", h.ActivateDisplay)
			r.Put("/{id}/last-seen", h.UpdateLastSeen)

			// Content events (separate rate limit)
			r.With(h.rateLimitContentEvents()).Post("/events", h.HandleContentEvents)

			// WebSocket endpoint (separate rate limit)
			r.With(h.rateLimitWebSocket()).Get("/ws", h.ServeWebSocket)
		})
	})

	return r
}
