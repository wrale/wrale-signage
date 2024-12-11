package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"log/slog"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/assert"
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

	t.Run("adds request id header", func(t *testing.T) {
		router := chi.NewRouter()
		router.Use(middleware.RequestID)

		// Add handler that checks request ID in context
		router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			requestID := middleware.GetReqID(r.Context())
			assert.NotEmpty(t, requestID)
			w.Header().Set(middleware.RequestIDHeader, requestID)
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.NotEmpty(t, rec.Header().Get(middleware.RequestIDHeader))
	})

	t.Run("recovers from panic", func(t *testing.T) {
		router := chi.NewRouter()
		router.Use(middleware.Recoverer)
		router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			panic("test panic")
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		router := NewRouter(handler)

		ctx, cancel := context.WithCancel(context.Background())

		req := httptest.NewRequest(http.MethodGet, "/api/v1alpha1/displays/123", nil)
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		// Cancel the context before request
		cancel()

		router.ServeHTTP(rec, req)

		// Should still get bad request due to invalid UUID, context cancellation
		// is handled gracefully
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}
