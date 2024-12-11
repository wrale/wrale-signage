package content

import (
	"context"

	"github.com/google/uuid"
)

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
