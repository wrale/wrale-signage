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

	// Validate URL is accessible
	if err := s.ValidateContent(ctx, content.Spec.URL); err != nil {
		return err
	}

	// Update status fields
	content.Status.LastValidated = time.Now()
	content.Status.IsHealthy = true
	content.Status.Version = 1

	// Store in repository
	return s.repo.CreateContent(ctx, content)
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
	// Initial implementation just checks if we have recent successful loads
	metrics, err := s.metrics.GetURLMetrics(ctx, url)
	if err != nil {
		return err
	}

	// Content considered valid if seen in last hour with < 10% error rate
	if time.Now().Unix()-metrics.LastSeen > 3600 {
		return ErrContentStale
	}

	if metrics.LoadCount > 0 {
		errorRate := float64(metrics.ErrorCount) / float64(metrics.LoadCount)
		if errorRate > 0.1 {
			return ErrContentUnreliable
		}
	}

	return nil
}
