// Package service implements the business logic for display management
package service

import (
	"log/slog"

	"github.com/wrale/wrale-signage/internal/wsignd/display"
)

// service implements the display.Service interface
type Service struct {
	repo      display.Repository
	publisher display.EventPublisher
	logger    *slog.Logger
}

// New creates a new display service instance
func New(repo display.Repository, publisher display.EventPublisher, logger *slog.Logger) display.Service {
	return &Service{
		repo:      repo,
		publisher: publisher,
		logger:    logger,
	}
}
