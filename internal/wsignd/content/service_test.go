package content

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/wrale/wrale-signage/api/types/v1alpha1"
)

type mockRepository struct {
	mock.Mock
}

func (m *mockRepository) CreateContent(ctx context.Context, content *v1alpha1.ContentSource) error {
	args := m.Called(ctx, content)
	return args.Error(0)
}

func (m *mockRepository) UpdateContent(ctx context.Context, content *v1alpha1.ContentSource) error {
	args := m.Called(ctx, content)
	return args.Error(0)
}

func (m *mockRepository) DeleteContent(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

func (m *mockRepository) GetContent(ctx context.Context, name string) (*v1alpha1.ContentSource, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*v1alpha1.ContentSource), args.Error(1)
}

func (m *mockRepository) ListContent(ctx context.Context) ([]v1alpha1.ContentSource, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]v1alpha1.ContentSource), args.Error(1)
}

func (m *mockRepository) SaveEvent(ctx context.Context, event Event) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *mockRepository) GetURLMetrics(ctx context.Context, url string, since time.Time) (*URLMetrics, error) {
	args := m.Called(ctx, url, since)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*URLMetrics), args.Error(1)
}

func (m *mockRepository) GetDisplayEvents(ctx context.Context, displayID uuid.UUID, since time.Time) ([]Event, error) {
	args := m.Called(ctx, displayID, since)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]Event), args.Error(1)
}

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
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*URLMetrics), args.Error(1)
}

type mockMonitor struct {
	mock.Mock
}

func (m *mockMonitor) CheckHealth(ctx context.Context, url string) (*HealthStatus, error) {
	args := m.Called(ctx, url)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*HealthStatus), args.Error(1)
}

func (m *mockMonitor) GetHealthHistory(ctx context.Context, url string) ([]HealthStatus, error) {
	args := m.Called(ctx, url)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]HealthStatus), args.Error(1)
}

func TestService_CreateContent(t *testing.T) {
	ctx := context.Background()
	content := &v1alpha1.ContentSource{
		ObjectMeta: v1alpha1.ObjectMeta{
			Name: "test-content",
		},
		Spec: v1alpha1.ContentSourceSpec{
			URL:  "https://example.com/content",
			Type: "static-page",
		},
	}

	repository := new(mockRepository)
	repository.On("CreateContent", ctx, mock.MatchedBy(func(c *v1alpha1.ContentSource) bool {
		return c.ObjectMeta.Name == content.ObjectMeta.Name &&
			c.Spec.URL == content.Spec.URL
	})).Return(nil)

	processor := new(mockProcessor)
	metrics := new(mockMetrics)
	monitor := new(mockMonitor)

	service := NewService(repository, processor, metrics, monitor)
	err := service.CreateContent(ctx, content)

	assert.NoError(t, err)
	repository.AssertExpectations(t)
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

	repository := new(mockRepository)
	processor := new(mockProcessor)
	processor.On("ProcessEvents", ctx, batch).Return(nil)
	metrics := new(mockMetrics)
	metrics.On("RecordMetrics", ctx, batch.Events[0]).Return(nil)
	monitor := new(mockMonitor)

	service := NewService(repository, processor, metrics, monitor)
	err := service.ReportEvents(ctx, batch)
	assert.NoError(t, err)

	processor.AssertExpectations(t)
	metrics.AssertExpectations(t)
}

func TestService_ValidateContent(t *testing.T) {
	ctx := context.Background()
	url := "https://example.com/content"
	repository := new(mockRepository)
	processor := new(mockProcessor)
	metrics := new(mockMetrics)
	monitor := new(mockMonitor)

	service := NewService(repository, processor, metrics, monitor)
	err := service.ValidateContent(ctx, url)
	assert.NoError(t, err)
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

	repository := new(mockRepository)
	processor := new(mockProcessor)
	metrics := new(mockMetrics)
	monitor := new(mockMonitor)
	monitor.On("CheckHealth", ctx, url).Return(status, nil)

	service := NewService(repository, processor, metrics, monitor)
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

	repository := new(mockRepository)
	processor := new(mockProcessor)
	metricsAggregator := new(mockMetrics)
	metricsAggregator.On("GetURLMetrics", ctx, url).Return(metrics, nil)
	monitor := new(mockMonitor)

	service := NewService(repository, processor, metricsAggregator, monitor)
	result, err := service.GetURLMetrics(ctx, url)

	assert.NoError(t, err)
	assert.Equal(t, metrics, result)
	metricsAggregator.AssertExpectations(t)
}
