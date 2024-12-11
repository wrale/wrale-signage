package content

import (
	"time"

	"github.com/google/uuid"
)

type EventType string

const (
	EventContentLoaded     EventType = "CONTENT_LOADED"
	EventContentError      EventType = "CONTENT_ERROR"
	EventContentVisible    EventType = "CONTENT_VISIBLE"
	EventContentHidden     EventType = "CONTENT_HIDDEN"
	EventContentInteractive EventType = "CONTENT_INTERACTIVE"
)

type Event struct {
	ID        uuid.UUID
	DisplayID uuid.UUID
	Type      EventType
	URL       string
	Timestamp time.Time
	Error     *EventError
	Metrics   *EventMetrics
	Context   map[string]string
}

type EventError struct {
	Code    string
	Message string
	Details map[string]interface{}
}

type EventMetrics struct {
	LoadTime        int64
	RenderTime      int64
	InteractiveTime int64
	ResourceStats   *ResourceStats
}

type ResourceStats struct {
	ImageCount  int
	ScriptCount int
	TotalBytes  int64
}

type EventBatch struct {
	DisplayID uuid.UUID
	Events    []Event
}