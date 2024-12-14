package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
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

// OAuthError represents an OAuth 2.0 error response
type OAuthError struct {
	Code        OAuthErrorType
	Description string
	Status      int
}

// Error implements the error interface
func (e *OAuthError) Error() string {
	return e.Description
}

// OAuth error constructors for common cases
func NewOAuthSlowDownError() *OAuthError {
	return &OAuthError{
		Code:        OAuthErrSlowDown,
		Description: "Too many requests, please reduce request rate",
		Status:      http.StatusTooManyRequests,
	}
}

func NewOAuthInvalidRequestError(description string) *OAuthError {
	return &OAuthError{
		Code:        OAuthErrInvalidRequest,
		Description: description,
		Status:      http.StatusBadRequest,
	}
}

func NewOAuthInvalidTokenError(description string) *OAuthError {
	return &OAuthError{
		Code:        OAuthErrExpiredToken,
		Description: description,
		Status:      http.StatusUnauthorized,
	}
}

func NewOAuthServerError(description string) *OAuthError {
	return &OAuthError{
		Code:        OAuthErrServerError,
		Description: description,
		Status:      http.StatusInternalServerError,
	}
}

// writeOAuthError writes a standardized OAuth 2.0 error response
func writeOAuthError(w http.ResponseWriter, err *OAuthError, logger *slog.Logger) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.WriteHeader(err.Status)

	response := struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description,omitempty"`
	}{
		Error:            string(err.Code),
		ErrorDescription: err.Description,
	}

	if encodeErr := json.NewEncoder(w).Encode(response); encodeErr != nil {
		logger.Error("failed to write error response",
			"error", encodeErr,
			"originalError", err,
		)
	}
}

// mapDomainError converts domain errors to OAuth errors
func mapDomainError(err error) *OAuthError {
	switch err.(type) {
	case *OAuthError:
		return err.(*OAuthError)
	default:
		return NewOAuthServerError("An unexpected error occurred")
	}
}
