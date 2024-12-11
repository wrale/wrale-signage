// Package display implements the display domain model and business logic
package display

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// State represents the current state of a display
type State string

const (
	// StateUnregistered indicates a display that hasn't completed registration
	StateUnregistered State = "UNREGISTERED"
	// StateActive indicates a properly registered and active display
	StateActive State = "ACTIVE"
	// StateOffline indicates a display that hasn't communicated recently
	StateOffline State = "OFFLINE"
	// StateDisabled indicates a manually disabled display
	StateDisabled State = "DISABLED"
)

// Display represents a digital signage display device
type Display struct {
	// ID is the unique identifier for this display
	ID uuid.UUID
	// Name is a human-readable identifier
	Name string
	// Location identifies where this display is physically located
	Location Location
	// State represents the display's current operational state
	State State
	// LastSeen is when the display last contacted the server
	LastSeen time.Time
	// Version tracks optimistic concurrency control
	Version int
	// Properties contains arbitrary key-value pairs for display metadata
	Properties map[string]string
	// ActivationCode holds the display's initial setup code
	ActivationCode string
}

// Location represents where a display is physically located
type Location struct {
	// SiteID identifies the physical location/building
	SiteID string
	// Zone identifies the area within the site (e.g., "entrance", "cafeteria")
	Zone string
	// Position provides additional positioning info within the zone
	Position string
}

// NewDisplay creates a new display with the given name and location
func NewDisplay(name string, location Location) (*Display, error) {
	if name == "" {
		return nil, ErrInvalidName{Name: name, Reason: "display name cannot be empty"}
	}
	if location.SiteID == "" {
		return nil, ErrInvalidLocation{Reason: "site ID cannot be empty"}
	}

	return &Display{
		ID:         uuid.New(),
		Name:       name,
		Location:   location,
		State:      StateUnregistered,
		LastSeen:   time.Now(),
		Version:    0, // Start at 0 for insert
		Properties: make(map[string]string),
	}, nil
}

// Activate transitions the display to the active state
func (d *Display) Activate() error {
	if d.State == StateDisabled {
		return ErrInvalidState{Current: d.State, Target: StateActive}
	}
	d.State = StateActive
	d.Version++
	return nil
}

// Disable transitions the display to the disabled state
func (d *Display) Disable() {
	d.State = StateDisabled
	d.Version++
}

// UpdateLastSeen updates the display's last seen timestamp
func (d *Display) UpdateLastSeen() {
	d.LastSeen = time.Now()
	d.Version++
}

// UpdateLocation updates the display's physical location
func (d *Display) UpdateLocation(location Location) error {
	if location.SiteID == "" {
		return ErrInvalidLocation{Reason: "site ID cannot be empty"}
	}
	d.Location = location
	d.Version++
	return nil
}

// SetProperty sets a display property
func (d *Display) SetProperty(key, value string) {
	if d.Properties == nil {
		d.Properties = make(map[string]string)
	}
	d.Properties[key] = value
	d.Version++
}

// ValidateActivationCode checks if the given code matches this display
func (d *Display) ValidateActivationCode(code string) error {
	// An empty code means no validation needed yet
	if d.ActivationCode == "" {
		return nil
	}
	if code != d.ActivationCode {
		return fmt.Errorf("invalid activation code")
	}
	return nil
}

// SetActivationCode sets the display's activation code
func (d *Display) SetActivationCode(code string) {
	d.ActivationCode = code
	d.Version++
}
