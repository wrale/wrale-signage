package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/wrale/wrale-signage/internal/wsignd/display"
	"github.com/wrale/wrale-signage/internal/wsignd/errors"
)

// UpdateLocation updates a display's physical location
func (s *Service) UpdateLocation(ctx context.Context, id uuid.UUID, location display.Location) error {
	const op = "DisplayService.UpdateLocation"

	// Get current display state
	d, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.IsNotFound(err) {
			return errors.NewError("NOT_FOUND", fmt.Sprintf("Display not found: %s", id), op, err)
		}
		return errors.NewError("LOOKUP_FAILED", "Failed to retrieve display", op, err)
	}

	// Update location through domain model
	if err := d.UpdateLocation(location); err != nil {
		return err // Domain model errors are already properly typed
	}

	// Persist changes
	if err := s.repo.Save(ctx, d); err != nil {
		if errors.IsVersionMismatch(err) {
			return errors.NewError("VERSION_CONFLICT", "Display was modified", op, err)
		}
		return errors.NewError("SAVE_FAILED", "Failed to save location update", op, err)
	}

	// Publish location changed event
	event := display.Event{
		Type:      display.EventLocationChanged,
		DisplayID: d.ID,
		Timestamp: time.Now(),
		Data: map[string]string{
			"siteId":   d.Location.SiteID,
			"zone":     d.Location.Zone,
			"position": d.Location.Position,
			"version":  fmt.Sprint(d.Version),
		},
	}

	if err := s.publisher.Publish(ctx, event); err != nil {
		// Log but don't fail the operation if event publishing fails
		// TODO: Add proper logging
		fmt.Printf("Failed to publish location changed event: %v\n", err)
	}

	return nil
}
