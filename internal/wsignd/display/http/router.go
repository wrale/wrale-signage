package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Router creates and configures the HTTP router for display endpoints
func (h *Handler) Router() chi.Router {
	r := chi.NewRouter()

	// Basic middleware for all routes
	r.Use(middleware.RealIP)
	r.Use(middleware.RequestID)
	r.Use(requestIDHeaderMiddleware)
	r.Use(recoverMiddleware(h.logger))
	r.Use(logMiddleware(h.logger))

	// Public health check endpoints
	r.Group(func(r chi.Router) {
		r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"status":"ok"}`))
		})
		r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"status":"ok"}`))
		})
	})

	// API Routes
	r.Route("/api/v1alpha1/displays", func(r chi.Router) {
		// Public routes for device registration flow
		r.Group(func(r chi.Router) {
			r.Use(h.rateLimiter.Common.DeviceCodeLimiter.Middleware)

			r.Post("/device/code", h.RequestDeviceCode)
			r.Post("/activate", h.ActivateDeviceCode)
		})

		// Protected routes requiring authentication
		r.Group(func(r chi.Router) {
			// Rate limiting before auth to prevent token validation overload
			r.Use(h.rateLimiter.Common.DisplayLimiter.Middleware)
			r.Use(authMiddleware(h.auth, h.logger))

			// Display management
			r.Get("/{id}", h.GetDisplay)
			r.Put("/{id}/activate", h.ActivateDisplay)
			r.Put("/{id}/last-seen", h.UpdateLastSeen)

			// WebSocket connection
			r.Get("/ws", h.HandleWebSocket)
		})

		// 404 Handler
		r.NotFound(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"not found"}`))
		})
	})

	return r
}

// NewRouter creates a new HTTP router for display endpoints
// Deprecated: Use Handler.Router() instead for consistent middleware and route configuration
func NewRouter(h *Handler) chi.Router {
	return h.Router()
}
