package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/wrale/wrale-signage/internal/wsignd/errors"
)

// SetProperty sets a display property
func (s *Service) SetProperty(ctx context.Context, id uuid.UUID, key, value string) error {
	const op = "DisplayService.SetProperty"

	if key == "" {
		return errors.NewError("INVALID_INPUT", "Property key cannot be empty", op, errors.ErrInvalidInput)
	}

	// Get current display state
	d, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.IsNotFound(err) {
			return errors.NewError("NOT_FOUND", fmt.Sprintf("Display not found: %s", id), op, err)
		}
		return errors.NewError("LOOKUP_FAILED", "Failed to retrieve display", op, err)
	}

	// Update property through domain model
	d.SetProperty(key, value)

	// Persist changes
	if err := s.repo.Save(ctx, d); err != nil {
		if errors.IsVersionMismatch(err) {
			return errors.NewError("VERSION_CONFLICT", "Display was modified", op, err)
		}
		return errors.NewError("SAVE_FAILED", "Failed to save property update", op, err)
	}

	return nil
}
