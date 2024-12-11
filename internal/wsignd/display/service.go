// Package display implements the display domain model and business logic
package display

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/wrale/wrale-signage/internal/wsignd/errors"
)

// service implements the display.Service interface by coordinating between
// the domain model, repository, and event publisher while enforcing business rules.
type service struct {
	repo      Repository
	publisher EventPublisher
}

// NewService creates a new display service instance.
func NewService(repo Repository, publisher EventPublisher) Service {
	return &service{
		repo:      repo,
		publisher: publisher,
	}
}

// Register creates a new display with the given name and location.
func (s *service) Register(ctx context.Context, name string, location Location) (*Display, error) {
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
	display, err := NewDisplay(name, location)
	if err != nil {
		return nil, errors.NewError("INVALID_INPUT", "Failed to create display", op, err)
	}

	// Persist the new display
	if err := s.repo.Save(ctx, display); err != nil {
		return nil, errors.NewError("SAVE_FAILED", "Failed to save display", op, err)
	}

	// Publish registration event
	event := Event{
		Type:      EventRegistered,
		DisplayID: display.ID,
		Timestamp: time.Now(),
		Data: map[string]string{
			"name":    display.Name,
			"siteId":  display.Location.SiteID,
			"zone":    display.Location.Zone,
			"state":   string(display.State),
			"version": fmt.Sprint(display.Version),
		},
	}

	if err := s.publisher.Publish(ctx, event); err != nil {
		// Log but don't fail the operation if event publishing fails
		// TODO: Add proper logging
		fmt.Printf("Failed to publish registration event: %v\n", err)
	}

	return display, nil
}

// Get retrieves a display by ID.
func (s *service) Get(ctx context.Context, id uuid.UUID) (*Display, error) {
	const op = "DisplayService.Get"

	display, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, errors.NewError("NOT_FOUND", fmt.Sprintf("Display not found: %s", id), op, err)
		}
		return nil, errors.NewError("LOOKUP_FAILED", "Failed to retrieve display", op, err)
	}

	return display, nil
}

// List retrieves displays matching the filter.
func (s *service) List(ctx context.Context, filter DisplayFilter) ([]*Display, error) {
	const op = "DisplayService.List"

	displays, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, errors.NewError("LIST_FAILED", "Failed to list displays", op, err)
	}

	return displays, nil
}

// UpdateLocation updates a display's physical location.
func (s *service) UpdateLocation(ctx context.Context, id uuid.UUID, location Location) error {
	const op = "DisplayService.UpdateLocation"

	// Get current display state
	display, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.IsNotFound(err) {
			return errors.NewError("NOT_FOUND", fmt.Sprintf("Display not found: %s", id), op, err)
		}
		return errors.NewError("LOOKUP_FAILED", "Failed to retrieve display", op, err)
	}

	// Update location through domain model
	if err := display.UpdateLocation(location); err != nil {
		return errors.NewError("INVALID_INPUT", "Invalid location update", op, err)
	}

	// Persist changes
	if err := s.repo.Save(ctx, display); err != nil {
		if errors.IsVersionMismatch(err) {
			return errors.NewError("VERSION_CONFLICT", "Display was modified", op, err)
		}
		return errors.NewError("SAVE_FAILED", "Failed to save location update", op, err)
	}

	// Publish location changed event
	event := Event{
		Type:      EventLocationChanged,
		DisplayID: display.ID,
		Timestamp: time.Now(),
		Data: map[string]string{
			"siteId":   display.Location.SiteID,
			"zone":     display.Location.Zone,
			"position": display.Location.Position,
			"version":  fmt.Sprint(display.Version),
		},
	}

	if err := s.publisher.Publish(ctx, event); err != nil {
		// Log but don't fail the operation if event publishing fails
		// TODO: Add proper logging
		fmt.Printf("Failed to publish location changed event: %v\n", err)
	}

	return nil
}

// Activate transitions a display to the active state.
func (s *service) Activate(ctx context.Context, id uuid.UUID) error {
	const op = "DisplayService.Activate"

	// Get current display state
	display, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.IsNotFound(err) {
			return errors.NewError("NOT_FOUND", fmt.Sprintf("Display not found: %s", id), op, err)
		}
		return errors.NewError("LOOKUP_FAILED", "Failed to retrieve display", op, err)
	}

	// Activate through domain model
	if err := display.Activate(); err != nil {
		return errors.NewError("INVALID_STATE", "Cannot activate display", op, err)
	}

	// Persist changes
	if err := s.repo.Save(ctx, display); err != nil {
		if errors.IsVersionMismatch(err) {
			return errors.NewError("VERSION_CONFLICT", "Display was modified", op, err)
		}
		return errors.NewError("SAVE_FAILED", "Failed to save state change", op, err)
	}

	// Publish activated event
	event := Event{
		Type:      EventActivated,
		DisplayID: display.ID,
		Timestamp: time.Now(),
		Data: map[string]string{
			"state":   string(display.State),
			"version": fmt.Sprint(display.Version),
		},
	}

	if err := s.publisher.Publish(ctx, event); err != nil {
		// Log but don't fail the operation if event publishing fails
		// TODO: Add proper logging
		fmt.Printf("Failed to publish activated event: %v\n", err)
	}

	return nil
}

// Disable transitions a display to the disabled state.
func (s *service) Disable(ctx context.Context, id uuid.UUID) error {
	const op = "DisplayService.Disable"

	// Get current display state
	display, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.IsNotFound(err) {
			return errors.NewError("NOT_FOUND", fmt.Sprintf("Display not found: %s", id), op, err)
		}
		return errors.NewError("LOOKUP_FAILED", "Failed to retrieve display", op, err)
	}

	// Disable through domain model
	display.Disable()

	// Persist changes
	if err := s.repo.Save(ctx, display); err != nil {
		if errors.IsVersionMismatch(err) {
			return errors.NewError("VERSION_CONFLICT", "Display was modified", op, err)
		}
		return errors.NewError("SAVE_FAILED", "Failed to save state change", op, err)
	}

	// Publish disabled event
	event := Event{
		Type:      EventDisabled,
		DisplayID: display.ID,
		Timestamp: time.Now(),
		Data: map[string]string{
			"state":   string(display.State),
			"version": fmt.Sprint(display.Version),
		},
	}

	if err := s.publisher.Publish(ctx, event); err != nil {
		// Log but don't fail the operation if event publishing fails
		// TODO: Add proper logging
		fmt.Printf("Failed to publish disabled event: %v\n", err)
	}

	return nil
}

// UpdateLastSeen updates the display's last seen timestamp.
func (s *service) UpdateLastSeen(ctx context.Context, id uuid.UUID) error {
	const op = "DisplayService.UpdateLastSeen"

	// Get current display state
	display, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.IsNotFound(err) {
			return errors.NewError("NOT_FOUND", fmt.Sprintf("Display not found: %s", id), op, err)
		}
		return errors.NewError("LOOKUP_FAILED", "Failed to retrieve display", op, err)
	}

	// Update timestamp through domain model
	display.UpdateLastSeen()

	// Persist changes with retry on version conflicts
	if err := s.repo.Save(ctx, display); err != nil {
		if errors.IsVersionMismatch(err) {
			return errors.NewError("VERSION_CONFLICT", "Display was modified", op, err)
		}
		return errors.NewError("SAVE_FAILED", "Failed to save timestamp update", op, err)
	}

	return nil
}

// SetProperty sets a display property.
func (s *service) SetProperty(ctx context.Context, id uuid.UUID, key, value string) error {
	const op = "DisplayService.SetProperty"

	if key == "" {
		return errors.NewError("INVALID_INPUT", "Property key cannot be empty", op, errors.ErrInvalidInput)
	}

	// Get current display state
	display, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.IsNotFound(err) {
			return errors.NewError("NOT_FOUND", fmt.Sprintf("Display not found: %s", id), op, err)
		}
		return errors.NewError("LOOKUP_FAILED", "Failed to retrieve display", op, err)
	}

	// Update property through domain model
	display.SetProperty(key, value)

	// Persist changes
	if err := s.repo.Save(ctx, display); err != nil {
		if errors.IsVersionMismatch(err) {
			return errors.NewError("VERSION_CONFLICT", "Display was modified", op, err)
		}
		return errors.NewError("SAVE_FAILED", "Failed to save property update", op, err)
	}

	return nil
}
