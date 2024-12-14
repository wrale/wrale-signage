package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/wrale/wrale-signage/internal/wsignd/auth"
	"github.com/wrale/wrale-signage/internal/wsignd/display"
	"github.com/wrale/wrale-signage/internal/wsignd/display/activation"
	werrors "github.com/wrale/wrale-signage/internal/wsignd/errors"
	"github.com/wrale/wrale-signage/internal/wsignd/ratelimit"
)

// OAuthErrorType represents standard OAuth 2.0 error codes
type OAuthErrorType string

// OAuth 2.0 Device Flow error codes following RFC 8628 and OAuth Core spec
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

// OAuth error responses for domain concepts aligned with RFC 6749 Section 5.2
const (
	OAuthErrDisplayNotFound = "display not found"
	OAuthErrDisplayExists   = "display already exists"
	OAuthErrCodeNotFound    = "activation code not found"
	OAuthErrCodeExpired     = "activation code expired"
	OAuthErrRateLimited     = "too many requests"
)

// OAuthError represents an OAuth 2.0 error response with HTTP status
type OAuthError struct {
	Code        OAuthErrorType
	Description string
	Status      int
	Cause       error // Original error that caused this
}

// Error implements the error interface with descriptive message
func (e *OAuthError) Error() string {
	if e.Cause != nil {
		return e.Description + ": " + e.Cause.Error()
	}
	return e.Description
}

// Unwrap returns the underlying error for error chain support
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

// Constructor helpers for common OAuth errors
func NewOAuthSlowDownError(cause error) *OAuthError {
	return NewOAuthError(OAuthErrSlowDown,
		OAuthErrRateLimited,
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

// writeError maps any error to an OAuth error response and writes it
func writeError(w http.ResponseWriter, err error, defaultStatus int, logger *slog.Logger) {
	oauthErr := mapToOAuthError(err, defaultStatus)
	writeOAuthResponse(w, oauthErr, logger)
}

// writeOAuthResponse writes a pre-constructed OAuth error response with proper headers
func writeOAuthResponse(w http.ResponseWriter, err *OAuthError, logger *slog.Logger) {
	// Set security headers first
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")

	// Set status code
	w.WriteHeader(err.Status)

	// Build OAuth-compliant response body
	response := struct {
		Code        string `json:"code,omitempty"`              // Used for domain errors
		Message     string `json:"message,omitempty"`           // Used for domain errors
		Error       string `json:"error"`                       // OAuth error code
		Description string `json:"error_description,omitempty"` // OAuth description
	}{
		Error:       string(err.Code),
		Description: err.Description,
	}

	// For domain errors, also include code/message format
	if err.Code == OAuthErrInvalidRequest {
		response.Code = err.Description
		response.Message = err.Description
		response.Description = "" // Avoid duplication
	}

	// Log and encode response
	if encodeErr := json.NewEncoder(w).Encode(response); encodeErr != nil {
		logger.Error("failed to write error response",
			"error", encodeErr,
			"originalError", err,
		)
	}
}

// mapToOAuthError converts errors to OAuth-compliant responses following display domain rules
func mapToOAuthError(err error, defaultStatus int) *OAuthError {
	// Already an OAuth error
	if oauthErr, ok := err.(*OAuthError); ok {
		return oauthErr
	}

	// Handle rate limit errors first
	if errors.Is(err, ratelimit.ErrLimitExceeded) {
		return NewOAuthSlowDownError(err)
	}

	// Handle authentication/authorization errors
	if errors.Is(err, auth.ErrTokenExpired) {
		return NewOAuthInvalidTokenError("Token expired", err)
	}
	if errors.Is(err, auth.ErrTokenInvalid) {
		return NewOAuthError(OAuthErrInvalidClient, "Invalid token", http.StatusUnauthorized, err)
	}

	// Map core error types
	switch {
	case errors.Is(err, werrors.ErrUnauthorized):
		return NewOAuthError(OAuthErrInvalidClient,
			"Authentication required",
			http.StatusUnauthorized,
			err)

	case errors.Is(err, werrors.ErrForbidden):
		return NewOAuthError(OAuthErrAccessDenied,
			"Access denied",
			http.StatusForbidden,
			err)

	case errors.Is(err, werrors.ErrNotFound):
		return NewOAuthError(OAuthErrInvalidRequest,
			"Resource not found",
			http.StatusNotFound,
			err)

	case errors.Is(err, werrors.ErrInvalidInput):
		return NewOAuthError(OAuthErrInvalidRequest,
			err.Error(),
			http.StatusBadRequest,
			err)
	}

	// Map display domain errors preserving messages
	var notFoundErr *display.ErrNotFound
	var existsErr *display.ErrExists
	var invalidStateErr *display.ErrInvalidState
	var invalidNameErr *display.ErrInvalidName
	var invalidLocationErr *display.ErrInvalidLocation
	var versionMismatchErr *display.ErrVersionMismatch

	switch {
	case errors.As(err, &notFoundErr):
		// Use 404 for all not found errors with domain-specific message
		return NewOAuthError(OAuthErrInvalidRequest,
			notFoundErr.Error(),
			http.StatusNotFound,
			err)

	case errors.As(err, &existsErr):
		return NewOAuthError(OAuthErrInvalidRequest,
			existsErr.Error(),
			http.StatusConflict,
			err)

	case errors.As(err, &invalidStateErr):
		return NewOAuthInvalidRequestError(
			invalidStateErr.Error(),
			err)

	case errors.As(err, &invalidNameErr):
		return NewOAuthInvalidRequestError(
			invalidNameErr.Error(),
			err)

	case errors.As(err, &invalidLocationErr):
		return NewOAuthInvalidRequestError(
			invalidLocationErr.Error(),
			err)

	case errors.As(err, &versionMismatchErr):
		return NewOAuthError(OAuthErrInvalidRequest,
			versionMismatchErr.Error(),
			http.StatusConflict,
			err)
	}

	// Map activation domain errors
	switch {
	case errors.Is(err, activation.ErrCodeNotFound):
		return NewOAuthError(OAuthErrInvalidGrant,
			OAuthErrCodeNotFound,
			http.StatusNotFound,
			err)

	case errors.Is(err, activation.ErrCodeExpired):
		return NewOAuthError(OAuthErrExpiredToken,
			OAuthErrCodeExpired,
			http.StatusBadRequest,
			err)

	case errors.Is(err, activation.ErrAlreadyActive):
		return NewOAuthError(OAuthErrInvalidGrant,
			"Display already activated",
			http.StatusConflict,
			err)
	}

	// Default to server error for unhandled cases
	return NewOAuthError(OAuthErrServerError,
		"An unexpected error occurred",
		defaultStatus,
		err)
}
