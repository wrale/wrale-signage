package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/wrale/wrale-signage/internal/wsignd/display"
	"github.com/wrale/wrale-signage/internal/wsignd/errors"
)

// Get retrieves a display by ID
func (s *Service) Get(ctx context.Context, id uuid.UUID) (*display.Display, error) {
	const op = "DisplayService.Get"

	d, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, errors.NewError("NOT_FOUND", fmt.Sprintf("Display not found: %s", id), op, err)
		}
		return nil, errors.NewError("LOOKUP_FAILED", "Failed to retrieve display", op, err)
	}

	return d, nil
}

// GetByName retrieves a display by name
func (s *Service) GetByName(ctx context.Context, name string) (*display.Display, error) {
	const op = "DisplayService.GetByName"

	d, err := s.repo.FindByName(ctx, name)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, errors.NewError("NOT_FOUND", fmt.Sprintf("Display not found: %s", name), op, err)
		}
		return nil, errors.NewError("LOOKUP_FAILED", "Failed to retrieve display", op, err)
	}

	return d, nil
}

// List retrieves displays matching the filter
func (s *Service) List(ctx context.Context, filter display.DisplayFilter) ([]*display.Display, error) {
	const op = "DisplayService.List"

	displays, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, errors.NewError("LIST_FAILED", "Failed to list displays", op, err)
	}

	return displays, nil
}
