package content

import (
	"errors"
	"fmt"
)

// Standard error codes for content operations
const (
	ErrCodeInvalidData      = "INVALID_DATA"
	ErrCodeInvalidReference = "INVALID_REFERENCE"
	ErrCodeAlreadyExists    = "ALREADY_EXISTS"
)

// Common errors returned by content operations
var (
	ErrNotFound          = errors.New("content not found")
	ErrContentStale      = errors.New("content not seen recently")
	ErrContentUnreliable = errors.New("content has high error rate")
	ErrDisplayNotFound   = errors.New("display not found")
)

// Error represents a content operation error with additional context
type Error struct {
	Code    string // Machine-readable error code
	Message string // Human-readable error message
	Op      string // Operation where error occurred
	Err     error  // Original error if wrapping
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v [%s]", e.Op, e.Message, e.Code)
	}
	return fmt.Sprintf("%s: %s [%s]", e.Op, e.Message, e.Code)
}

func (e *Error) Unwrap() error {
	return e.Err
}

// IsNotFound returns true if err is or wraps ErrNotFound
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsDisplayNotFound returns true if err is or wraps ErrDisplayNotFound
func IsDisplayNotFound(err error) bool {
	return errors.Is(err, ErrDisplayNotFound)
}
