package content

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockProcessor struct {
	mock.Mock
}

func (m *mockProcessor) ProcessEvents(ctx context.Context, batch EventBatch) error {
	args := m.Called(ctx, batch)
	return args.Error(0)
}

type mockMetrics struct {
	mock.Mock
}

func (m *mockMetrics) RecordMetrics(ctx context.Context, event Event) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *mockMetrics) GetURLMetrics(ctx context.Context, url string) (*URLMetrics, error) {
	args := m.Called(ctx, url)
	return args.Get(0).(*URLMetrics), args.Error(1)
}

type mockMonitor struct {
	mock.Mock
}

func (m *mockMonitor) CheckHealth(ctx context.Context, url string) (*HealthStatus, error) {
	args := m.Called(ctx, url)
	return args.Get(0).(*HealthStatus), args.Error(1)
}

func (m *mockMonitor) GetHealthHistory(ctx context.Context, url string) ([]HealthStatus, error) {
	args := m.Called(ctx, url)
	return args.Get(0).([]HealthStatus), args.Error(1)
}

func TestService_ReportEvents(t *testing.T) {
	ctx := context.Background()
	batch := EventBatch{
		DisplayID: uuid.New(),
		Events: []Event{
			{
				ID:        uuid.New(),
				Type:      EventContentLoaded,
				URL:       "https://example.com/content",
				Timestamp: time.Now(),
			},
		},
	}

	processor := new(mockProcessor)
	processor.On("ProcessEvents", ctx, batch).Return(nil)

	metrics := new(mockMetrics)
	metrics.On("RecordMetrics", ctx, batch.Events[0]).Return(nil)

	monitor := new(mockMonitor)

	service := NewService(processor, metrics, monitor)
	err := service.ReportEvents(ctx, batch)
	assert.NoError(t, err)

	processor.AssertExpectations(t)
	metrics.AssertExpectations(t)
}

func TestService_ValidateContent(t *testing.T) {
	ctx := context.Background()
	url := "https://example.com/content"

	tests := []struct {
		name      string
		metrics   *URLMetrics
		wantError error
	}{
		{
			name: "valid_content",
			metrics: &URLMetrics{
				URL:        url,
				LastSeen:   time.Now().Unix(),
				LoadCount:  100,
				ErrorCount: 5,
			},
			wantError: nil,
		},
		{
			name: "stale_content",
			metrics: &URLMetrics{
				URL:      url,
				LastSeen: time.Now().Add(-2 * time.Hour).Unix(),
			},
			wantError: ErrContentStale,
		},
		{
			name: "unreliable_content",
			metrics: &URLMetrics{
				URL:        url,
				LastSeen:   time.Now().Unix(),
				LoadCount:  100,
				ErrorCount: 15, // 15% error rate
			},
			wantError: ErrContentUnreliable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor := new(mockProcessor)
			metrics := new(mockMetrics)
			metrics.On("GetURLMetrics", ctx, url).Return(tt.metrics, nil)
			monitor := new(mockMonitor)

			service := NewService(processor, metrics, monitor)
			err := service.ValidateContent(ctx, url)

			if tt.wantError != nil {
				assert.ErrorIs(t, err, tt.wantError)
			} else {
				assert.NoError(t, err)
			}
			metrics.AssertExpectations(t)
		})
	}
}

func TestService_GetURLHealth(t *testing.T) {
	ctx := context.Background()
	url := "https://example.com/content"
	status := &HealthStatus{
		URL:       url,
		Healthy:   true,
		LastCheck: time.Now().Unix(),
		Displays:  []uuid.UUID{uuid.New()},
	}

	processor := new(mockProcessor)
	metrics := new(mockMetrics)
	monitor := new(mockMonitor)
	monitor.On("CheckHealth", ctx, url).Return(status, nil)

	service := NewService(processor, metrics, monitor)
	result, err := service.GetURLHealth(ctx, url)

	assert.NoError(t, err)
	assert.Equal(t, status, result)
	monitor.AssertExpectations(t)
}

func TestService_GetURLMetrics(t *testing.T) {
	ctx := context.Background()
	url := "https://example.com/content"
	metrics := &URLMetrics{
		URL:           url,
		LastSeen:      time.Now().Unix(),
		LoadCount:     100,
		ErrorCount:    5,
		AvgLoadTime:   500,
		AvgRenderTime: 200,
		ErrorRates: map[string]float64{
			"LOAD_FAILED": 0.05,
		},
	}

	processor := new(mockProcessor)
	metricsAggregator := new(mockMetrics)
	metricsAggregator.On("GetURLMetrics", ctx, url).Return(metrics, nil)
	monitor := new(mockMonitor)

	service := NewService(processor, metricsAggregator, monitor)
	result, err := service.GetURLMetrics(ctx, url)

	assert.NoError(t, err)
	assert.Equal(t, metrics, result)
	metricsAggregator.AssertExpectations(t)
}
