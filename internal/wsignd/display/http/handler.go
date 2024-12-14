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

// Rate limit configurations
type CommonRateLimits struct {
	APIRequestLimiter   ratelimit.Limiter
	DeviceCodeLimiter   ratelimit.Limiter
	ContentEventLimiter ratelimit.Limiter
}

// newCommonRateLimits creates standard rate limiters
func (h *Handler) newCommonRateLimits() (*CommonRateLimits, error) {
	// API requests: 100/minute with burst of 5
	apiLimiter, err := h.ratelimit.GetLimit("api_request", &ratelimit.Config{
		Rate:  100,
		Burst: 5,
		TTL:   time.Minute,
	})
	if err != nil {
		return nil, err
	}

	// Device activation: 5/minute
	deviceLimiter, err := h.ratelimit.GetLimit("device_activation", &ratelimit.Config{
		Rate:  5,
		Burst: 1,
		TTL:   time.Minute,
	})
	if err != nil {
		return nil, err
	}

	// Content events: 1000/minute with burst of 100
	contentLimiter, err := h.ratelimit.GetLimit("content_events", &ratelimit.Config{
		Rate:  1000,
		Burst: 100,
		TTL:   time.Minute,
	})
	if err != nil {
		return nil, err
	}

	return &CommonRateLimits{
		APIRequestLimiter:   apiLimiter,
		DeviceCodeLimiter:   deviceLimiter,
		ContentEventLimiter: contentLimiter,
	}, nil
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

	// Setup rate limiters
	limits, err := h.newCommonRateLimits()
	if err != nil {
		panic(err) // Rate limits are required for operation
	}

	// Public routes
	r.Group(func(r chi.Router) {
		r.Use(middleware.Timeout(10 * time.Second))

		// Health check endpoints
		r.Get("/healthz", h.handleHealth())
		r.Get("/readyz", h.handleReady())

		// Device activation flow
		r.With(limits.DeviceCodeLimiter.Middleware).Post("/device/code", h.RequestDeviceCode)
		r.With(limits.DeviceCodeLimiter.Middleware).Post("/activate", h.ActivateDeviceCode)
	})

	// Protected routes requiring authentication
	r.Group(func(r chi.Router) {
		r.Use(middleware.Timeout(30 * time.Second))
		r.Use(authMiddleware(h.auth, h.logger))
		r.Use(limits.APIRequestLimiter.Middleware)

		// Display management
		r.Get("/{displayID}", h.GetDisplay)
		r.Put("/{displayID}/activate", h.ActivateDisplay)
		r.Put("/{displayID}/last-seen", h.UpdateLastSeen)

		// Content event reporting
		r.With(limits.ContentEventLimiter.Middleware).Post("/events", h.HandleContentEvents)

		// WebSocket endpoint for display control
		r.Get("/ws", h.ServeWebSocket)
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
