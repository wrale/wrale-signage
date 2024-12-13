package http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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

				th.SetupRateLimitBypass()

				// Add panic handler
				router.HandleFunc("/test/panic", func(w http.ResponseWriter, r *http.Request) {
					panic("test panic")
				})

				req := httptest.NewRequest(http.MethodGet, "/test/panic", nil)
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
				defer func() {
					th.RateLimit.AssertExpectations(t)
				}()

				th.SetupRateLimitBypass()

				// Add test handler
				router := th.Handler.Router()
				router.Get("/test/id", func(w http.ResponseWriter, r *http.Request) {
					// Request ID should be propagated and added to response headers
					w.WriteHeader(http.StatusOK)
				})

				req := httptest.NewRequest(http.MethodGet, "/test/id", nil)
				rec := httptest.NewRecorder()

				router.ServeHTTP(rec, req)

				assert.Equal(t, http.StatusOK, rec.Code)
				assert.NotEmpty(t, rec.Header().Get("X-Request-ID"))
				assert.Equal(t, rec.Header().Get("Request-ID"), rec.Header().Get("X-Request-ID"))
			},
		},
		{
			name: "enforces rate limits",
			test: func(t *testing.T) {
				th := testhttp.NewTestHandler()
				defer func() {
					th.RateLimit.AssertExpectations(t)
				}()

				// Configure rate limit mock to deny request
				th.RateLimit.On("GetLimit", "api").Return(ratelimit.Limit{
					Rate:      1,
					Period:    time.Minute,
					BurstSize: 1,
				})
				th.RateLimit.On("Allow", "api", "test-request-id").Return(ratelimit.ErrLimitExceeded)

				req := httptest.NewRequest(http.MethodGet, "/api/v1alpha1/displays/123", nil)
				rec := httptest.NewRecorder()

				th.Handler.Router().ServeHTTP(rec, req)

				assert.Equal(t, http.StatusTooManyRequests, rec.Code)
			},
		},
		{
			name: "handles context cancellation",
			test: func(t *testing.T) {
				th := testhttp.NewTestHandler()
				router := th.Handler.Router()

				th.SetupRateLimitBypass()

				// Add handler that blocks until context is cancelled
				router.HandleFunc("/test/slow", func(w http.ResponseWriter, r *http.Request) {
					<-r.Context().Done()
					w.WriteHeader(http.StatusServiceUnavailable)
				})

				// Create cancellable request
				ctx, cancel := context.WithCancel(context.Background())
				req := httptest.NewRequest(http.MethodGet, "/test/slow", nil).WithContext(ctx)
				rec := httptest.NewRecorder()

				// Cancel context after a short delay
				go func() {
					time.Sleep(100 * time.Millisecond)
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
