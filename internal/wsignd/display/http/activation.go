package http

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/wrale/wrale-signage/api/types/v1alpha1"
	"github.com/wrale/wrale-signage/internal/wsignd/display"
	"github.com/wrale/wrale-signage/internal/wsignd/display/activation"
)

// Maximum request body size for registration/activation requests.
// We keep this relatively small as these payloads should never be large.
const maxRequestBodySize = 1 << 20 // 1MB

// RequestDeviceCode handles device code generation according to OAuth 2.0 Device Authorization Grant.
// This implements the first step of the device flow where an unauthenticated device requests
// an activation code that a user can enter on another device.
func (h *Handler) RequestDeviceCode(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetReqID(r.Context())
	logger := h.logger.With("requestID", reqID)

	logger.Info("handling device code request",
		"remoteAddr", r.RemoteAddr,
	)

	// Generate a new device code pair
	code, err := h.activation.GenerateCode(r.Context())
	if err != nil {
		logger.Error("failed to generate device code", "error", err)
		writeError(w, err, http.StatusInternalServerError, logger)
		return
	}

	// Build activation URLs for user convenience
	verificationURI := "/api/v1alpha1/displays/activate"
	verificationURIComplete := verificationURI + "?code=" + code.UserCode

	// Prepare the response following RFC 8628
	resp := &v1alpha1.DeviceCodeResponse{
		DeviceCode:              code.DeviceCode,
		UserCode:                code.UserCode,
		ExpiresIn:               int(time.Until(code.ExpiresAt).Seconds()),
		PollInterval:            code.PollInterval,
		VerificationURI:         verificationURI,
		VerificationURIComplete: verificationURIComplete,
	}

	// Set security headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logger.Error("failed to encode response", "error", err)
		writeError(w, NewOAuthServerError("Failed to encode response", err), http.StatusInternalServerError, logger)
		return
	}
}

// validateActivationRequest performs comprehensive validation of the activation request.
// This ensures all required fields are present and correctly formatted before we attempt
// any registration or activation steps.
func validateActivationRequest(body []byte) (*v1alpha1.DisplayRegistrationRequest, error) {
	if len(body) == 0 {
		return nil, NewOAuthInvalidRequestError("Request body is required", nil)
	}

	var req v1alpha1.DisplayRegistrationRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, NewOAuthInvalidRequestError("Invalid request format", err)
	}

	// Validate all required fields according to OAuth specification
	if req.ActivationCode == "" {
		return nil, NewOAuthInvalidRequestError("Activation code is required", nil)
	}

	// Validate display-specific fields
	if req.Name == "" {
		return nil, NewOAuthInvalidRequestError("Display name is required", nil)
	}

	// Validate location fields which are required for proper display registration
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
// This implements the verification endpoint where an authenticated user confirms
// the device activation by entering the user code.
func (h *Handler) ActivateDeviceCode(w http.ResponseWriter, r *http.Request) {
	// Extract request ID and create a request-scoped logger
	reqID := middleware.GetReqID(r.Context())
	logger := h.logger.With(
		"requestID", reqID,
		"activationCode", "",
		"name", "",
	)

	logger.Info("activating display")

	// Verify we have a request body
	if r.Body == nil {
		writeError(w, NewOAuthInvalidRequestError("Request body is required", nil), http.StatusBadRequest, logger)
		return
	}

	// Enforce request size limits for security
	tempBody := http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	body, err := io.ReadAll(tempBody)
	if err != nil {
		logger.Error("failed to read request body", "error", err)
		writeError(w, NewOAuthInvalidRequestError("Request too large or malformed", err), http.StatusBadRequest, logger)
		return
	}
	defer r.Body.Close()

	// Validate the activation request content
	req, err := validateActivationRequest(body)
	if err != nil {
		logger.Error("invalid request", "error", err)
		writeError(w, err, http.StatusBadRequest, logger)
		return
	}

	// Update logger with request context
	logger = logger.With(
		"activationCode", req.ActivationCode,
		"name", req.Name,
	)

	// First register the display - this creates the display record
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

	// After registration, activate the device code to associate it with the display
	if err := h.activation.ActivateCode(r.Context(), req.ActivationCode, d.ID); err != nil {
		logger.Error("failed to activate device code",
			"error", err,
			"displayID", d.ID,
		)
		writeError(w, err, http.StatusInternalServerError, logger)
		return
	}

	// Generate authentication tokens for the activated display
	token, err := h.auth.CreateToken(r.Context(), d.ID)
	if err != nil {
		logger.Error("failed to generate auth token",
			"error", err,
			"displayID", d.ID,
		)
		writeError(w, NewOAuthServerError("Failed to generate tokens", err), http.StatusInternalServerError, logger)
		return
	}

	// Build successful activation response
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

	// Set security headers for auth response
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
