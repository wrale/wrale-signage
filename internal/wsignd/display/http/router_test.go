package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/wrale/wrale-signage/internal/wsignd/display"
)

func TestRouter(t *testing.T) {
	handler, mockSvc := newTestHandler()

	// Setup default mock behavior for name lookup
	mockSvc.On("GetByName", mock.Anything, mock.AnythingOfType("string")).Return(nil, display.ErrNotFound{ID: "unknown"})

	router := handler.Router()

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
	}{
		{
			name:       "display registration endpoint exists",
			method:     http.MethodPost,
			path:       "/api/v1alpha1/displays",
			wantStatus: http.StatusBadRequest, // Invalid JSON body
		},
		{
			name:       "display get endpoint exists",
			method:     http.MethodGet,
			path:       "/api/v1alpha1/displays/123",
			wantStatus: http.StatusNotFound, // Not found after name lookup
		},
		{
			name:       "display activate endpoint exists",
			method:     http.MethodPut,
			path:       "/api/v1alpha1/displays/123/activate",
			wantStatus: http.StatusBadRequest, // Invalid UUID
		},
		{
			name:       "display last seen endpoint exists",
			method:     http.MethodPut,
			path:       "/api/v1alpha1/displays/123/last-seen",
			wantStatus: http.StatusBadRequest, // Invalid UUID
		},
		{
			name:       "non-existent endpoint returns 404",
			method:     http.MethodGet,
			path:       "/api/v1alpha1/non-existent",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

func TestRouterMiddleware(t *testing.T) {
	handler, mockSvc := newTestHandler()
	router := handler.Router()

	// Setup default mock behavior for name lookup
	mockSvc.On("GetByName", mock.Anything, mock.AnythingOfType("string")).Return(nil, display.ErrNotFound{ID: "unknown"})

	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "adds request id header",
			test: func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, "/api/v1alpha1/displays/123", nil)
				rec := httptest.NewRecorder()

				router.ServeHTTP(rec, req)

				assert.NotEmpty(t, rec.Header().Get("Request-ID"))
			},
		},
		{
			name: "recovers from panic",
			test: func(t *testing.T) {
				// Create a test handler that panics
				router.HandleFunc("/panic", func(w http.ResponseWriter, r *http.Request) {
					panic("test panic")
				})

				req := httptest.NewRequest(http.MethodGet, "/panic", nil)
				rec := httptest.NewRecorder()

				router.ServeHTTP(rec, req)

				assert.Equal(t, http.StatusInternalServerError, rec.Code)
			},
		},
		{
			name: "handles context cancellation",
			test: func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				req := httptest.NewRequest(http.MethodGet, "/api/v1alpha1/displays/123", nil).WithContext(ctx)
				rec := httptest.NewRecorder()

				// Cancel before request
				cancel()

				router.ServeHTTP(rec, req)

				// Should return NotFound since the service returns not found
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}
