// Package service implements the business logic for display management
package service

import (
	"github.com/wrale/wrale-signage/internal/wsignd/display"
)

// service implements the display.Service interface
type Service struct {
	repo      display.Repository
	publisher display.EventPublisher
}

// New creates a new display service instance
func New(repo display.Repository, publisher display.EventPublisher) display.Service {
	return &Service{
		repo:      repo,
		publisher: publisher,
	}
}
