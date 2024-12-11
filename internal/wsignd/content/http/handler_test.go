package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/wrale/wrale-signage/api/types/v1alpha1"
	"github.com/wrale/wrale-signage/internal/wsignd/content"
)

type mockService struct {
	mock.Mock
}

func (m *mockService) CreateContent(ctx context.Context, content *v1alpha1.ContentSource) error {
	args := m.Called(ctx, content)
	return args.Error(0)
}

func (m *mockService) UpdateContent(ctx context.Context, content *v1alpha1.ContentSource) error {
	args := m.Called(ctx, content)
	return args.Error(0)
}

func (m *mockService) DeleteContent(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

func (m *mockService) GetContent(ctx context.Context, name string) (*v1alpha1.ContentSource, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*v1alpha1.ContentSource), args.Error(1)
}

func (m *mockService) ListContent(ctx context.Context) ([]v1alpha1.ContentSource, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]v1alpha1.ContentSource), args.Error(1)
}

func (m *mockService) ReportEvents(ctx context.Context, batch content.EventBatch) error {
	args := m.Called(ctx, batch)
	return args.Error(0)
}

func (m *mockService) GetURLHealth(ctx context.Context, url string) (*content.HealthStatus, error) {
	args := m.Called(ctx, url)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*content.HealthStatus), args.Error(1)
}

func (m *mockService) GetURLMetrics(ctx context.Context, url string) (*content.URLMetrics, error) {
	args := m.Called(ctx, url)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*content.URLMetrics), args.Error(1)
}

func (m *mockService) ValidateContent(ctx context.Context, url string) error {
	args := m.Called(ctx, url)
	return args.Error(0)
}

// matchEventBatch is a custom matcher for EventBatch that ignores monotonic clock values
func matchEventBatch(expected content.EventBatch) interface{} {
	return mock.MatchedBy(func(actual content.EventBatch) bool {
		if expected.DisplayID != actual.DisplayID {
			return false
		}
		if len(expected.Events) != len(actual.Events) {
			return false
		}
		for i, expectedEvent := range expected.Events {
			actualEvent := actual.Events[i]
			if expectedEvent.ID != actualEvent.ID ||
				expectedEvent.DisplayID != actualEvent.DisplayID ||
				expectedEvent.Type != actualEvent.Type ||
				expectedEvent.URL != actualEvent.URL {
				return false
			}
			// Compare timestamps ignoring monotonic clock
			if !expectedEvent.Timestamp.Equal(actualEvent.Timestamp) {
				return false
			}
		}
		return true
	})
}

func TestCreateContent(t *testing.T) {
	tests := []struct {
		name         string
		content      *v1alpha1.ContentSource
		validateErr  error
		createErr    error
		expectedCode int
	}{
		{
			name: "successful_create",
			content: &v1alpha1.ContentSource{
				ObjectMeta: v1alpha1.ObjectMeta{
					Name: "test-content",
				},
				Spec: v1alpha1.ContentSourceSpec{
					URL:  "https://example.com/content",
					Type: "static-page",
				},
			},
			validateErr:  nil,
			createErr:    nil,
			expectedCode: http.StatusCreated,
		},
		{
			name:         "invalid_request",
			content:      nil,
			expectedCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := new(mockService)
			if tt.content != nil {
				mockSvc.On("ValidateContent", mock.Anything, tt.content.Spec.URL).Return(tt.validateErr)
				if tt.validateErr == nil {
					mockSvc.On("CreateContent", mock.Anything, mock.MatchedBy(func(c *v1alpha1.ContentSource) bool {
						return c.ObjectMeta.Name == tt.content.ObjectMeta.Name &&
							c.Spec.URL == tt.content.Spec.URL
					})).Return(tt.createErr)
				}
			}

			handler := NewHandler(mockSvc, zerolog.Nop())

			var body []byte
			if tt.content != nil {
				body, _ = json.Marshal(tt.content)
			}

			req := httptest.NewRequest(http.MethodPost, "/content", bytes.NewReader(body))
			w := httptest.NewRecorder()

			handler.handleCreateContent(w, req)
			assert.Equal(t, tt.expectedCode, w.Code)
			mockSvc.AssertExpectations(t)
		})
	}
}

func TestReportEvents(t *testing.T) {
	tests := []struct {
		name         string
		batch        *content.EventBatch
		serviceError error
		expectedCode int
	}{
		{
			name: "successful_report",
			batch: &content.EventBatch{
				DisplayID: uuid.New(),
				Events: []content.Event{
					{
						ID:        uuid.New(),
						Type:      content.EventContentLoaded,
						URL:       "https://example.com/content",
						Timestamp: time.Now(),
					},
				},
			},
			serviceError: nil,
			expectedCode: http.StatusAccepted,
		},
		{
			name:         "invalid_request",
			batch:        nil,
			serviceError: nil,
			expectedCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := new(mockService)
			if tt.batch != nil {
				mockSvc.On("ReportEvents", mock.Anything, matchEventBatch(*tt.batch)).Return(tt.serviceError)
			}

			handler := NewHandler(mockSvc, zerolog.Nop())

			var body []byte
			if tt.batch != nil {
				body, _ = json.Marshal(tt.batch)
			}

			req := httptest.NewRequest("POST", "/events", bytes.NewReader(body))
			w := httptest.NewRecorder()

			handler.handleReportEvents(w, req)
			assert.Equal(t, tt.expectedCode, w.Code)
			mockSvc.AssertExpectations(t)
		})
	}
}

func TestGetURLHealth(t *testing.T) {
	testURL := "https://example.com/content"
	tests := []struct {
		name         string
		url          string
		mockHealth   *content.HealthStatus
		mockError    error
		expectedCode int
	}{
		{
			name: "healthy_url",
			url:  testURL,
			mockHealth: &content.HealthStatus{
				URL:       testURL,
				Healthy:   true,
				LastCheck: time.Now().Unix(),
			},
			mockError:    nil,
			expectedCode: http.StatusOK,
		},
		{
			name:         "missing_url",
			url:          "",
			mockHealth:   nil,
			mockError:    nil,
			expectedCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := new(mockService)
			if tt.mockHealth != nil {
				mockSvc.On("GetURLHealth", mock.Anything, tt.url).Return(tt.mockHealth, tt.mockError)
			}

			handler := NewHandler(mockSvc, zerolog.Nop())

			req := httptest.NewRequest("GET", "/health/"+tt.url, nil)
			w := httptest.NewRecorder()

			// Add URL parameter to context using chi
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("url", tt.url)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			handler.handleGetURLHealth(w, req)
			assert.Equal(t, tt.expectedCode, w.Code)
			mockSvc.AssertExpectations(t)

			if tt.mockHealth != nil && tt.expectedCode == http.StatusOK {
				var response content.HealthStatus
				err := json.NewDecoder(w.Body).Decode(&response)
				assert.NoError(t, err)
				assert.Equal(t, tt.mockHealth.URL, response.URL)
				assert.Equal(t, tt.mockHealth.Healthy, response.Healthy)
			}
		})
	}
}
