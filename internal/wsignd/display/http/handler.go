package http

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/wrale/wrale-signage/internal/wsignd/auth"
	"github.com/wrale/wrale-signage/internal/wsignd/display"
	"github.com/wrale/wrale-signage/internal/wsignd/display/activation"
	"github.com/wrale/wrale-signage/internal/wsignd/ratelimit"
)

// Handler implements HTTP handlers for display management
type Handler struct {
	service    display.Service
	activation activation.Service
	auth       auth.Service
	rateLimit  ratelimit.Service
	logger     *slog.Logger
	hub        *Hub
}

// NewHandler creates a new display HTTP handler
func NewHandler(service display.Service, activation activation.Service, auth auth.Service, rateLimit ratelimit.Service, logger *slog.Logger) *Handler {
	h := &Handler{
		service:    service,
		activation: activation,
		auth:       auth,
		rateLimit:  rateLimit,
		logger:     logger,
	}
	h.hub = newHub(h.rateLimit, logger)
	go h.hub.run(context.Background()) // TODO: manage lifecycle with context
	return h
}

// Router returns a configured chi router for display endpoints
func (h *Handler) Router() *chi.Mux {
	r := chi.NewRouter()

	// Base middleware in dependency order
	r.Use(middleware.RequestID)
	r.Use(requestIDHeaderMiddleware)
	r.Use(middleware.RealIP)
	r.Use(recoverMiddleware(h.logger)) // Replace default recoverer with JSON version
	r.Use(logMiddleware(h.logger))

	// Create rate limit middleware groups
	rateLimits := ratelimit.NewCommonRateLimits(h.rateLimit, h.logger)

	// API Routes v1alpha1
	r.Route("/api/v1alpha1/displays", func(r chi.Router) {
		// Public endpoints with rate limits
		r.Group(func(r chi.Router) {
			// Initial display registration endpoint
			r.With(rateLimits.APIRequestLimiter()).Post("/", h.RegisterDisplay)

			// Device activation endpoints
			r.Group(func(r chi.Router) {
				r.Use(rateLimits.DeviceCodeLimiter())
				r.Post("/device/code", h.RequestDeviceCode)
				r.Post("/activate", h.ActivateDeviceCode)
			})

			// Token management
			r.Group(func(r chi.Router) {
				r.Use(rateLimits.TokenRefreshLimiter())
				r.Post("/token/refresh", h.RefreshToken)
			})
		})

		// Protected display management routes requiring authentication
		r.Group(func(r chi.Router) {
			// Order: Auth -> Rate Limit -> Routes
			r.Use(authMiddleware(h.auth, h.logger))
			r.Use(rateLimits.APIRequestLimiter())

			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", h.GetDisplay)
				r.Put("/activate", h.ActivateDisplay)
				r.Put("/last-seen", h.UpdateLastSeen)
			})

			// WebSocket endpoint with its own rate limit
			r.With(rateLimits.WebSocketLimiter()).Get("/ws", h.ServeWs)
		})

		// Health check endpoints bypass rate limits
		r.Group(func(r chi.Router) {
			r.Get("/healthz", h.healthCheck)
			r.Get("/readyz", h.readyCheck)
		})
	})

	return r
}

// writeJSON writes a JSON response, handling encoding errors
func writeJSON(w http.ResponseWriter, status int, v interface{}, logger *slog.Logger) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		logger.Error("failed to encode JSON response",
			"error", err,
		)
		// Don't try to write the error since we already wrote headers
	}
}

// healthCheck implements a basic health check
func (h *Handler) healthCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	}, h.logger)
}

// readyCheck implements a readiness check
func (h *Handler) readyCheck(w http.ResponseWriter, r *http.Request) {
	// Check dependencies
	status := http.StatusOK
	result := map[string]string{
		"status": "ok",
	}

	writeJSON(w, status, result, h.logger)
}
