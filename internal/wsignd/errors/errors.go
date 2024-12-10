// Package errors provides standardized error handling for the Wrale Signage server
package errors

import (
	"errors"
	"fmt"
)

// Common sentinel errors that can be used across the application
var (
	// ErrNotFound indicates a requested resource doesn't exist
	ErrNotFound = errors.New("resource not found")
	
	// ErrConflict indicates a resource already exists
	ErrConflict = errors.New("resource already exists")
	
	// ErrInvalidInput indicates invalid input parameters
	ErrInvalidInput = errors.New("invalid input")
	
	// ErrUnauthorized indicates missing or invalid authentication
	ErrUnauthorized = errors.New("unauthorized")
	
	// ErrForbidden indicates the authenticated user lacks permission
	ErrForbidden = errors.New("forbidden")
	
	// ErrVersionMismatch indicates optimistic concurrency failure
	ErrVersionMismatch = errors.New("version mismatch")
)

// Error represents a domain error with additional context
type Error struct {
	// Code is a machine-readable error code
	Code string
	// Message is a human-readable error description
	Message string
	// Op describes the operation that failed
	Op string
	// Err is the underlying error
	Err error
}

// Error implements the error interface with a formatted message
func (e *Error) Error() string {
	if e.Op != "" {
		return fmt.Sprintf("%s: %s", e.Op, e.Message)
	}
	return e.Message
}

// Unwrap returns the underlying error for error chain handling
func (e *Error) Unwrap() error {
	return e.Err
}

// NewError creates a new Error with the given details
func NewError(code string, message string, op string, err error) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Op:      op,
		Err:     err,
	}
}

// IsNotFound returns true if err represents a not found error
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsConflict returns true if err represents a conflict error
func IsConflict(err error) bool {
	return errors.Is(err, ErrConflict)
}

// IsInvalidInput returns true if err represents an invalid input error
func IsInvalidInput(err error) bool {
	return errors.Is(err, ErrInvalidInput)
}

// IsUnauthorized returns true if err represents an unauthorized error
func IsUnauthorized(err error) bool {
	return errors.Is(err, ErrUnauthorized)
}

// IsForbidden returns true if err represents a forbidden error
func IsForbidden(err error) bool {
	return errors.Is(err, ErrForbidden)
}

// IsVersionMismatch returns true if err represents a version mismatch error
func IsVersionMismatch(err error) bool {
	return errors.Is(err, ErrVersionMismatch)
}
