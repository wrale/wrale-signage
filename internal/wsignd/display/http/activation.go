package http

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/wrale/wrale-signage/api/types/v1alpha1"
	"github.com/wrale/wrale-signage/internal/wsignd/display/activation"
	werrors "github.com/wrale/wrale-signage/internal/wsignd/errors"
)

// RequestDeviceCode handles device code generation
func (h *Handler) RequestDeviceCode(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetReqID(r.Context())
	h.logger.Info("handling device code request",
		"requestID", reqID,
		"remoteAddr", r.RemoteAddr,
	)

	code, err := h.activation.GenerateCode(r.Context())
	if err != nil {
		h.logger.Error("failed to generate device code",
			"error", err,
			"requestID", reqID,
		)
		h.writeError(w, err, http.StatusInternalServerError)
		return
	}

	resp := &v1alpha1.DeviceCodeResponse{
		DeviceCode:      code.DeviceCode,
		UserCode:        code.UserCode,
		ExpiresIn:       int(time.Until(code.ExpiresAt).Seconds()),
		PollInterval:    code.PollInterval,
		VerificationURI: "/api/v1alpha1/displays/activate",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// ActivateDeviceCode handles device activation by user code
func (h *Handler) ActivateDeviceCode(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetReqID(r.Context())

	var req v1alpha1.DisplayRegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode activation request",
			"error", err,
			"requestID", reqID,
		)
		h.writeError(w, werrors.NewError("INVALID_INPUT", "invalid request body", "ActivateDeviceCode", err), http.StatusBadRequest)
		return
	}

	h.logger.Info("activating display",
		"requestID", reqID,
		"activationCode", req.ActivationCode,
		"name", req.Name,
	)

	displayID, err := h.activation.ActivateCode(r.Context(), req.ActivationCode)
	if err != nil {
		status := http.StatusInternalServerError
		if err == activation.ErrCodeNotFound || err == activation.ErrCodeExpired {
			status = http.StatusNotFound
		}
		h.writeError(w, err, status)
		return
	}

	// Create display with provided details
	display, err := h.service.Register(r.Context(), req.Name, display.Location{
		SiteID:   req.Location.SiteID,
		Zone:     req.Location.Zone,
		Position: req.Location.Position,
	})
	if err != nil {
		h.logger.Error("failed to register display",
			"error", err,
			"requestID", reqID,
			"displayID", displayID,
		)
		h.writeError(w, err, http.StatusInternalServerError)
		return
	}

	resp := &v1alpha1.DisplayRegistrationResponse{
		Display: &v1alpha1.Display{
			TypeMeta: v1alpha1.TypeMeta{
				Kind:       "Display",
				APIVersion: "v1alpha1",
			},
			ObjectMeta: v1alpha1.ObjectMeta{
				ID:   display.ID,
				Name: display.Name,
			},
			Spec: v1alpha1.DisplaySpec{
				Location: v1alpha1.DisplayLocation{
					SiteID:   display.Location.SiteID,
					Zone:     display.Location.Zone,
					Position: display.Location.Position,
				},
				Properties: display.Properties,
			},
			Status: v1alpha1.DisplayStatus{
				State:    v1alpha1.DisplayState(display.State),
				LastSeen: display.LastSeen,
				Version:  display.Version,
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
