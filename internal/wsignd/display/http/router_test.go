package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRouter(t *testing.T) {
	handler, _ := newTestHandler()
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
			wantStatus: http.StatusBadRequest, // Invalid UUID
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
	handler, _ := newTestHandler()
	router := handler.Router()

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

				assert.NotEmpty(t, rec.Header().Get("X-Request-ID"))
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

				// Should still return BadRequest (invalid UUID) rather than timeout
				// since the handler is fast enough
				assert.Equal(t, http.StatusBadRequest, rec.Code)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}
