package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"log/slog"
	"os"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRouter(t *testing.T) {
	mockSvc := &mockService{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := NewHandler(mockSvc, logger)
	router := NewRouter(handler)

	tests := []struct {
		name           string
		method         string
		path           string
		wantStatusCode int
	}{
		{
			name:           "display registration endpoint exists",
			method:         http.MethodPost,
			path:           "/api/v1alpha1/displays",
			wantStatusCode: http.StatusBadRequest, // Expect bad request due to no body
		},
		{
			name:           "display get endpoint exists",
			method:         http.MethodGet,
			path:           "/api/v1alpha1/displays/123",
			wantStatusCode: http.StatusBadRequest, // Expect bad request due to invalid UUID
		},
		{
			name:           "display activate endpoint exists",
			method:         http.MethodPut,
			path:           "/api/v1alpha1/displays/123/activate",
			wantStatusCode: http.StatusBadRequest, // Expect bad request due to invalid UUID
		},
		{
			name:           "display last seen endpoint exists",
			method:         http.MethodPut,
			path:           "/api/v1alpha1/displays/123/last-seen",
			wantStatusCode: http.StatusBadRequest, // Expect bad request due to invalid UUID
		},
		{
			name:           "non-existent endpoint returns 404",
			method:         http.MethodGet,
			path:           "/api/v1alpha1/non-existent",
			wantStatusCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatusCode, rec.Code)
		})
	}
}

func TestRouterMiddleware(t *testing.T) {
	mockSvc := &mockService{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := NewHandler(mockSvc, logger)
	router := NewRouter(handler)

	t.Run("adds request id header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1alpha1/displays/123", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		requestID := rec.Header().Get("X-Request-Id")
		assert.NotEmpty(t, requestID)
	})

	t.Run("recovers from panic", func(t *testing.T) {
		mockSvc := &mockService{}
		mockSvc.On("Get", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			panic("test panic")
		}).Return(nil, nil)

		req := httptest.NewRequest(http.MethodGet, "/api/v1alpha1/displays/123", nil)
		rec := httptest.NewRecorder()

		// This should not panic
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		req := httptest.NewRequest(http.MethodGet, "/api/v1alpha1/displays/123", nil)
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		// Cancel the context before request
		cancel()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}
