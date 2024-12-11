package v1alpha1

import (
	"time"

	"github.com/google/uuid"
)

// DisplayEventType represents types of display-related events
type DisplayEventType string

const (
	// DisplayEventRegistered indicates new display registration
	DisplayEventRegistered DisplayEventType = "REGISTERED"
	// DisplayEventActivated indicates display activation
	DisplayEventActivated DisplayEventType = "ACTIVATED"
	// DisplayEventDisabled indicates display was disabled
	DisplayEventDisabled DisplayEventType = "DISABLED"
	// DisplayEventLocationChanged indicates display location change
	DisplayEventLocationChanged DisplayEventType = "LOCATION_CHANGED"
)

// ContentEventType represents types of content-related events
type ContentEventType string

const (
	// ContentEventLoaded indicates content successfully loaded
	ContentEventLoaded ContentEventType = "CONTENT_LOADED"
	// ContentEventError indicates content loading error
	ContentEventError ContentEventType = "CONTENT_ERROR"
	// ContentEventVisible indicates content became visible
	ContentEventVisible ContentEventType = "CONTENT_VISIBLE"
	// ContentEventHidden indicates content was hidden
	ContentEventHidden ContentEventType = "CONTENT_HIDDEN"
	// ContentEventInteractive indicates content became interactive
	ContentEventInteractive ContentEventType = "CONTENT_INTERACTIVE"
)

// DisplayEvent represents a display lifecycle event
type DisplayEvent struct {
	// TypeMeta describes API version details
	TypeMeta `json:",inline"`
	// Type indicates what kind of event occurred
	Type DisplayEventType `json:"type"`
	// DisplayID identifies which display the event is about
	DisplayID uuid.UUID `json:"displayId"`
	// Timestamp records when the event occurred
	Timestamp time.Time `json:"timestamp"`
	// Data contains event-specific details
	Data map[string]string `json:"data,omitempty"`
}

// ContentEvent represents a content lifecycle event
type ContentEvent struct {
	// TypeMeta describes API version details
	TypeMeta `json:",inline"`
	// ID uniquely identifies this event
	ID uuid.UUID `json:"id"`
	// DisplayID identifies the display reporting the event
	DisplayID uuid.UUID `json:"displayId"`
	// Type indicates what kind of event occurred
	Type ContentEventType `json:"type"`
	// URL identifies the content involved
	URL string `json:"url"`
	// Timestamp records when the event occurred
	Timestamp time.Time `json:"timestamp"`
	// Error contains failure details if applicable
	Error *EventError `json:"error,omitempty"`
	// Metrics contains performance data if applicable
	Metrics *EventMetrics `json:"metrics,omitempty"`
	// Context contains additional metadata
	Context map[string]string `json:"context,omitempty"`
}

// EventError represents content event error details
type EventError struct {
	// Code provides error classification
	Code string `json:"code"`
	// Message provides error details
	Message string `json:"message"`
	// Details contains structured error data
	Details map[string]interface{} `json:"details,omitempty"`
}

// EventMetrics contains content performance measurements
type EventMetrics struct {
	// LoadTime measures content loading duration
	LoadTime int64 `json:"loadTime"`
	// RenderTime measures initial render duration
	RenderTime int64 `json:"renderTime"`
	// InteractiveTime measures time until interactive
	InteractiveTime int64 `json:"interactiveTime"`
	// ResourceStats contains resource usage details
	ResourceStats *ResourceStats `json:"resourceStats,omitempty"`
}

// ResourceStats contains content resource usage details
type ResourceStats struct {
	// ImageCount is number of images loaded
	ImageCount int `json:"imageCount"`
	// ScriptCount is number of scripts loaded
	ScriptCount int `json:"scriptCount"`
	// TotalBytes is total content size
	TotalBytes int64 `json:"totalBytes"`
}

// ContentEventBatch represents multiple content events
type ContentEventBatch struct {
	// TypeMeta describes API version details
	TypeMeta `json:",inline"`
	// DisplayID identifies the display reporting events
	DisplayID uuid.UUID `json:"displayId"`
	// Events contains the batch of events
	Events []ContentEvent `json:"events"`
}
