package http

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/wrale/wrale-signage/api/types/v1alpha1"
	"github.com/wrale/wrale-signage/internal/wsignd/display"
)

// Maximum request body size for registration/activation requests.
const maxRequestBodySize = 1 << 20 // 1MB

// RequestDeviceCode handles device code generation according to OAuth 2.0 Device Authorization Grant.
func (h *Handler) RequestDeviceCode(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetReqID(r.Context())
	logger := h.logger.With("requestID", reqID)

	logger.Info("handling device code request",
		"remoteAddr", r.RemoteAddr,
	)

	code, err := h.activation.GenerateCode(r.Context())
	if err != nil {
		logger.Error("failed to generate device code", "error", err)
		writeError(w, err, http.StatusInternalServerError, logger)
		return
	}

	verificationURI := "/api/v1alpha1/displays/activate"
	verificationURIComplete := verificationURI + "?code=" + code.UserCode

	resp := &v1alpha1.DeviceCodeResponse{
		DeviceCode:              code.DeviceCode,
		UserCode:                code.UserCode,
		ExpiresIn:               int(time.Until(code.ExpiresAt).Seconds()),
		PollInterval:            code.PollInterval,
		VerificationURI:         verificationURI,
		VerificationURIComplete: verificationURIComplete,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logger.Error("failed to encode response", "error", err)
		writeError(w, NewOAuthServerError("Failed to encode response", err), http.StatusInternalServerError, logger)
		return
	}
}

func validateActivationRequest(body []byte) (*v1alpha1.DisplayRegistrationRequest, error) {
	if len(body) == 0 {
		return nil, NewOAuthInvalidRequestError("Request body is required", nil)
	}

	var req v1alpha1.DisplayRegistrationRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, NewOAuthInvalidRequestError("Invalid request format", err)
	}

	if req.ActivationCode == "" {
		return nil, NewOAuthInvalidRequestError("Activation code is required", nil)
	}
	if req.Name == "" {
		return nil, NewOAuthInvalidRequestError("Display name is required", nil)
	}
	if req.Location.SiteID == "" {
		return nil, NewOAuthInvalidRequestError("Site ID is required", nil)
	}
	if req.Location.Zone == "" {
		return nil, NewOAuthInvalidRequestError("Zone is required", nil)
	}
	if req.Location.Position == "" {
		return nil, NewOAuthInvalidRequestError("Position is required", nil)
	}

	return &req, nil
}

// ActivateDeviceCode handles display activation following OAuth 2.0 device flow.
func (h *Handler) ActivateDeviceCode(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetReqID(r.Context())
	logger := h.logger.With(
		"requestID", reqID,
		"activationCode", "",
		"name", "",
	)

	logger.Info("activating display")

	if r.Body == nil {
		writeError(w, NewOAuthInvalidRequestError("Request body is required", nil), http.StatusBadRequest, logger)
		return
	}

	tempBody := http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	body, err := io.ReadAll(tempBody)
	if err != nil {
		logger.Error("failed to read request body", "error", err)
		writeError(w, NewOAuthInvalidRequestError("Request too large or malformed", err), http.StatusBadRequest, logger)
		return
	}
	defer r.Body.Close()

	req, err := validateActivationRequest(body)
	if err != nil {
		logger.Error("invalid request", "error", err)
		writeError(w, err, http.StatusBadRequest, logger)
		return
	}

	logger = logger.With(
		"activationCode", req.ActivationCode,
		"name", req.Name,
	)

	d, err := h.service.Register(r.Context(), req.Name, display.Location{
		SiteID:   req.Location.SiteID,
		Zone:     req.Location.Zone,
		Position: req.Location.Position,
	})
	if err != nil {
		logger.Error("failed to register display", "error", err)
		writeError(w, err, http.StatusInternalServerError, logger)
		return
	}

	if err := h.activation.ActivateCode(r.Context(), req.ActivationCode, d.ID); err != nil {
		logger.Error("failed to activate device code",
			"error", err,
			"displayID", d.ID,
		)
		writeError(w, err, http.StatusInternalServerError, logger)
		return
	}

	token, err := h.auth.CreateToken(r.Context(), d.ID)
	if err != nil {
		logger.Error("failed to generate auth token",
			"error", err,
			"displayID", d.ID,
		)
		writeError(w, NewOAuthServerError("Failed to generate tokens", err), http.StatusInternalServerError, logger)
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
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logger.Error("failed to encode response", "error", err)
		writeError(w, NewOAuthServerError("Failed to encode response", err), http.StatusInternalServerError, logger)
		return
	}

	logger.Info("display activated successfully",
		"displayID", d.ID,
		"name", d.Name,
	)
}
