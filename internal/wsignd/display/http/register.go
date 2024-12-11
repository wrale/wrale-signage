// Package http implements the HTTP handlers for the display service
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

	var req v1alpha1.DisplayRegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode registration request",
			"error", err,
			"requestID", reqID,
		)
		h.writeError(w, werrors.NewError("INVALID_INPUT", "invalid request body", "RegisterDisplay", err), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

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

		// Map domain errors to HTTP status codes
		var status int
		switch err.(type) {
		case display.ErrExists:
			status = http.StatusConflict
		case display.ErrInvalidName, display.ErrInvalidLocation:
			status = http.StatusBadRequest
		default:
			status = http.StatusInternalServerError
		}

		h.writeError(w, err, status)
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
