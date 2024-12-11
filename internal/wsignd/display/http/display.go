package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/wrale/wrale-signage/api/types/v1alpha1"
	"github.com/wrale/wrale-signage/internal/wsignd/display"
	werrors "github.com/wrale/wrale-signage/internal/wsignd/errors"
)

// GetDisplay handles requests to get display status by ID or name
func (h *Handler) GetDisplay(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetReqID(r.Context())
	idStr := chi.URLParam(r, "id")

	// Try UUID first
	var d *display.Display
	var err error

	id, parseErr := uuid.Parse(idStr)
	if parseErr == nil {
		d, err = h.service.Get(r.Context(), id)
	} else {
		// Fallback to name lookup
		h.logger.Info("ID not UUID, trying name lookup",
			"requestID", reqID,
			"name", idStr,
		)
		d, err = h.service.GetByName(r.Context(), idStr)
	}

	if err != nil {
		h.logger.Error("failed to get display",
			"error", err,
			"requestID", reqID,
			"id", idStr,
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
			"requestID", reqID,
		)
		h.writeError(w, werrors.NewError("INTERNAL", "failed to encode response", "GetDisplay", err), http.StatusInternalServerError)
		return
	}
}

// ActivateDisplay handles display activation requests
func (h *Handler) ActivateDisplay(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetReqID(r.Context())
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.logger.Error("invalid display ID",
			"error", err,
			"requestID", reqID,
			"id", idStr,
		)
		h.writeError(w, werrors.NewError("INVALID_INPUT", "invalid display ID", "ActivateDisplay", err), http.StatusBadRequest)
		return
	}

	d, err := h.service.Activate(r.Context(), id)
	if err != nil {
		h.logger.Error("failed to activate display",
			"error", err,
			"requestID", reqID,
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
			"requestID", reqID,
		)
		h.writeError(w, werrors.NewError("INTERNAL", "failed to encode response", "ActivateDisplay", err), http.StatusInternalServerError)
		return
	}
}

// UpdateLastSeen updates the display's last seen timestamp
func (h *Handler) UpdateLastSeen(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetReqID(r.Context())
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.logger.Error("invalid display ID",
			"error", err,
			"requestID", reqID,
			"id", idStr,
		)
		h.writeError(w, werrors.NewError("INVALID_INPUT", "invalid display ID", "UpdateLastSeen", err), http.StatusBadRequest)
		return
	}

	if err := h.service.UpdateLastSeen(r.Context(), id); err != nil {
		h.logger.Error("failed to update last seen",
			"error", err,
			"requestID", reqID,
			"id", id,
		)
		h.writeError(w, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
