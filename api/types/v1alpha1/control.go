package v1alpha1

import "time"

// ControlMessageType defines types of control messages
type ControlMessageType string

const (
	// ControlMessageSequenceUpdate indicates a content sequence change
	ControlMessageSequenceUpdate ControlMessageType = "SEQUENCE_UPDATE"
	// ControlMessageReload indicates display should reload device URL
	ControlMessageReload ControlMessageType = "RELOAD"
	// ControlMessageStatus indicates display status report
	ControlMessageStatus ControlMessageType = "STATUS"
)

// ControlMessage represents a message sent over display control WebSocket
type ControlMessage struct {
	// TypeMeta describes API version details
	TypeMeta `json:",inline"`
	// Type indicates the kind of control message
	Type ControlMessageType `json:"type"`
	// Sequence contains content sequence details if applicable
	Sequence *ContentSequence `json:"sequence,omitempty"`
	// Timestamp indicates when message was created
	Timestamp time.Time `json:"timestamp"`
	// Error contains error details if applicable
	Error *ControlError `json:"error,omitempty"`
	// Status contains display status if applicable
	Status *ControlStatus `json:"status,omitempty"`
}

// ContentSequence defines ordered content items to display
type ContentSequence struct {
	// Items is the ordered list of content to display
	Items []ContentItem `json:"items"`
}

// ContentItem represents a single item in display sequence
type ContentItem struct {
	// URL points to cacheable content location
	URL string `json:"url"`
	// Duration specifies how long to show content
	Duration ContentDuration `json:"duration"`
	// Transition defines how to switch to next content
	Transition ContentTransition `json:"transition"`
}

// ContentDuration specifies content display timing
type ContentDuration struct {
	// Type indicates if duration is fixed or video-length
	Type string `json:"type"`
	// Value specifies seconds if Type is "fixed"
	Value int `json:"value,omitempty"`
}

// ContentTransition defines transition between content items
type ContentTransition struct {
	// Type indicates transition animation style
	Type string `json:"type"`
	// Duration specifies transition length in milliseconds
	Duration int `json:"duration"`
}

// ControlError represents control message errors
type ControlError struct {
	// Code provides error classification
	Code string `json:"code"`
	// Message provides error details
	Message string `json:"message"`
}

// ControlStatus represents current display state for control messages
type ControlStatus struct {
	// CurrentURL indicates content being shown
	CurrentURL string `json:"currentUrl"`
	// State indicates display operational state
	State DisplayState `json:"state"`
	// LastError contains most recent error if any
	LastError *string `json:"lastError,omitempty"`
	// UpdatedAt indicates when status was generated
	UpdatedAt time.Time `json:"updatedAt"`
}
