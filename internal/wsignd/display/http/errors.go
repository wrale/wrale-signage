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

// Domain error codes for consistent error responses
const (
	ErrCodeNotFound        = "NOT_FOUND"
	ErrCodeAlreadyExists   = "ALREADY_EXISTS"
	ErrCodeInvalidInput    = "INVALID_INPUT"
	ErrCodeRateLimited     = "RATE_LIMITED"
	ErrCodeInvalidState    = "INVALID_STATE"
	ErrCodeVersionMismatch = "VERSION_MISMATCH"
	ErrCodeInvalidLocation = "INVALID_LOCATION"
	ErrCodeInvalidName     = "INVALID_NAME"
)

// OAuthError represents an OAuth 2.0 error response with HTTP status
type OAuthError struct {
	Code        OAuthErrorType `json:"error"`
	Description string         `json:"error_description,omitempty"`
	Status      int            `json:"-"`
	Cause       error          `json:"-"`
	DomainCode  string         `json:"code,omitempty"`
}

func (e *OAuthError) Error() string {
	if e.Cause != nil {
		return e.Description + ": " + e.Cause.Error()
	}
	return e.Description
}

func (e *OAuthError) Unwrap() error {
	return e.Cause
}

// NewOAuthError creates a new OAuth error without domain code
func NewOAuthError(code OAuthErrorType, description string, status int, cause error) *OAuthError {
	return &OAuthError{
		Code:        code,
		Description: description,
		Status:      status,
		Cause:       cause,
	}
}

// NewDomainError creates an OAuth error with domain context
func NewDomainError(code OAuthErrorType, domainCode, description string, status int, cause error) *OAuthError {
	return &OAuthError{
		Code:        code,
		DomainCode:  domainCode,
		Description: description,
		Status:      status,
		Cause:       cause,
	}
}

// Common OAuth error constructors
func NewOAuthSlowDownError(cause error) *OAuthError {
	return NewDomainError(OAuthErrSlowDown,
		ErrCodeRateLimited,
		"Rate limit exceeded",
		http.StatusTooManyRequests,
		cause)
}

func NewOAuthInvalidRequestError(description string, cause error) *OAuthError {
	return NewOAuthError(OAuthErrInvalidRequest,
		description,
		http.StatusBadRequest,
		cause)
}

func NewOAuthUnauthorizedError(description string, cause error) *OAuthError {
	return NewOAuthError(OAuthErrInvalidClient,
		description,
		http.StatusUnauthorized,
		cause)
}

func NewOAuthServerError(cause error) *OAuthError {
	return NewOAuthError(OAuthErrServerError,
		"Internal server error",
		http.StatusInternalServerError,
		cause)
}

// writeError maps domain errors to OAuth responses
func writeError(w http.ResponseWriter, err error, defaultStatus int, logger *slog.Logger) {
	oauthErr := mapToOAuthError(err, defaultStatus)
	writeOAuthResponse(w, oauthErr, logger)
}

// writeOAuthResponse writes a structured error response
func writeOAuthResponse(w http.ResponseWriter, err *OAuthError, logger *slog.Logger) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")

	w.WriteHeader(err.Status)

	// Use domain error info when available
	response := struct {
		Error       string `json:"error"`                       // OAuth error type
		Description string `json:"error_description,omitempty"` // OAuth description
		Code        string `json:"code,omitempty"`              // Domain error code
		Message     string `json:"message,omitempty"`           // Human message
	}{
		Error: string(err.Code),
	}

	if err.DomainCode != "" {
		response.Code = err.DomainCode
		response.Message = err.Description // Use description as message
	} else {
		response.Description = err.Description
	}

	if encodeErr := json.NewEncoder(w).Encode(response); encodeErr != nil {
		logger.Error("failed to encode error response",
			"error", encodeErr,
			"original_error", err)
	}
}

// mapToOAuthError provides consistent error mapping
func mapToOAuthError(err error, defaultStatus int) *OAuthError {
	// Return existing OAuth errors
	if oauthErr, ok := err.(*OAuthError); ok {
		return oauthErr
	}

	// Handle rate limiting first
	if errors.Is(err, ratelimit.ErrLimitExceeded) {
		return NewOAuthSlowDownError(err)
	}

	// Map authentication errors
	if errors.Is(err, auth.ErrTokenExpired) || errors.Is(err, auth.ErrTokenInvalid) {
		return NewOAuthUnauthorizedError("Invalid or expired token", err)
	}

	// Map core error types
	switch {
	case errors.Is(err, werrors.ErrUnauthorized):
		return NewOAuthUnauthorizedError("Authentication required", err)

	case errors.Is(err, werrors.ErrForbidden):
		return NewOAuthError(OAuthErrAccessDenied,
			"Access denied",
			http.StatusForbidden,
			err)

	case errors.Is(err, werrors.ErrNotFound):
		return NewDomainError(OAuthErrInvalidRequest,
			ErrCodeNotFound,
			err.Error(),
			http.StatusNotFound,
			err)

	case errors.Is(err, werrors.ErrInvalidInput):
		return NewDomainError(OAuthErrInvalidRequest,
			ErrCodeInvalidInput,
			err.Error(),
			http.StatusBadRequest,
			err)
	}

	// Map display domain errors
	var notFoundErr *display.ErrNotFound
	var existsErr *display.ErrExists
	var invalidStateErr *display.ErrInvalidState
	var invalidNameErr *display.ErrInvalidName
	var invalidLocationErr *display.ErrInvalidLocation
	var versionMismatchErr *display.ErrVersionMismatch

	switch {
	case errors.As(err, &notFoundErr):
		return NewDomainError(OAuthErrInvalidRequest,
			ErrCodeNotFound,
			err.Error(),
			http.StatusNotFound,
			err)

	case errors.As(err, &existsErr):
		return NewDomainError(OAuthErrInvalidRequest,
			ErrCodeAlreadyExists,
			err.Error(),
			http.StatusConflict,
			err)

	case errors.As(err, &invalidStateErr):
		return NewDomainError(OAuthErrInvalidRequest,
			ErrCodeInvalidState,
			err.Error(),
			http.StatusBadRequest,
			err)

	case errors.As(err, &invalidNameErr):
		return NewDomainError(OAuthErrInvalidRequest,
			ErrCodeInvalidName,
			err.Error(),
			http.StatusBadRequest,
			err)

	case errors.As(err, &invalidLocationErr):
		return NewDomainError(OAuthErrInvalidRequest,
			ErrCodeInvalidLocation,
			err.Error(),
			http.StatusBadRequest,
			err)

	case errors.As(err, &versionMismatchErr):
		return NewDomainError(OAuthErrInvalidRequest,
			ErrCodeVersionMismatch,
			err.Error(),
			http.StatusConflict,
			err)
	}

	// Map activation errors
	switch {
	case errors.Is(err, activation.ErrCodeNotFound):
		return NewDomainError(OAuthErrInvalidGrant,
			ErrCodeNotFound,
			"Activation code not found",
			http.StatusNotFound,
			err)

	case errors.Is(err, activation.ErrCodeExpired):
		return NewDomainError(OAuthErrExpiredToken,
			ErrCodeNotFound,
			"Activation code expired",
			http.StatusBadRequest,
			err)

	case errors.Is(err, activation.ErrAlreadyActive):
		return NewDomainError(OAuthErrInvalidGrant,
			ErrCodeAlreadyExists,
			"Display already activated",
			http.StatusConflict,
			err)
	}

	// Default server error
	return NewOAuthServerError(err)
}
