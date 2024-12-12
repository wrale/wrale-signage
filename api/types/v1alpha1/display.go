// Package v1alpha1 contains API types for the Wrale Signage system.
package v1alpha1

import (
	"time"

	"github.com/google/uuid"
)

// DisplayState represents the possible states of a display
type DisplayState string

const (
	// DisplayStateUnregistered indicates a display that hasn't completed registration
	DisplayStateUnregistered DisplayState = "UNREGISTERED"
	// DisplayStateActive indicates a properly registered and active display
	DisplayStateActive DisplayState = "ACTIVE"
	// DisplayStateOffline indicates a display that hasn't communicated recently
	DisplayStateOffline DisplayState = "OFFLINE"
	// DisplayStateDisabled indicates a manually disabled display
	DisplayStateDisabled DisplayState = "DISABLED"
)

// DisplayLocation represents where a display is physically located
type DisplayLocation struct {
	// SiteID identifies the physical location/building
	SiteID string `json:"siteId"`
	// Zone identifies the area within the site (e.g., "entrance", "cafeteria")
	Zone string `json:"zone"`
	// Position provides additional positioning info within the zone
	Position string `json:"position"`
}

// Display represents a digital signage display in the system
type Display struct {
	// TypeMeta describes the versioning of this object
	TypeMeta `json:",inline"`
	// ObjectMeta provides metadata about the display
	ObjectMeta `json:"metadata,omitempty"`

	// Spec holds the desired state of this display
	Spec DisplaySpec `json:"spec"`
	// Status holds the current state of this display
	Status DisplayStatus `json:"status"`
}

// DisplaySpec defines the desired state of a Display
type DisplaySpec struct {
	// Location defines where this display is physically located
	Location DisplayLocation `json:"location"`
	// Properties contains arbitrary key-value pairs for display metadata
	Properties map[string]string `json:"properties,omitempty"`
}

// DisplayStatus defines the observed state of a Display
type DisplayStatus struct {
	// State indicates the display's current operational state
	State DisplayState `json:"state"`
	// LastSeen is when the display last contacted the server
	LastSeen time.Time `json:"lastSeen"`
	// Version tracks optimistic concurrency control
	Version int `json:"version"`
}

// TypeMeta describes an individual object's type and API version
type TypeMeta struct {
	// Kind is a string value representing the type of this object
	Kind string `json:"kind,omitempty"`
	// APIVersion defines the versioned schema of this object
	APIVersion string `json:"apiVersion,omitempty"`
}

// ObjectMeta is metadata that all persisted resources must have
type ObjectMeta struct {
	// ID uniquely identifies this object
	ID uuid.UUID `json:"id,omitempty"`
	// Name is a human-readable identifier for this object
	Name string `json:"name"`
	// CreatedAt indicates when this object was created
	CreatedAt time.Time `json:"createdAt,omitempty"`
	// UpdatedAt indicates when this object was last modified
	UpdatedAt time.Time `json:"updatedAt,omitempty"`
}

// DisplayRegistrationRequest represents a request to register a new display
type DisplayRegistrationRequest struct {
	// Name is the desired name for the display
	Name string `json:"name"`
	// Location specifies where the display will be located
	Location DisplayLocation `json:"location"`
	// ActivationCode is the code shown on the display
	ActivationCode string `json:"activationCode"`
}

// DisplayFilter defines criteria for listing displays
type DisplayFilter struct {
	// SiteID filters by location site ID
	SiteID string `json:"siteId,omitempty"`
	// Zone filters by location zone
	Zone string `json:"zone,omitempty"`
	// States filters by display states
	States []DisplayState `json:"states,omitempty"`
}

// DisplayList is a list of displays
type DisplayList struct {
	// TypeMeta describes the versioning of this object
	TypeMeta `json:",inline"`

	// Items is the list of Display objects
	Items []Display `json:"items"`
}

// DisplayUpdateRequest represents a request to update a display
type DisplayUpdateRequest struct {
	// Location updates the display's physical location
	Location *DisplayLocation `json:"location,omitempty"`
	// Properties updates the display's metadata
	Properties map[string]string `json:"properties,omitempty"`
}

// ContentAssignment represents content assigned to displays
type ContentAssignment struct {
	// TypeMeta describes the versioning of this object
	TypeMeta `json:",inline"`
	// ObjectMeta provides metadata about the assignment
	ObjectMeta `json:"metadata,omitempty"`

	// DisplaySelector specifies which displays this content targets
	DisplaySelector DisplaySelector `json:"displaySelector"`
	// ContentURL is the URL where displays can fetch the content
	ContentURL string `json:"contentUrl"`
	// ValidFrom indicates when this content becomes active
	ValidFrom *time.Time `json:"validFrom,omitempty"`
	// ValidUntil indicates when this content expires
	ValidUntil *time.Time `json:"validUntil,omitempty"`
}

// ContentAssignmentList is a list of content assignments
type ContentAssignmentList struct {
	// TypeMeta describes the versioning of this object
	TypeMeta `json:",inline"`

	// Items is the list of ContentAssignment objects
	Items []ContentAssignment `json:"items"`
}
