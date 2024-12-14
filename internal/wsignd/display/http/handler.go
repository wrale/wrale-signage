package http

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/wrale/wrale-signage/internal/wsignd/auth"
	"github.com/wrale/wrale-signage/internal/wsignd/display"
	"github.com/wrale/wrale-signage/internal/wsignd/display/activation"
	"github.com/wrale/wrale-signage/internal/wsignd/ratelimit"
)

// Handler encapsulates the HTTP API for display management
type Handler struct {
	service    display.Service
	activation activation.Service
	auth       auth.Service
	ratelimit  ratelimit.Service
	logger     *slog.Logger
}

// NewHandler creates a new HTTP handler for display endpoints
func NewHandler(
	service display.Service,
	activation activation.Service,
	auth auth.Service,
	ratelimit ratelimit.Service,
	logger *slog.Logger,
) *Handler {
	return &Handler{
		service:    service,
		activation: activation,
		auth:       auth,
		ratelimit:  ratelimit,
		logger:     logger,
	}
}

// Router returns the HTTP router for display endpoints
func (h *Handler) Router() http.Handler {
	r := chi.NewRouter()

	// Add common middleware
	r.Use(middleware.RequestID)
	r.Use(requestIDHeaderMiddleware)
	r.Use(middleware.RealIP)
	r.Use(recoverMiddleware(h.logger))
	r.Use(logMiddleware(h.logger))

	// Initialize common rate limiters
	rateLimits := ratelimit.NewCommonRateLimits(h.ratelimit, h.logger)

	// Public routes
	r.Group(func(r chi.Router) {
		r.Use(middleware.Timeout(10 * time.Second))

		// Health check endpoints (no rate limiting)
		r.Get("/healthz", h.handleHealth())
		r.Get("/readyz", h.handleReady())

		// Device activation flow (with device code rate limiting)
		deviceCodeGroup := chi.NewRouter()
		deviceCodeGroup.Use(rateLimits.DeviceCodeLimiter())
		deviceCodeGroup.Post("/device/code", h.RequestDeviceCode)
		deviceCodeGroup.Post("/activate", h.ActivateDeviceCode)
		r.Mount("/", deviceCodeGroup)
	})

	// Protected routes requiring authentication
	r.Group(func(r chi.Router) {
		r.Use(middleware.Timeout(30 * time.Second))
		r.Use(authMiddleware(h.auth, h.logger))
		r.Use(rateLimits.APIRequestLimiter())

		// Display management
		r.Get("/{displayID}", h.GetDisplay)
		r.Put("/{displayID}/activate", h.ActivateDisplay)
		r.Put("/{displayID}/last-seen", h.UpdateLastSeen)

		// Content event reporting (with specific rate limit)
		r.With(rateLimits.APIRequestLimiter()).Post("/events", h.HandleContentEvents)

		// WebSocket endpoint (with WebSocket-specific rate limit)
		r.With(rateLimits.WebSocketLimiter()).Get("/ws", h.ServeWebSocket)
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
