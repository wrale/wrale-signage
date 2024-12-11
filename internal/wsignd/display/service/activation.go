package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/wrale/wrale-signage/internal/wsignd/display"
	"github.com/wrale/wrale-signage/internal/wsignd/errors"
)

// Activate transitions a display to the active state
func (s *Service) Activate(ctx context.Context, id uuid.UUID) (*display.Display, error) {
	const op = "DisplayService.Activate"

	// Get current display state
	d, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, errors.NewError("NOT_FOUND", fmt.Sprintf("Display not found: %s", id), op, err)
		}
		return nil, errors.NewError("LOOKUP_FAILED", "Failed to retrieve display", op, err)
	}

	// Activate through domain model
	if err := d.Activate(); err != nil {
		return nil, err // Domain model errors are already properly typed
	}

	// Persist changes
	if err := s.repo.Save(ctx, d); err != nil {
		if errors.IsVersionMismatch(err) {
			return nil, errors.NewError("VERSION_CONFLICT", "Display was modified", op, err)
		}
		return nil, errors.NewError("SAVE_FAILED", "Failed to save state change", op, err)
	}

	// Publish activated event
	event := display.Event{
		Type:      display.EventActivated,
		DisplayID: d.ID,
		Timestamp: time.Now(),
		Data: map[string]string{
			"state":   string(d.State),
			"version": fmt.Sprint(d.Version),
		},
	}

	if err := s.publisher.Publish(ctx, event); err != nil {
		// Log but don't fail the operation if event publishing fails
		// TODO: Add proper logging
		fmt.Printf("Failed to publish activated event: %v\n", err)
	}

	return d, nil
}

// Disable transitions a display to the disabled state
func (s *Service) Disable(ctx context.Context, id uuid.UUID) error {
	const op = "DisplayService.Disable"

	// Get current display state
	d, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.IsNotFound(err) {
			return errors.NewError("NOT_FOUND", fmt.Sprintf("Display not found: %s", id), op, err)
		}
		return errors.NewError("LOOKUP_FAILED", "Failed to retrieve display", op, err)
	}

	// Disable through domain model
	d.Disable()

	// Persist changes
	if err := s.repo.Save(ctx, d); err != nil {
		if errors.IsVersionMismatch(err) {
			return errors.NewError("VERSION_CONFLICT", "Display was modified", op, err)
		}
		return errors.NewError("SAVE_FAILED", "Failed to save state change", op, err)
	}

	// Publish disabled event
	event := display.Event{
		Type:      display.EventDisabled,
		DisplayID: d.ID,
		Timestamp: time.Now(),
		Data: map[string]string{
			"state":   string(d.State),
			"version": fmt.Sprint(d.Version),
		},
	}

	if err := s.publisher.Publish(ctx, event); err != nil {
		// Log but don't fail the operation if event publishing fails
		// TODO: Add proper logging
		fmt.Printf("Failed to publish disabled event: %v\n", err)
	}

	return nil
}

// UpdateLastSeen updates the display's last seen timestamp
func (s *Service) UpdateLastSeen(ctx context.Context, id uuid.UUID) error {
	const op = "DisplayService.UpdateLastSeen"

	// Get current display state
	d, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.IsNotFound(err) {
			return errors.NewError("NOT_FOUND", fmt.Sprintf("Display not found: %s", id), op, err)
		}
		return errors.NewError("LOOKUP_FAILED", "Failed to retrieve display", op, err)
	}

	// Update timestamp through domain model
	d.UpdateLastSeen()

	// Persist changes with retry on version conflicts
	if err := s.repo.Save(ctx, d); err != nil {
		if errors.IsVersionMismatch(err) {
			return errors.NewError("VERSION_CONFLICT", "Display was modified", op, err)
		}
		return errors.NewError("SAVE_FAILED", "Failed to save timestamp update", op, err)
	}

	return nil
}
