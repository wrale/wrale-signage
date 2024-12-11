// Package service implements the business logic for display management
package service

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
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

// Get retrieves a display by ID
func (s *Service) Get(ctx context.Context, id uuid.UUID) (*display.Display, error) {
	return s.repo.FindByID(ctx, id)
}

// GetByName retrieves a display by name
func (s *Service) GetByName(ctx context.Context, name string) (*display.Display, error) {
	return s.repo.FindByName(ctx, name)
}
