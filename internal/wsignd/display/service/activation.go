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

	s.logger.Info("activating display",
		"displayID", id,
		"operation", op,
	)

	// Get current display state
	d, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.IsNotFound(err) {
			s.logger.Error("display not found",
				"error", err,
				"displayID", id,
				"operation", op,
			)
			return nil, errors.NewError("NOT_FOUND", fmt.Sprintf("Display not found: %s", id), op, err)
		}
		s.logger.Error("failed to retrieve display",
			"error", err,
			"displayID", id,
			"operation", op,
		)
		return nil, errors.NewError("LOOKUP_FAILED", "Failed to retrieve display", op, err)
	}

	s.logger.Info("found display",
		"displayID", id,
		"name", d.Name,
		"currentState", d.State,
		"currentVersion", d.Version,
		"operation", op,
	)

	// Activate through domain model
	if err := d.Activate(); err != nil {
		s.logger.Error("failed to transition state",
			"error", err,
			"displayID", id,
			"currentState", d.State,
			"operation", op,
		)
		return nil, err // Domain model errors are already properly typed
	}

	// Persist changes
	if err := s.repo.Save(ctx, d); err != nil {
		if errors.IsVersionMismatch(err) {
			s.logger.Error("version conflict during save",
				"error", err,
				"displayID", id,
				"version", d.Version,
				"operation", op,
			)
			return nil, errors.NewError("VERSION_CONFLICT", "Display was modified", op, err)
		}
		s.logger.Error("failed to save state change",
			"error", err,
			"displayID", id,
			"newState", d.State,
			"version", d.Version,
			"operation", op,
		)
		return nil, errors.NewError("SAVE_FAILED", "Failed to save state change", op, err)
	}

	s.logger.Info("activated display",
		"displayID", id,
		"name", d.Name,
		"newState", d.State,
		"newVersion", d.Version,
		"operation", op,
	)

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
		s.logger.Error("failed to publish event",
			"error", err,
			"displayID", id,
			"eventType", event.Type,
			"operation", op,
		)
	}

	return d, nil
}

// Disable transitions a display to the disabled state
func (s *Service) Disable(ctx context.Context, id uuid.UUID) error {
	const op = "DisplayService.Disable"

	s.logger.Info("disabling display",
		"displayID", id,
		"operation", op,
	)

	// Get current display state
	d, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.IsNotFound(err) {
			s.logger.Error("display not found",
				"error", err,
				"displayID", id,
				"operation", op,
			)
			return errors.NewError("NOT_FOUND", fmt.Sprintf("Display not found: %s", id), op, err)
		}
		s.logger.Error("failed to retrieve display",
			"error", err,
			"displayID", id,
			"operation", op,
		)
		return errors.NewError("LOOKUP_FAILED", "Failed to retrieve display", op, err)
	}

	s.logger.Info("found display",
		"displayID", id,
		"name", d.Name,
		"currentState", d.State,
		"currentVersion", d.Version,
		"operation", op,
	)

	// Disable through domain model
	d.Disable()

	// Persist changes
	if err := s.repo.Save(ctx, d); err != nil {
		if errors.IsVersionMismatch(err) {
			s.logger.Error("version conflict during save",
				"error", err,
				"displayID", id,
				"version", d.Version,
				"operation", op,
			)
			return errors.NewError("VERSION_CONFLICT", "Display was modified", op, err)
		}
		s.logger.Error("failed to save state change",
			"error", err,
			"displayID", id,
			"newState", d.State,
			"version", d.Version,
			"operation", op,
		)
		return errors.NewError("SAVE_FAILED", "Failed to save state change", op, err)
	}

	s.logger.Info("disabled display",
		"displayID", id,
		"name", d.Name,
		"newState", d.State,
		"newVersion", d.Version,
		"operation", op,
	)

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
		s.logger.Error("failed to publish event",
			"error", err,
			"displayID", id,
			"eventType", event.Type,
			"operation", op,
		)
	}

	return nil
}

// UpdateLastSeen updates the display's last seen timestamp
func (s *Service) UpdateLastSeen(ctx context.Context, id uuid.UUID) error {
	const op = "DisplayService.UpdateLastSeen"

	s.logger.Info("updating last seen timestamp",
		"displayID", id,
		"operation", op,
	)

	// Get current display state
	d, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.IsNotFound(err) {
			s.logger.Error("display not found",
				"error", err,
				"displayID", id,
				"operation", op,
			)
			return errors.NewError("NOT_FOUND", fmt.Sprintf("Display not found: %s", id), op, err)
		}
		s.logger.Error("failed to retrieve display",
			"error", err,
			"displayID", id,
			"operation", op,
		)
		return errors.NewError("LOOKUP_FAILED", "Failed to retrieve display", op, err)
	}

	// Update timestamp through domain model
	d.UpdateLastSeen()

	// Persist changes with retry on version conflicts
	if err := s.repo.Save(ctx, d); err != nil {
		if errors.IsVersionMismatch(err) {
			s.logger.Error("version conflict during save",
				"error", err,
				"displayID", id,
				"version", d.Version,
				"operation", op,
			)
			return errors.NewError("VERSION_CONFLICT", "Display was modified", op, err)
		}
		s.logger.Error("failed to save timestamp update",
			"error", err,
			"displayID", id,
			"version", d.Version,
			"operation", op,
		)
		return errors.NewError("SAVE_FAILED", "Failed to save timestamp update", op, err)
	}

	s.logger.Info("updated last seen timestamp",
		"displayID", id,
		"name", d.Name,
		"lastSeen", d.LastSeen,
		"version", d.Version,
		"operation", op,
	)

	return nil
}
