package http

import (
	"context"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"log/slog"

	"github.com/wrale/wrale-signage/internal/wsignd/display"
)

// Handler implements HTTP handlers for display management
type Handler struct {
	service display.Service
	logger  *slog.Logger
	hub     *Hub
}

// NewHandler creates a new display HTTP handler
func NewHandler(service display.Service, logger *slog.Logger) *Handler {
	h := &Handler{
		service: service,
		logger:  logger,
	}
	h.hub = newHub(logger)
	go h.hub.run(context.Background()) // TODO: manage lifecycle with context
	return h
}

// Router returns a configured chi router for display endpoints
func (h *Handler) Router() *chi.Mux {
	r := chi.NewRouter()

	// Add our middleware
	r.Use(logMiddleware(h.logger))   // Our structured logging
	r.Use(middleware.RequestID)      // Generates request IDs
	r.Use(middleware.RealIP)         // Uses X-Forwarded-For if present
	r.Use(middleware.Recoverer)      // Recovers from panics
	r.Use(requestIDHeaderMiddleware) // Ensures request ID in response

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

		// WebSocket endpoint
		r.Get("/ws", h.ServeWs)
	})

	return r
}
