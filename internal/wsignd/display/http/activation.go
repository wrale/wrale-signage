package http

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/wrale/wrale-signage/api/types/v1alpha1"
	"github.com/wrale/wrale-signage/internal/wsignd/display"
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
		h.writeError(w, werrors.NewError("GENERATION_FAILED", "failed to generate device code", "RequestDeviceCode", err), http.StatusInternalServerError)
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
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("failed to encode response",
			"error", err,
			"requestID", reqID,
		)
		h.writeError(w, werrors.NewError("ENCODING_ERROR", "failed to encode response", "RequestDeviceCode", err), http.StatusInternalServerError)
		return
	}
}

// validateActivationRequest performs initial validation of request body and required fields
func validateActivationRequest(body []byte) (*v1alpha1.DisplayRegistrationRequest, error) {
	if len(body) == 0 {
		return nil, werrors.NewError("INVALID_INPUT", "request body is required", "ActivateDeviceCode", nil)
	}

	var req v1alpha1.DisplayRegistrationRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, werrors.NewError("INVALID_INPUT", "invalid request body", "ActivateDeviceCode", err)
	}

	if req.ActivationCode == "" || req.Name == "" {
		return nil, werrors.NewError("INVALID_INPUT", "activation code and display name are required", "ActivateDeviceCode", nil)
	}

	return &req, nil
}

// ActivateDeviceCode handles device activation by user code
func (h *Handler) ActivateDeviceCode(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetReqID(r.Context())

	h.logger.Info("activating display",
		"requestID", reqID,
		"activationCode", "",
		"name", "",
	)

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.writeError(w, werrors.NewError("INVALID_INPUT", "failed to read request body", "ActivateDeviceCode", err), http.StatusBadRequest)
		return
	}

	// Validate request
	req, err := validateActivationRequest(body)
	if err != nil {
		h.writeError(w, err, http.StatusBadRequest)
		return
	}

	// Register display first
	d, err := h.service.Register(r.Context(), req.Name, display.Location{
		SiteID:   req.Location.SiteID,
		Zone:     req.Location.Zone,
		Position: req.Location.Position,
	})
	if err != nil {
		h.logger.Error("failed to register display",
			"error", err,
			"requestID", reqID,
			"name", req.Name,
		)

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

	// Now activate the device code with display ID
	if err := h.activation.ActivateCode(r.Context(), req.ActivationCode, d.ID); err != nil {
		h.logger.Error("failed to activate device code",
			"error", err,
			"requestID", reqID,
			"displayID", d.ID,
		)

		status := http.StatusInternalServerError
		if err == activation.ErrCodeNotFound || err == activation.ErrCodeExpired {
			status = http.StatusNotFound
		}
		h.writeError(w, werrors.NewError("NOT_FOUND", "activation code not found", "ActivateDeviceCode", err), status)
		return
	}

	// Generate authentication tokens
	token, err := h.auth.CreateToken(r.Context(), d.ID)
	if err != nil {
		h.logger.Error("failed to generate auth token",
			"error", err,
			"requestID", reqID,
			"displayID", d.ID,
		)
		h.writeError(w, werrors.NewError("TOKEN_ERROR", "activation succeeded but token generation failed", "ActivateDeviceCode", err), http.StatusInternalServerError)
		return
	}

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
		Auth: &v1alpha1.DisplayAuthTokens{
			AccessToken:      token.AccessToken,
			RefreshToken:     token.RefreshToken,
			TokenType:        "Bearer",
			ExpiresIn:        int(time.Until(token.AccessTokenExpiry).Seconds()),
			RefreshExpiresIn: int(time.Until(token.RefreshTokenExpiry).Seconds()),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("failed to encode response",
			"error", err,
			"requestID", reqID,
		)
		h.writeError(w, werrors.NewError("ENCODING_ERROR", "failed to encode response", "ActivateDeviceCode", err), http.StatusInternalServerError)
		return
	}
}
