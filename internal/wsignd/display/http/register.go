package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/wrale/wrale-signage/api/types/v1alpha1"
	"github.com/wrale/wrale-signage/internal/wsignd/display"
	werrors "github.com/wrale/wrale-signage/internal/wsignd/errors"
)

// RegisterDisplay handles display registration requests
func (h *Handler) RegisterDisplay(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetReqID(r.Context())
	h.logger.Info("handling display registration request",
		"requestID", reqID,
		"remoteAddr", r.RemoteAddr,
	)

	// Log request body for debugging
	var body struct {
		Name     string   `json:"name"`
		Location struct{} `json:"location"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.logger.Error("failed to decode request body",
			"error", err,
			"requestID", reqID,
		)
		h.writeError(w, werrors.NewError("INVALID_INPUT", "invalid request body", "RegisterDisplay", err), http.StatusBadRequest)
		return
	}
	h.logger.Debug("received registration request",
		"requestID", reqID,
		"name", body.Name,
		"body", body,
	)

	// Re-decode for actual processing
	r.Body.Close()
	var req v1alpha1.DisplayRegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode registration request",
			"error", err,
			"requestID", reqID,
		)
		h.writeError(w, werrors.NewError("INVALID_INPUT", "invalid request body", "RegisterDisplay", err), http.StatusBadRequest)
		return
	}

	// Validate request fields
	if req.Name == "" {
		h.logger.Error("display name is required",
			"requestID", reqID,
			"name", req.Name,
		)
		h.writeError(w, werrors.NewError("INVALID_INPUT", "display name is required", "RegisterDisplay", nil), http.StatusBadRequest)
		return
	}
	if req.Location.SiteID == "" {
		h.logger.Error("site ID is required",
			"requestID", reqID,
			"name", req.Name,
		)
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
			"requestID", reqID,
			"name", req.Name,
			"location", location,
		)
		h.writeError(w, err, http.StatusInternalServerError)
		return
	}

	h.logger.Info("display registered successfully",
		"requestID", reqID,
		"displayID", d.ID,
		"name", d.Name,
		"location", d.Location,
	)

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
			"requestID", reqID,
		)
		h.writeError(w, werrors.NewError("INTERNAL", "failed to encode response", "RegisterDisplay", err), http.StatusInternalServerError)
		return
	}
}
