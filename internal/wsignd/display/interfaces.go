package display

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Repository defines the interface for display persistence
type Repository interface {
	// Save persists a display to storage
	Save(ctx context.Context, display *Display) error

	// FindByID retrieves a display by its unique identifier
	FindByID(ctx context.Context, id uuid.UUID) (*Display, error)

	// FindByName retrieves a display by its name
	FindByName(ctx context.Context, name string) (*Display, error)

	// List retrieves displays matching the given filter
	List(ctx context.Context, filter DisplayFilter) ([]*Display, error)

	// Delete removes a display from storage
	Delete(ctx context.Context, id uuid.UUID) error
}

// DisplayFilter defines criteria for listing displays
type DisplayFilter struct {
	// SiteID filters by location site ID
	SiteID string
	// Zone filters by location zone
	Zone string
	// States filters by display states
	States []State
}

// Service defines the interface for display business operations
type Service interface {
	// Register creates a new display
	Register(ctx context.Context, name string, location Location) (*Display, error)

	// Get retrieves a display by ID
	Get(ctx context.Context, id uuid.UUID) (*Display, error)

	// GetByName retrieves a display by name
	GetByName(ctx context.Context, name string) (*Display, error)

	// List retrieves displays matching the filter
	List(ctx context.Context, filter DisplayFilter) ([]*Display, error)

	// UpdateLocation updates a display's physical location
	UpdateLocation(ctx context.Context, id uuid.UUID, location Location) error

	// Activate transitions a display to the active state
	Activate(ctx context.Context, id uuid.UUID) (*Display, error)

	// Disable transitions a display to the disabled state
	Disable(ctx context.Context, id uuid.UUID) error

	// UpdateLastSeen updates the display's last seen timestamp
	UpdateLastSeen(ctx context.Context, id uuid.UUID) error

	// SetProperty sets a display property
	SetProperty(ctx context.Context, id uuid.UUID, key, value string) error
}

// EventType represents types of display events
type EventType string

const (
	// EventRegistered indicates a new display registration
	EventRegistered EventType = "REGISTERED"
	// EventActivated indicates a display activation
	EventActivated EventType = "ACTIVATED"
	// EventDisabled indicates a display was disabled
	EventDisabled EventType = "DISABLED"
	// EventLocationChanged indicates a display location change
	EventLocationChanged EventType = "LOCATION_CHANGED"
)

// Event represents something that happened to a display
type Event struct {
	// Type indicates what kind of event occurred
	Type EventType
	// DisplayID identifies which display the event is about
	DisplayID uuid.UUID
	// Timestamp records when the event occurred
	Timestamp time.Time
	// Data contains event-specific details
	Data map[string]string
}

// EventPublisher defines the interface for publishing display events
type EventPublisher interface {
	// Publish sends an event to interested subscribers
	Publish(ctx context.Context, event Event) error
}
