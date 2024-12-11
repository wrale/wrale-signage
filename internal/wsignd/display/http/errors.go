package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/wrale/wrale-signage/internal/wsignd/display"
	werrors "github.com/wrale/wrale-signage/internal/wsignd/errors"
)

// mapDomainError converts domain-specific errors to werrors.Error
func (h *Handler) mapDomainError(err error, op string) error {
	switch e := err.(type) {
	case display.ErrNotFound:
		return werrors.NewError("NOT_FOUND", e.Error(), op, nil)
	case display.ErrExists:
		return werrors.NewError("CONFLICT", e.Error(), op, nil)
	case display.ErrInvalidState:
		return werrors.NewError("INVALID_INPUT", e.Error(), op, nil)
	case display.ErrInvalidName:
		return werrors.NewError("INVALID_INPUT", e.Error(), op, nil)
	case display.ErrInvalidLocation:
		return werrors.NewError("INVALID_INPUT", e.Error(), op, nil)
	case display.ErrVersionMismatch:
		return werrors.NewError("CONFLICT", e.Error(), op, nil)
	default:
		return werrors.NewError("INTERNAL", "internal server error", op, err)
	}
}

// writeError writes a JSON error response, falling back to plain text if JSON encoding fails
func (h *Handler) writeError(w http.ResponseWriter, err error, defaultStatus int) {
	// First map domain errors to werrors
	if _, ok := err.(*werrors.Error); !ok {
		err = h.mapDomainError(err, "http")
	}

	var werr *werrors.Error
	if errors.As(err, &werr) {
		status := defaultStatus
		switch werr.Code {
		case "NOT_FOUND":
			status = http.StatusNotFound
		case "CONFLICT":
			status = http.StatusConflict
		case "INVALID_INPUT":
			status = http.StatusBadRequest
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)

		response := map[string]string{
			"code":    werr.Code,
			"message": werr.Message,
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			// Log JSON encoding error and fall back to plain text
			h.logger.Error("failed to encode error response",
				"error", err,
				"original_error", werr,
			)
			http.Error(w, fmt.Sprintf("%s: %s", werr.Code, werr.Message), status)
		}
		return
	}

	// Default error response
	http.Error(w, "internal server error", defaultStatus)
}
