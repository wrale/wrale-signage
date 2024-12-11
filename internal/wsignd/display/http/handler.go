package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"log/slog"

	"github.com/wrale/wrale-signage/internal/wsignd/display"
	werrors "github.com/wrale/wrale-signage/internal/wsignd/errors"
)

// Handler implements HTTP handlers for display management
type Handler struct {
	service display.Service
	logger  *slog.Logger
	hub     *Hub
}

// NewHandler creates a new display HTTP handler
func NewHandler(service display.Service, logger *slog.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
		hub:     newHub(logger),
	}
}

func (h *Handler) writeError(w http.ResponseWriter, err error, status int) {
	var response struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}

	// Extract error details if available
	if werr, ok := err.(*werrors.Error); ok {
		response.Code = werr.Code
		response.Message = werr.Message
	} else {
		response.Code = "UNKNOWN"
		response.Message = err.Error()
	}

	h.logger.Error("request failed",
		"code", response.Code,
		"message", response.Message,
		"status", status,
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(response)
}

// GetDisplay handles requests to get display details
func (h *Handler) GetDisplay(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Log attempt to fetch display
	h.logger.Info("fetching display",
		"displayId", id,
		"remoteAddr", r.RemoteAddr,
	)

	// Attempt to fetch display
	display, err := h.service.Get(r.Context(), id)
	if err != nil {
		status := http.StatusInternalServerError
		if werrors.IsNotFound(err) {
			status = http.StatusNotFound
		}
		h.writeError(w, err, status)
		return
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(display)
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
