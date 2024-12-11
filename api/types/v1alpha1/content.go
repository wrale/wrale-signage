// Package v1alpha1 contains API types for the Wrale Signage system
package v1alpha1

import (
	"net/url"
	"time"
)

// ContentSource represents a content source that can be used by displays
type ContentSource struct {
	// TypeMeta describes the versioning of this object
	TypeMeta `json:",inline"`
	// ObjectMeta provides metadata about the content source
	ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state of this content source
	Spec ContentSourceSpec `json:"spec"`
	// Status holds the current state of this content source
	Status ContentSourceStatus `json:"status"`
}

// ContentSourceSpec defines the desired state of a ContentSource
type ContentSourceSpec struct {
	// URL is the fully qualified URL where the content can be fetched
	URL string `json:"url"`
	// Type identifies what kind of content this is (e.g., "welcome", "menu")
	Type string `json:"type"`
	// Properties contains additional metadata about the content
	Properties map[string]string `json:"properties,omitempty"`
	// PlaybackDuration specifies how long to display this content (0 means video length or indefinite)
	PlaybackDuration time.Duration `json:"playbackDuration,omitempty"`
}

// Validate checks the ContentSourceSpec for validity
func (s *ContentSourceSpec) Validate() error {
	if s.URL == "" {
		return &Error{Code: "InvalidContent", Message: "content URL is required"}
	}
	if _, err := url.ParseRequestURI(s.URL); err != nil {
		return &Error{Code: "InvalidContent", Message: "invalid content URL"}
	}
	if s.Type == "" {
		return &Error{Code: "InvalidContent", Message: "content type is required"}
	}
	return nil
}

// ContentSourceStatus defines the observed state of a ContentSource
type ContentSourceStatus struct {
	// LastValidated indicates when the content was last validated
	LastValidated time.Time `json:"lastValidated"`
	// Hash is a content-based identifier for caching
	Hash string `json:"hash"`
	// Version tracks updates to this content source
	Version int `json:"version"`
	// IsHealthy indicates if the content is currently accessible
	IsHealthy bool `json:"isHealthy"`
	// LastError captures the most recent error, if any
	LastError string `json:"lastError,omitempty"`
}

// ContentSourceUpdate represents a partial update to a content source
type ContentSourceUpdate struct {
	// URL updates where the content can be fetched
	URL *string `json:"url,omitempty"`
	// Properties updates the content metadata
	Properties map[string]string `json:"properties,omitempty"`
	// PlaybackDuration updates how long to display the content
	PlaybackDuration *time.Duration `json:"playbackDuration,omitempty"`
}

// Validate checks the ContentSourceUpdate for validity
func (u *ContentSourceUpdate) Validate() error {
	if u.URL != nil {
		if *u.URL == "" {
			return &Error{Code: "InvalidContent", Message: "content URL cannot be empty"}
		}
		if _, err := url.ParseRequestURI(*u.URL); err != nil {
			return &Error{Code: "InvalidContent", Message: "invalid content URL"}
		}
	}
	return nil
}

// ContentSourceList is a list of content sources
type ContentSourceList struct {
	// TypeMeta describes the versioning of this object
	TypeMeta `json:",inline"`

	// Items is the list of ContentSource objects
	Items []ContentSource `json:"items"`
}
