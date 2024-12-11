// Package v1alpha1 contains API types for the Wrale Signage system.
package v1alpha1

import (
	"fmt"
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

// RuleFilter defines criteria for filtering redirect rules
type RuleFilter struct {
	// DisplaySelector contains location-based filtering criteria
	DisplaySelector `json:"displaySelector"`
}

// RedirectRuleUpdate specifies changes to an existing redirect rule
type RedirectRuleUpdate struct {
	// Priority is the new rule priority (nil means no change)
	Priority *int `json:"priority,omitempty"`
	// DisplaySelector contains updated location selectors (nil means no change)
	DisplaySelector *DisplaySelector `json:"displaySelector,omitempty"`
	// Content contains updated content redirect (nil means no change)
	Content *ContentRedirect `json:"content,omitempty"`
	// Schedule contains updated scheduling (nil means no change, empty means remove schedule)
	Schedule *Schedule `json:"schedule,omitempty"`
}

// RuleOrderUpdate specifies how to change a rule's position in the evaluation order
type RuleOrderUpdate struct {
	// Position specifies where to move the rule ("before", "after", "start", "end")
	Position string `json:"position"`
	// RelativeTo is the name of the reference rule for before/after positions
	RelativeTo string `json:"relativeTo,omitempty"`
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

// DisplaySelector identifies displays by their location attributes
type DisplaySelector struct {
	// SiteID identifies a physical location
	SiteID string `json:"siteId,omitempty"`
	// Zone identifies an area within a site
	Zone string `json:"zone,omitempty"`
	// Position identifies a specific spot within a zone
	Position string `json:"position,omitempty"`
}

// ListResponse wraps lists of items with metadata
type ListResponse struct {
	// Items contains the listed objects
	Items []interface{} `json:"items"`
	// TotalCount is the total number of matching items
	TotalCount int `json:"totalCount,omitempty"`
}

// Error represents an API error response
type Error struct {
	// Code is a machine-readable error code
	Code string `json:"code"`
	// Message is a human-readable error description
	Message string `json:"message"`
	// Details contains additional error context
	Details interface{} `json:"details,omitempty"`
}

// Error implements the error interface
func (e *Error) Error() string {
	if e.Details != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}
