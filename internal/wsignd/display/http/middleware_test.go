package http_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	testhttp "github.com/wrale/wrale-signage/internal/wsignd/display/http/testing"
	"github.com/wrale/wrale-signage/internal/wsignd/ratelimit"
)

func TestMiddleware(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "recovers from panic",
			test: func(t *testing.T) {
				th := testhttp.NewTestHandler()
				router := th.Handler.Router()

				defer func() {
					th.Service.AssertExpectations(t)
					th.RateLimit.AssertExpectations(t)
				}()

				// Add panic handler
				router.HandleFunc("/api/v1alpha1/displays/panic", func(w http.ResponseWriter, r *http.Request) {
					panic("test panic")
				})

				req := httptest.NewRequest(http.MethodGet, "/api/v1alpha1/displays/panic", nil)
				rec := httptest.NewRecorder()

				router.ServeHTTP(rec, req)

				assert.Equal(t, http.StatusInternalServerError, rec.Code)

				var resp struct {
					Error string `json:"error"`
				}
				err := json.NewDecoder(rec.Body).Decode(&resp)
				assert.NoError(t, err)
				assert.Equal(t, "internal error", resp.Error)
			},
		},
		{
			name: "adds request id to context",
			test: func(t *testing.T) {
				th := testhttp.NewTestHandler()
				router := th.Handler.Router()

				defer func() {
					th.Service.AssertExpectations(t)
					th.RateLimit.AssertExpectations(t)
				}()

				var capturedID string
				router.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
					capturedID = r.Header.Get("X-Request-ID")
					w.WriteHeader(http.StatusOK)
				})

				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				rec := httptest.NewRecorder()

				router.ServeHTTP(rec, req)

				// Request ID should be present and match between context and response header
				assert.NotEmpty(t, capturedID)
				assert.Equal(t, capturedID, rec.Header().Get("X-Request-ID"))
			},
		},
		{
			name: "enforces rate limits",
			test: func(t *testing.T) {
				th := testhttp.NewTestHandler()

				defer func() {
					th.Service.AssertExpectations(t)
					th.RateLimit.AssertExpectations(t)
				}()

				// Configure rate limit mock to deny request
				th.RateLimit.On("GetLimit", "display").Return(ratelimit.Limit{
					Rate:      1,
					Period:    time.Minute,
					BurstSize: 1,
				})
				th.RateLimit.On("Allow", mock.Anything, mock.Anything).Return(errors.New("rate limit exceeded"))

				req := httptest.NewRequest(http.MethodGet, "/api/v1alpha1/displays/123", nil)
				rec := httptest.NewRecorder()

				// The request should be rate limited before hitting auth
				th.Handler.Router().ServeHTTP(rec, req)

				assert.Equal(t, http.StatusTooManyRequests, rec.Code)
			},
		},
		{
			name: "handles context cancellation",
			test: func(t *testing.T) {
				th := testhttp.NewTestHandler()
				router := th.Handler.Router()

				defer func() {
					th.Service.AssertExpectations(t)
					th.RateLimit.AssertExpectations(t)
				}()

				// Add slow handler that will be cancelled
				router.HandleFunc("/api/v1alpha1/displays/slow", func(w http.ResponseWriter, r *http.Request) {
					<-r.Context().Done()
					w.WriteHeader(http.StatusServiceUnavailable)
				})

				req := httptest.NewRequest(http.MethodGet, "/api/v1alpha1/displays/slow", nil)
				ctx, cancel := context.WithCancel(req.Context())
				req = req.WithContext(ctx)
				rec := httptest.NewRecorder()

				// Cancel immediately
				go func() {
					cancel()
				}()

				router.ServeHTTP(rec, req)
				assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}
