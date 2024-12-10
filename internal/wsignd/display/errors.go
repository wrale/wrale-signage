// Package display implements the display domain model and business logic
package display

import "fmt"

// ErrVersionMismatch indicates a concurrent modification conflict
type ErrVersionMismatch struct {
	ID string
}

func (e ErrVersionMismatch) Error() string {
	return fmt.Sprintf("version mismatch for display %s: concurrent modification detected", e.ID)
}

// ErrNotFound indicates a display lookup failure
type ErrNotFound struct {
	ID string
}

func (e ErrNotFound) Error() string {
	return fmt.Sprintf("display not found: %s", e.ID)
}

// ErrInvalidState indicates an invalid state transition
type ErrInvalidState struct {
	Current State
	Target  State
}

func (e ErrInvalidState) Error() string {
	return fmt.Sprintf("invalid state transition from %s to %s", e.Current, e.Target)
}

// ErrInvalidName indicates an invalid display name
type ErrInvalidName struct {
	Name   string
	Reason string
}

func (e ErrInvalidName) Error() string {
	return fmt.Sprintf("invalid display name %q: %s", e.Name, e.Reason)
}

// ErrInvalidLocation indicates an invalid display location
type ErrInvalidLocation struct {
	Reason string
}

func (e ErrInvalidLocation) Error() string {
	return fmt.Sprintf("invalid display location: %s", e.Reason)
}
