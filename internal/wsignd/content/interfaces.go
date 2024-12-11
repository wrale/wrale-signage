package content

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/wrale/wrale-signage/api/types/v1alpha1"
)

// Service defines the content service interface
type Service interface {
	CreateContent(ctx context.Context, content *v1alpha1.ContentSource) error
	ReportEvents(ctx context.Context, batch EventBatch) error
	GetURLHealth(ctx context.Context, url string) (*HealthStatus, error)
	GetURLMetrics(ctx context.Context, url string) (*URLMetrics, error)
	ValidateContent(ctx context.Context, url string) error
}

type Repository interface {
	CreateContent(ctx context.Context, content *v1alpha1.ContentSource) error
	SaveEvent(ctx context.Context, event Event) error
	GetURLMetrics(ctx context.Context, url string, since time.Time) (*URLMetrics, error)
	GetDisplayEvents(ctx context.Context, displayID uuid.UUID, since time.Time) ([]Event, error)
}

type EventProcessor interface {
	ProcessEvents(ctx context.Context, batch EventBatch) error
}

type MetricsAggregator interface {
	RecordMetrics(ctx context.Context, event Event) error
	GetURLMetrics(ctx context.Context, url string) (*URLMetrics, error)
}

type HealthMonitor interface {
	CheckHealth(ctx context.Context, url string) (*HealthStatus, error)
	GetHealthHistory(ctx context.Context, url string) ([]HealthStatus, error)
}

type URLMetrics struct {
	URL           string
	LastSeen      int64
	LoadCount     int64
	ErrorCount    int64
	AvgLoadTime   float64
	AvgRenderTime float64
	ErrorRates    map[string]float64
}

type HealthStatus struct {
	URL       string
	Healthy   bool
	Issues    []string
	LastCheck int64
	Displays  []uuid.UUID
}
