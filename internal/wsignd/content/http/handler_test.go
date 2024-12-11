package http

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/wrale/wrale-signage/internal/wsignd/content"
)

type mockService struct {
	mock.Mock
}

func (m *mockService) ReportEvents(ctx context.Context, batch content.EventBatch) error {
	args := m.Called(ctx, batch)
	return args.Error(0)
}

func (m *mockService) GetURLHealth(ctx context.Context, url string) (*content.HealthStatus, error) {
	args := m.Called(ctx, url)
	return args.Get(0).(*content.HealthStatus), args.Error(1)
}

func (m *mockService) GetURLMetrics(ctx context.Context, url string) (*content.URLMetrics, error) {
	args := m.Called(ctx, url)
	return args.Get(0).(*content.URLMetrics), args.Error(1)
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
				mockSvc.On("ReportEvents", mock.Anything, *tt.batch).Return(tt.serviceError)
			}

			handler := NewHandler(mockSvc, slog.Default())

			var body []byte
			if tt.batch != nil {
				body, _ = json.Marshal(tt.batch)
			}

			req := httptest.NewRequest("POST", "/events", bytes.NewReader(body))
			w := httptest.NewRecorder()

			handler.ReportEvents(w, req)
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

			handler := NewHandler(mockSvc, slog.Default())

			req := httptest.NewRequest("GET", "/health/"+tt.url, nil)
			w := httptest.NewRecorder()

			// Add URL parameter to context using chi
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("url", tt.url)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			handler.GetURLHealth(w, req)
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
