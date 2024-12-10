// Package v1alpha1 contains API types for the Wrale Signage system.
package v1alpha1

import (
	"time"
)

// RedirectRule defines how to map display requests to content URLs
type RedirectRule struct {
	// Name identifies this rule for management and logging
	Name string `json:"name"`
	// Priority determines rule ordering (higher numbers take precedence)
	Priority int `json:"priority"`
	// DisplaySelector identifies which displays this rule applies to
	DisplaySelector DisplaySelector `json:"displaySelector"`
	// Content specifies what content URL to redirect to
	Content ContentRedirect `json:"content"`
	// Schedule optionally restricts when this rule is active
	Schedule *Schedule `json:"schedule,omitempty"`
}

// DisplaySelector specifies which displays a rule applies to
type DisplaySelector struct {
	// SiteID matches displays at a specific site
	SiteID string `json:"siteId,omitempty"`
	// Zone matches displays in a specific zone
	Zone string `json:"zone,omitempty"`
	// Position matches displays at a specific position
	Position string `json:"position,omitempty"`
}

// ContentRedirect specifies where to redirect matching displays
type ContentRedirect struct {
	// ContentType identifies the type of content (e.g., "welcome", "menu", "emergency")
	ContentType string `json:"contentType"`
	// Version identifies the content version (e.g., "current", "2024-spring")
	Version string `json:"version"`
	// Hash identifies the specific content revision
	Hash string `json:"hash"`
}

// Schedule defines when a rule is active
type Schedule struct {
	// ActiveFrom is when the rule becomes active (inclusive)
	ActiveFrom *time.Time `json:"activeFrom,omitempty"`
	// ActiveUntil is when the rule becomes inactive (exclusive)
	ActiveUntil *time.Time `json:"activeUntil,omitempty"`
	// DaysOfWeek restricts which days the rule is active
	DaysOfWeek []time.Weekday `json:"daysOfWeek,omitempty"`
	// TimeOfDay restricts times during active days
	TimeOfDay *TimeRange `json:"timeOfDay,omitempty"`
}

// TimeRange represents a time period within a day
type TimeRange struct {
	// Start is when the range begins (e.g., "09:00")
	Start string `json:"start"`
	// End is when the range ends (e.g., "17:00")
	End string `json:"end"`
}

// DisplayRegistration holds data for registering a new display
type DisplayRegistration struct {
	// ActivationCode is shown on the display during setup
	ActivationCode string `json:"activationCode"`
	// Location specifies where the display is located
	Location DisplayLocation `json:"location"`
	// Properties contains optional display metadata
	Properties map[string]string `json:"properties,omitempty"`
}

// DisplayLocation describes where a display is physically located
type DisplayLocation struct {
	// SiteID identifies the physical building/location
	SiteID string `json:"siteId"`
	// Zone identifies the area within the site
	Zone string `json:"zone"`
	// Position provides placement details within the zone
	Position string `json:"position"`
}

// DisplayStatus provides information about a registered display
type DisplayStatus struct {
	// GUID is the display's unique identifier
	GUID string `json:"guid"`
	// Location is where the display is located
	Location DisplayLocation `json:"location"`
	// LastSeen is when the display last contacted the server
	LastSeen *time.Time `json:"lastSeen,omitempty"`
	// CurrentContent describes what content is being shown
	CurrentContent *DisplayContent `json:"currentContent,omitempty"`
	// Properties contains display metadata
	Properties map[string]string `json:"properties,omitempty"`
}

// DisplayContent describes content being shown on a display
type DisplayContent struct {
	// RuleName identifies which rule selected this content
	RuleName string `json:"ruleName"`
	// ContentURL is the full URL being displayed
	ContentURL string `json:"contentUrl"`
	// RedirectedAt is when this content was last served
	RedirectedAt time.Time `json:"redirectedAt"`
}

// Error represents an API error response
type Error struct {
	// Code is a machine-readable error code
	Code string `json:"code"`
	// Message is a human-readable error description
	Message string `json:"message"`
}

// ListResponse wraps lists of items with metadata
type ListResponse struct {
	// Items contains the listed objects
	Items []interface{} `json:"items"`
	// TotalCount is the total number of matching items
	TotalCount int `json:"totalCount,omitempty"`
}
