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

	d, err := s.repo.FindByID(ctx, id)
	if err != nil {
		s.logger.Error("failed to find display", "error", err, "id", id)
		return nil, errors.NewError("NOT_FOUND", fmt.Sprintf("Display not found: %s", id), op, err)
	}

	currentVersion := d.Version
	if err := d.Activate(); err != nil {
		s.logger.Error("failed to activate display", "error", err, "id", id, "state", d.State)
		return nil, err
	}

	// Ensure version is properly incremented
	d.Version = currentVersion
	if err := s.repo.Save(ctx, d); err != nil {
		s.logger.Error("failed to save display", "error", err, "id", id)
		return nil, errors.NewError("SAVE_FAILED", "Failed to save state change", op, err)
	}

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
		s.logger.Error("failed to publish event", "error", err, "id", id)
	}

	return d, nil
}

// Disable transitions a display to the disabled state
func (s *Service) Disable(ctx context.Context, id uuid.UUID) error {
	const op = "DisplayService.Disable"

	d, err := s.repo.FindByID(ctx, id)
	if err != nil {
		s.logger.Error("failed to find display", "error", err, "id", id)
		return errors.NewError("NOT_FOUND", fmt.Sprintf("Display not found: %s", id), op, err)
	}

	currentVersion := d.Version
	d.Disable()
	d.Version = currentVersion

	if err := s.repo.Save(ctx, d); err != nil {
		s.logger.Error("failed to save display", "error", err, "id", id)
		return errors.NewError("SAVE_FAILED", "Failed to save state change", op, err)
	}

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
		s.logger.Error("failed to publish event", "error", err, "id", id)
	}

	return nil
}

// UpdateLastSeen updates the display's last seen timestamp
func (s *Service) UpdateLastSeen(ctx context.Context, id uuid.UUID) error {
	const op = "DisplayService.UpdateLastSeen"

	d, err := s.repo.FindByID(ctx, id)
	if err != nil {
		s.logger.Error("failed to find display", "error", err, "id", id)
		return errors.NewError("NOT_FOUND", fmt.Sprintf("Display not found: %s", id), op, err)
	}

	currentVersion := d.Version
	d.UpdateLastSeen()
	d.Version = currentVersion

	if err := s.repo.Save(ctx, d); err != nil {
		s.logger.Error("failed to save display", "error", err, "id", id)
		return errors.NewError("SAVE_FAILED", "Failed to save timestamp update", op, err)
	}

	return nil
}
