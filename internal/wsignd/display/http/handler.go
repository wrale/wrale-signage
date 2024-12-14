package http

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/wrale/wrale-signage/api/types/v1alpha1"
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
	hub        *Hub
}

// NewHandler creates a new HTTP handler for display endpoints
func NewHandler(
	service display.Service,
	activation activation.Service,
	auth auth.Service,
	ratelimit ratelimit.Service,
	logger *slog.Logger,
) *Handler {
	h := &Handler{
		service:    service,
		activation: activation,
		auth:       auth,
		ratelimit:  ratelimit,
		logger:     logger,
	}

	// Initialize hub
	h.hub = newHub(ratelimit, logger)
	go h.hub.run(context.Background())

	return h
}

// writeError writes an OAuth-compliant error response
func (h *Handler) writeError(w http.ResponseWriter, err error, defaultStatus int) {
	writeError(w, err, defaultStatus, h.logger)
}

// HandleContentEvents handles content playback and error events from displays
func (h *Handler) HandleContentEvents(w http.ResponseWriter, r *http.Request) {
	var events []v1alpha1.ContentEvent
	if err := json.NewDecoder(r.Body).Decode(&events); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Get authenticated display ID from context
	displayID, ok := GetDisplayID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// TODO: Process events through content service
	h.logger.Info("received content events",
		"displayId", displayID,
		"eventCount", len(events),
	)

	w.WriteHeader(http.StatusAccepted)
}

// ServeWebSocket upgrades the HTTP connection to WebSocket
func (h *Handler) ServeWebSocket(w http.ResponseWriter, r *http.Request) {
	h.ServeWs(w, r)
}

// handleHealth returns basic health check status
func (h *Handler) handleHealth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"status":"ok"}`)); err != nil {
			h.logger.Error("failed to write health response",
				"error", err,
				"path", r.URL.Path,
			)
		}
	}
}

// handleReady checks if the server is ready to accept requests
func (h *Handler) handleReady() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"status":"ok"}`)); err != nil {
			h.logger.Error("failed to write ready response",
				"error", err,
				"path", r.URL.Path,
			)
		}
	}
}
