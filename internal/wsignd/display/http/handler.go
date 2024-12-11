// Package http provides HTTP handlers for the display service
package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/wrale/wrale-signage/api/types/v1alpha1"
	"github.com/wrale/wrale-signage/internal/wsignd/display"
)

// Handler implements HTTP handlers for display management
type Handler struct {
	service display.Service
	logger  *slog.Logger
}

// NewHandler creates a new display HTTP handler
func NewHandler(service display.Service, logger *slog.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// RegisterDisplay handles display registration requests
func (h *Handler) RegisterDisplay(w http.ResponseWriter, r *http.Request) {
	var req v1alpha1.DisplayRegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Convert API types to domain types
	location := display.Location{
		SiteID:   req.Location.SiteID,
		Zone:     req.Location.Zone,
		Position: req.Location.Position,
	}

	// Register display through service
	d, err := h.service.Register(r.Context(), req.Name, location)
	if err != nil {
		h.logger.Error("failed to register display",
			"error", err,
			"name", req.Name,
		)
		http.Error(w, "registration failed", http.StatusInternalServerError)
		return
	}

	// Convert domain type to API response
	resp := &v1alpha1.DisplayRegistrationResponse{
		Display: &v1alpha1.Display{
			TypeMeta: v1alpha1.TypeMeta{
				Kind:       "Display",
				APIVersion: "v1alpha1",
			},
			ObjectMeta: v1alpha1.ObjectMeta{
				ID:   d.ID,
				Name: d.Name,
			},
			Spec: v1alpha1.DisplaySpec{
				Location: v1alpha1.DisplayLocation{
					SiteID:   d.Location.SiteID,
					Zone:     d.Location.Zone,
					Position: d.Location.Position,
				},
				Properties: d.Properties,
			},
			Status: v1alpha1.DisplayStatus{
				State:    v1alpha1.DisplayState(d.State),
				LastSeen: d.LastSeen,
				Version:  d.Version,
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("failed to encode response",
			"error", err,
		)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
}

// GetDisplay handles requests to get display status
func (h *Handler) GetDisplay(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid display ID", http.StatusBadRequest)
		return
	}

	d, err := h.service.Get(r.Context(), id)
	if err != nil {
		h.logger.Error("failed to get display",
			"error", err,
			"id", id,
		)
		http.Error(w, "display not found", http.StatusNotFound)
		return
	}

	// Convert to API type
	resp := &v1alpha1.Display{
		TypeMeta: v1alpha1.TypeMeta{
			Kind:       "Display",
			APIVersion: "v1alpha1",
		},
		ObjectMeta: v1alpha1.ObjectMeta{
			ID:   d.ID,
			Name: d.Name,
		},
		Spec: v1alpha1.DisplaySpec{
			Location: v1alpha1.DisplayLocation{
				SiteID:   d.Location.SiteID,
				Zone:     d.Location.Zone,
				Position: d.Location.Position,
			},
			Properties: d.Properties,
		},
		Status: v1alpha1.DisplayStatus{
			State:    v1alpha1.DisplayState(d.State),
			LastSeen: d.LastSeen,
			Version:  d.Version,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("failed to encode response",
			"error", err,
		)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
}

// ActivateDisplay handles display activation requests
func (h *Handler) ActivateDisplay(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid display ID", http.StatusBadRequest)
		return
	}

	if err := h.service.Activate(r.Context(), id); err != nil {
		h.logger.Error("failed to activate display",
			"error", err,
			"id", id,
		)
		http.Error(w, "activation failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// UpdateLastSeen updates the display's last seen timestamp
func (h *Handler) UpdateLastSeen(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid display ID", http.StatusBadRequest)
		return
	}

	if err := h.service.UpdateLastSeen(r.Context(), id); err != nil {
		h.logger.Error("failed to update last seen",
			"error", err,
			"id", id,
		)
		http.Error(w, "update failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
