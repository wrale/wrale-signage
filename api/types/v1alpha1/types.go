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
	DisplaySelector `json:"displaySelector"`
	// Content specifies what content URL to redirect to
	Content ContentRedirect `json:"content"`
	// Schedule optionally restricts when this rule is active
	Schedule *Schedule `json:"schedule,omitempty"`
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

// ListResponse wraps lists of items with metadata
type ListResponse struct {
	// Items contains the listed objects
	Items []interface{} `json:"items"`
	// TotalCount is the total number of matching items
	TotalCount int `json:"totalCount,omitempty"`
}
