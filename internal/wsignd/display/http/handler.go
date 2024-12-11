// Package http provides HTTP handlers for the display service
package http

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/wrale/wrale-signage/api/types/v1alpha1"
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
	h := &Handler{
		service: service,
		logger:  logger,
	}
	h.hub = newHub(logger)
	go h.hub.run(context.Background()) // TODO: manage lifecycle with context
	return h
}

// writeError writes a JSON error response
func (h *Handler) writeError(w http.ResponseWriter, err error, defaultStatus int) {
	var werr *werrors.Error
	if errors.As(err, &werr) {
		status := defaultStatus
		switch werr.Code {
		case "NOT_FOUND":
			status = http.StatusNotFound
		case "CONFLICT":
			status = http.StatusConflict
		case "INVALID_INPUT":
			status = http.StatusBadRequest
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(map[string]string{
			"code":    werr.Code,
			"message": werr.Message,
		})
		return
	}

	// Default error response
	http.Error(w, "internal server error", defaultStatus)
}

// RegisterDisplay handles display registration requests
func (h *Handler) RegisterDisplay(w http.ResponseWriter, r *http.Request) {
	var req v1alpha1.DisplayRegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, werrors.NewError("INVALID_INPUT", "invalid request body", "RegisterDisplay", err), http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Name == "" {
		h.writeError(w, werrors.NewError("INVALID_INPUT", "display name is required", "RegisterDisplay", nil), http.StatusBadRequest)
		return
	}
	if req.Location.SiteID == "" {
		h.writeError(w, werrors.NewError("INVALID_INPUT", "site ID is required", "RegisterDisplay", nil), http.StatusBadRequest)
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
		h.writeError(w, err, http.StatusInternalServerError)
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
		h.writeError(w, werrors.NewError("INTERNAL", "failed to encode response", "RegisterDisplay", err), http.StatusInternalServerError)
		return
	}
}

// GetDisplay handles requests to get display status
func (h *Handler) GetDisplay(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.writeError(w, werrors.NewError("INVALID_INPUT", "invalid display ID", "GetDisplay", err), http.StatusBadRequest)
		return
	}

	d, err := h.service.Get(r.Context(), id)
	if err != nil {
		h.logger.Error("failed to get display",
			"error", err,
			"id", id,
		)
		h.writeError(w, err, http.StatusInternalServerError)
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
		h.writeError(w, werrors.NewError("INTERNAL", "failed to encode response", "GetDisplay", err), http.StatusInternalServerError)
		return
	}
}

// ActivateDisplay handles display activation requests
func (h *Handler) ActivateDisplay(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.writeError(w, werrors.NewError("INVALID_INPUT", "invalid display ID", "ActivateDisplay", err), http.StatusBadRequest)
		return
	}

	if err := h.service.Activate(r.Context(), id); err != nil {
		h.logger.Error("failed to activate display",
			"error", err,
			"id", id,
		)
		h.writeError(w, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// UpdateLastSeen updates the display's last seen timestamp
func (h *Handler) UpdateLastSeen(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.writeError(w, werrors.NewError("INVALID_INPUT", "invalid display ID", "UpdateLastSeen", err), http.StatusBadRequest)
		return
	}

	if err := h.service.UpdateLastSeen(r.Context(), id); err != nil {
		h.logger.Error("failed to update last seen",
			"error", err,
			"id", id,
		)
		h.writeError(w, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
