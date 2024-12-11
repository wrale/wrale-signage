package content

import (
	"context"
	"time"

	"github.com/wrale/wrale-signage/api/types/v1alpha1"
)

type contentService struct {
	repo      Repository
	processor EventProcessor
	metrics   MetricsAggregator
	monitor   HealthMonitor
}

func NewService(repo Repository, processor EventProcessor, metrics MetricsAggregator, monitor HealthMonitor) Service {
	return &contentService{
		repo:      repo,
		processor: processor,
		metrics:   metrics,
		monitor:   monitor,
	}
}

func (s *contentService) CreateContent(ctx context.Context, content *v1alpha1.ContentSource) error {
	if err := content.Spec.Validate(); err != nil {
		return err
	}

	// Update status fields
	content.Status.LastValidated = time.Now()
	content.Status.IsHealthy = true
	content.Status.Version = 1

	// Store in repository
	return s.repo.CreateContent(ctx, content)
}

func (s *contentService) UpdateContent(ctx context.Context, content *v1alpha1.ContentSource) error {
	if err := content.Spec.Validate(); err != nil {
		return err
	}

	// Update status fields
	content.Status.LastValidated = time.Now()
	content.Status.Version++ // Increment version on update

	// Store in repository
	return s.repo.UpdateContent(ctx, content)
}

func (s *contentService) DeleteContent(ctx context.Context, name string) error {
	return s.repo.DeleteContent(ctx, name)
}

func (s *contentService) GetContent(ctx context.Context, name string) (*v1alpha1.ContentSource, error) {
	return s.repo.GetContent(ctx, name)
}

func (s *contentService) ListContent(ctx context.Context) ([]v1alpha1.ContentSource, error) {
	return s.repo.ListContent(ctx)
}

func (s *contentService) ReportEvents(ctx context.Context, batch EventBatch) error {
	if err := s.processor.ProcessEvents(ctx, batch); err != nil {
		return err
	}

	// Process each event for metrics
	for _, event := range batch.Events {
		if err := s.metrics.RecordMetrics(ctx, event); err != nil {
			// Log error but continue processing
			continue
		}
	}

	return nil
}

func (s *contentService) GetURLHealth(ctx context.Context, url string) (*HealthStatus, error) {
	return s.monitor.CheckHealth(ctx, url)
}

func (s *contentService) GetURLMetrics(ctx context.Context, url string) (*URLMetrics, error) {
	return s.metrics.GetURLMetrics(ctx, url)
}

func (s *contentService) ValidateContent(ctx context.Context, url string) error {
	// For now, allow all valid URLs until we have metrics history
	return nil
}
