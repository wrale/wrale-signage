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

const (
	maxRequestBodySize = 1 << 20 // 1MB
)

// OAuth 2.0 Device Flow error codes
const (
	errAccessDenied         = "access_denied"
	errAuthorizationPending = "authorization_pending"
	errExpiredToken         = "expired_token"
	errSlowDown             = "slow_down"
	errInvalidRequest       = "invalid_request"
	errInvalidGrant         = "invalid_grant"
	errInvalidClient        = "invalid_client"
	errServerError          = "server_error"
)

// writeOAuthError writes an OAuth 2.0 compliant error response
func writeOAuthError(w http.ResponseWriter, code string, description string, status int, logger *slog.Logger) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	response := struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description,omitempty"`
	}{
		Error:            code,
		ErrorDescription: description,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Error("failed to encode error response",
			"error", err,
			"originalError", code,
		)
	}
}

// RequestDeviceCode handles device code generation according to OAuth 2.0 Device Authorization Grant
func (h *Handler) RequestDeviceCode(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetReqID(r.Context())
	logger := h.logger.With("requestID", reqID)

	logger.Info("handling device code request",
		"remoteAddr", r.RemoteAddr,
	)

	code, err := h.activation.GenerateCode(r.Context())
	if err != nil {
		logger.Error("failed to generate device code", "error", err)
		writeOAuthError(w, errServerError, "Failed to generate device code", http.StatusInternalServerError, logger)
		return
	}

	// Complete URI includes the code for better UX
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
		writeOAuthError(w, errServerError, "Failed to encode response", http.StatusInternalServerError, logger)
		return
	}
}

// validateActivationRequest performs validation according to OAuth 2.0 device flow
func validateActivationRequest(body []byte) (*v1alpha1.DisplayRegistrationRequest, error) {
	if len(body) == 0 {
		return nil, werrors.NewError(errInvalidRequest, "Request body is required", "ActivateDeviceCode", nil)
	}

	var req v1alpha1.DisplayRegistrationRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, werrors.NewError(errInvalidRequest, "Invalid request format", "ActivateDeviceCode", err)
	}

	// Validate all required fields
	if req.ActivationCode == "" {
		return nil, werrors.NewError(errInvalidRequest, "Activation code is required", "ActivateDeviceCode", nil)
	}

	if req.Name == "" {
		return nil, werrors.NewError(errInvalidRequest, "Display name is required", "ActivateDeviceCode", nil)
	}

	// Validate location fields
	if req.Location.SiteID == "" {
		return nil, werrors.NewError(errInvalidRequest, "Site ID is required", "ActivateDeviceCode", nil)
	}
	if req.Location.Zone == "" {
		return nil, werrors.NewError(errInvalidRequest, "Zone is required", "ActivateDeviceCode", nil)
	}
	if req.Location.Position == "" {
		return nil, werrors.NewError(errInvalidRequest, "Position is required", "ActivateDeviceCode", nil)
	}

	return &req, nil
}

// ActivateDeviceCode handles device activation following OAuth 2.0 device flow
func (h *Handler) ActivateDeviceCode(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetReqID(r.Context())
	logger := h.logger.With(
		"requestID", reqID,
		"activationCode", "",
		"name", "",
	)

	logger.Info("activating display")

	// Ensure request has a body
	if r.Body == nil {
		writeOAuthError(w, errInvalidRequest, "Request body is required", http.StatusBadRequest, logger)
		return
	}

	// Read request body with size limit
	tempBody := http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	body, err := io.ReadAll(tempBody)
	if err != nil {
		logger.Error("failed to read request body", "error", err)
		writeOAuthError(w, errInvalidRequest, "Request too large or malformed", http.StatusBadRequest, logger)
		return
	}
	defer r.Body.Close()

	// Validate request
	req, err := validateActivationRequest(body)
	if err != nil {
		logger.Error("invalid request", "error", err)
		if werr, ok := err.(*werrors.Error); ok {
			writeOAuthError(w, werr.Code, werr.Message, http.StatusBadRequest, logger)
		} else {
			writeOAuthError(w, errInvalidRequest, "Invalid request", http.StatusBadRequest, logger)
		}
		return
	}

	// Update logger context
	logger = logger.With(
		"activationCode", req.ActivationCode,
		"name", req.Name,
	)

	// Register display first
	d, err := h.service.Register(r.Context(), req.Name, display.Location{
		SiteID:   req.Location.SiteID,
		Zone:     req.Location.Zone,
		Position: req.Location.Position,
	})
	if err != nil {
		logger.Error("failed to register display", "error", err)

		switch err.(type) {
		case display.ErrExists:
			writeOAuthError(w, errInvalidRequest, "Display name already in use", http.StatusConflict, logger)
		case display.ErrInvalidName:
			writeOAuthError(w, errInvalidRequest, "Invalid display name", http.StatusBadRequest, logger)
		case display.ErrInvalidLocation:
			writeOAuthError(w, errInvalidRequest, "Invalid display location", http.StatusBadRequest, logger)
		default:
			writeOAuthError(w, errServerError, "Registration failed", http.StatusInternalServerError, logger)
		}
		return
	}

	// Activate the device code
	if err := h.activation.ActivateCode(r.Context(), req.ActivationCode, d.ID); err != nil {
		logger.Error("failed to activate device code",
			"error", err,
			"displayID", d.ID,
		)

		switch err {
		case activation.ErrCodeNotFound:
			writeOAuthError(w, errInvalidGrant, "Invalid activation code", http.StatusBadRequest, logger)
		case activation.ErrCodeExpired:
			writeOAuthError(w, errExpiredToken, "Activation code expired", http.StatusBadRequest, logger)
		case activation.ErrAlreadyActive:
			writeOAuthError(w, errInvalidGrant, "Code already activated", http.StatusConflict, logger)
		default:
			writeOAuthError(w, errServerError, "Activation failed", http.StatusInternalServerError, logger)
		}
		return
	}

	// Generate authentication tokens
	token, err := h.auth.CreateToken(r.Context(), d.ID)
	if err != nil {
		logger.Error("failed to generate auth token",
			"error", err,
			"displayID", d.ID,
		)
		writeOAuthError(w, errServerError, "Failed to generate tokens", http.StatusInternalServerError, logger)
		return
	}

	// Build successful response
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

	// Set security headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logger.Error("failed to encode response", "error", err)
		writeOAuthError(w, errServerError, "Failed to encode response", http.StatusInternalServerError, logger)
		return
	}

	logger.Info("display activated successfully",
		"displayID", d.ID,
		"name", d.Name,
	)
}
