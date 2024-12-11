package service

import (
	"context"

	"github.com/wrale/wrale-signage/internal/wsignd/display"
)

// NoopEventPublisher is a no-op implementation of display.EventPublisher
type NoopEventPublisher struct{}

func NewNoopEventPublisher() *NoopEventPublisher {
	return &NoopEventPublisher{}
}

func (p *NoopEventPublisher) Publish(ctx context.Context, event display.Event) error {
	return nil
}
