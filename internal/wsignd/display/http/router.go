package http

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter creates a new HTTP router for display endpoints
func NewRouter(h *Handler) chi.Router {
	r := chi.NewRouter()

	// Middleware in dependency order
	r.Use(middleware.RequestID)
	r.Use(requestIDHeaderMiddleware)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(logMiddleware(h.logger))

	// API Routes v1alpha1
	r.Route("/api/v1alpha1/displays", func(r chi.Router) {
		// Device activation flow
		r.Post("/device/code", h.RequestDeviceCode)
		r.Post("/activate", h.ActivateDeviceCode)

		// Display registration and management
		r.Post("/", h.RegisterDisplay)
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", h.GetDisplay)
			r.Put("/activate", h.ActivateDisplay)
			r.Put("/last-seen", h.UpdateLastSeen)
		})

		// WebSocket control endpoint
		r.Get("/ws", h.ServeWs)
	})

	return r
}
