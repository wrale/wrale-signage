package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/wrale/wrale-signage/internal/wsignd/display"
	"github.com/wrale/wrale-signage/internal/wsignd/display/activation"
)

// OAuthErrorType represents standard OAuth 2.0 error codes
type OAuthErrorType string

// OAuth 2.0 Device Flow error codes
const (
	OAuthErrAccessDenied         OAuthErrorType = "access_denied"
	OAuthErrAuthorizationPending OAuthErrorType = "authorization_pending"
	OAuthErrExpiredToken         OAuthErrorType = "expired_token"
	OAuthErrSlowDown             OAuthErrorType = "slow_down"
	OAuthErrInvalidRequest       OAuthErrorType = "invalid_request"
	OAuthErrInvalidGrant         OAuthErrorType = "invalid_grant"
	OAuthErrInvalidClient        OAuthErrorType = "invalid_client"
	OAuthErrServerError          OAuthErrorType = "server_error"
)

// OAuthError represents an OAuth 2.0 error response with HTTP status
type OAuthError struct {
	Code        OAuthErrorType
	Description string
	Status      int
	Cause       error // Original error that caused this
}

// Error implements the error interface
func (e *OAuthError) Error() string {
	if e.Cause != nil {
		return e.Description + ": " + e.Cause.Error()
	}
	return e.Description
}

// Unwrap returns the underlying error
func (e *OAuthError) Unwrap() error {
	return e.Cause
}

// NewOAuthError creates a new OAuth error with cause
func NewOAuthError(code OAuthErrorType, description string, status int, cause error) *OAuthError {
	return &OAuthError{
		Code:        code,
		Description: description,
		Status:      status,
		Cause:       cause,
	}
}

// OAuth error constructors for common cases
func NewOAuthSlowDownError(cause error) *OAuthError {
	return NewOAuthError(OAuthErrSlowDown,
		"Too many requests, please reduce request rate",
		http.StatusTooManyRequests,
		cause)
}

func NewOAuthInvalidRequestError(description string, cause error) *OAuthError {
	return NewOAuthError(OAuthErrInvalidRequest,
		description,
		http.StatusBadRequest,
		cause)
}

func NewOAuthInvalidTokenError(description string, cause error) *OAuthError {
	return NewOAuthError(OAuthErrExpiredToken,
		description,
		http.StatusUnauthorized,
		cause)
}

func NewOAuthServerError(description string, cause error) *OAuthError {
	return NewOAuthError(OAuthErrServerError,
		description,
		http.StatusInternalServerError,
		cause)
}

// writeError writes any error as an appropriate OAuth error response
func writeError(w http.ResponseWriter, err error, defaultStatus int, logger *slog.Logger) {
	oauthErr := mapToOAuthError(err, defaultStatus)
	writeOAuthResponse(w, oauthErr, logger)
}

// writeOAuthResponse writes a pre-constructed OAuth error response
func writeOAuthResponse(w http.ResponseWriter, err *OAuthError, logger *slog.Logger) {
	// Set security headers first
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")

	// Set status code
	w.WriteHeader(err.Status)

	// Build response body
	response := struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description,omitempty"`
	}{
		Error:            string(err.Code),
		ErrorDescription: err.Description,
	}

	// Log and encode response
	if encodeErr := json.NewEncoder(w).Encode(response); encodeErr != nil {
		logger.Error("failed to write error response",
			"error", encodeErr,
			"originalError", err,
		)
	}
}

// mapToOAuthError converts domain errors to OAuth errors
func mapToOAuthError(err error, defaultStatus int) *OAuthError {
	// Already an OAuth error
	if oauthErr, ok := err.(*OAuthError); ok {
		return oauthErr
	}

	// Map domain errors to OAuth errors using errors.Is
	switch {
	case errors.Is(err, display.ErrNotFound):
		return NewOAuthError(OAuthErrInvalidRequest,
			"Display not found",
			http.StatusNotFound,
			err)

	case errors.Is(err, display.ErrExists):
		return NewOAuthError(OAuthErrInvalidRequest,
			"Display already exists",
			http.StatusConflict,
			err)

	case errors.Is(err, display.ErrInvalidState):
		return NewOAuthError(OAuthErrInvalidRequest,
			"Invalid display state",
			http.StatusBadRequest,
			err)

	case errors.Is(err, display.ErrInvalidName):
		return NewOAuthError(OAuthErrInvalidRequest,
			"Invalid display name",
			http.StatusBadRequest,
			err)

	case errors.Is(err, display.ErrInvalidLocation):
		return NewOAuthError(OAuthErrInvalidRequest,
			"Invalid display location",
			http.StatusBadRequest,
			err)

	case errors.Is(err, display.ErrVersionMismatch):
		return NewOAuthError(OAuthErrInvalidRequest,
			"Display was modified by another request",
			http.StatusConflict,
			err)

	case errors.Is(err, activation.ErrCodeNotFound):
		return NewOAuthError(OAuthErrInvalidGrant,
			"Invalid activation code",
			http.StatusBadRequest,
			err)

	case errors.Is(err, activation.ErrCodeExpired):
		return NewOAuthError(OAuthErrExpiredToken,
			"Activation code expired",
			http.StatusBadRequest,
			err)

	case errors.Is(err, activation.ErrAlreadyActive):
		return NewOAuthError(OAuthErrInvalidGrant,
			"Code already activated",
			http.StatusConflict,
			err)

	default:
		return NewOAuthError(OAuthErrServerError,
			"An unexpected error occurred",
			defaultStatus,
			err)
	}
}
