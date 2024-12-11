package service

import (
	"context"
	"fmt"
	"time"

	"github.com/wrale/wrale-signage/internal/wsignd/display"
	"github.com/wrale/wrale-signage/internal/wsignd/errors"
)

// Register creates a new display with the given name and location
func (s *Service) Register(ctx context.Context, name string, location display.Location) (*display.Display, error) {
	const op = "DisplayService.Register"

	// Check if display already exists with this name
	existing, err := s.repo.FindByName(ctx, name)
	if err != nil && !errors.IsNotFound(err) {
		return nil, errors.NewError("REGISTRATION_FAILED", "Failed to check existing display", op, err)
	}
	if existing != nil {
		return nil, errors.NewError("DISPLAY_EXISTS", fmt.Sprintf("Display already exists with name: %s", name), op, errors.ErrConflict)
	}

	// Create new display through domain model
	d, err := display.NewDisplay(name, location)
	if err != nil {
		// Domain model errors are already properly typed
		return nil, err
	}

	// Persist the new display
	if err := s.repo.Save(ctx, d); err != nil {
		return nil, errors.NewError("SAVE_FAILED", "Failed to save display", op, err)
	}

	// Publish registration event
	event := display.Event{
		Type:      display.EventRegistered,
		DisplayID: d.ID,
		Timestamp: time.Now(),
		Data: map[string]string{
			"name":    d.Name,
			"siteId":  d.Location.SiteID,
			"zone":    d.Location.Zone,
			"state":   string(d.State),
			"version": fmt.Sprint(d.Version),
		},
	}

	if err := s.publisher.Publish(ctx, event); err != nil {
		// Log but don't fail the operation if event publishing fails
		// TODO: Add proper logging
		fmt.Printf("Failed to publish registration event: %v\n", err)
	}

	return d, nil
}
