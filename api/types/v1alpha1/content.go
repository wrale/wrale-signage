// Package v1alpha1 contains API types for the Wrale Signage system
package v1alpha1

import (
	"time"
)

// ContentSource represents a source of content for displays
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
	// URL is where the content can be fetched
	URL string `json:"url"`
	// Type identifies what kind of content this is (e.g., "welcome", "menu")
	Type string `json:"type"`
	// Properties contains additional metadata about the content
	Properties map[string]string `json:"properties,omitempty"`
}

// ContentSourceStatus defines the observed state of a ContentSource
type ContentSourceStatus struct {
	// LastValidated indicates when the content was last validated
	LastValidated time.Time `json:"lastValidated"`
	// Hash is a content-based identifier for caching
	Hash string `json:"hash"`
	// Version tracks updates to this content source
	Version int `json:"version"`
}

// ContentSourceUpdate represents a partial update to a content source
type ContentSourceUpdate struct {
	// URL updates where the content can be fetched
	URL *string `json:"url,omitempty"`
	// Properties updates the content metadata
	Properties map[string]string `json:"properties,omitempty"`
}

// ContentSourceList is a list of content sources
type ContentSourceList struct {
	// TypeMeta describes the versioning of this object
	TypeMeta `json:",inline"`

	// Items is the list of ContentSource objects
	Items []ContentSource `json:"items"`
}
