package http

import (
	"context"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"log/slog"

	"github.com/wrale/wrale-signage/internal/wsignd/auth"
	"github.com/wrale/wrale-signage/internal/wsignd/display"
	"github.com/wrale/wrale-signage/internal/wsignd/display/activation"
)

// Handler implements HTTP handlers for display management
type Handler struct {
	service    display.Service
	activation activation.Service
	auth       auth.Service
	logger     *slog.Logger
	hub        *Hub
}

// NewHandler creates a new display HTTP handler
func NewHandler(service display.Service, activation activation.Service, auth auth.Service, logger *slog.Logger) *Handler {
	h := &Handler{
		service:    service,
		activation: activation,
		auth:       auth,
		logger:     logger,
	}
	h.hub = newHub(logger)
	go h.hub.run(context.Background()) // TODO: manage lifecycle with context
	return h
}

// Router returns a configured chi router for display endpoints
func (h *Handler) Router() *chi.Mux {
	r := chi.NewRouter()

	// Middleware in dependency order
	r.Use(middleware.RequestID)
	r.Use(requestIDHeaderMiddleware)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(logMiddleware(h.logger))

	// API Routes v1alpha1
	r.Route("/api/v1alpha1/displays", func(r chi.Router) {
		// Public device activation flow
		r.Post("/device/code", h.RequestDeviceCode)
		r.Post("/activate", h.ActivateDeviceCode)

		// Protected display management routes
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware(h.auth, h.logger))

			r.Post("/", h.RegisterDisplay)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", h.GetDisplay)
				r.Put("/activate", h.ActivateDisplay)
				r.Put("/last-seen", h.UpdateLastSeen)
			})
			r.Get("/ws", h.ServeWs)
		})

		// Token management
		r.Post("/token/refresh", h.RefreshToken)
	})

	return r
}
