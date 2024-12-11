package http

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter creates a new HTTP router for display endpoints
func NewRouter(h *Handler) chi.Router {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	// API Routes v1alpha1
	r.Route("/api/v1alpha1/displays", func(r chi.Router) {
		// Display registration
		r.Post("/", h.RegisterDisplay)

		// Display management
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
