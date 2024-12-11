package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/wrale/wrale-signage/api/types/v1alpha1"
	"github.com/wrale/wrale-signage/internal/wsignd/display"
)

func TestRegisterDisplay(t *testing.T) {
	handler, mockSvc := newTestHandler()

	tests := []struct {
		name       string
		input      v1alpha1.DisplayRegistrationRequest
		mockSetup  func()
		wantStatus int
		wantErr    bool
	}{
		{
			name: "successful registration",
			input: v1alpha1.DisplayRegistrationRequest{
				Name: "test-display",
				Location: v1alpha1.DisplayLocation{
					SiteID:   "site-1",
					Zone:     "lobby",
					Position: "main",
				},
			},
			mockSetup: func() {
				mockSvc.On("Register",
					mock.Anything,
					"test-display",
					display.Location{
						SiteID:   "site-1",
						Zone:     "lobby",
						Position: "main",
					},
				).Return(&display.Display{
					ID:   uuid.New(),
					Name: "test-display",
					Location: display.Location{
						SiteID:   "site-1",
						Zone:     "lobby",
						Position: "main",
					},
					State:    display.StateUnregistered,
					LastSeen: time.Now(),
					Version:  1,
				}, nil)
			},
			wantStatus: http.StatusOK,
			wantErr:    false,
		},
		{
			name: "invalid request body",
			input: v1alpha1.DisplayRegistrationRequest{
				Name: "", // Empty name should fail
			},
			mockSetup: func() {
				// No mock setup needed - should fail validation before service call
			},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name: "display already exists",
			input: v1alpha1.DisplayRegistrationRequest{
				Name: "existing-display",
				Location: v1alpha1.DisplayLocation{
					SiteID: "site-1",
					Zone:   "lobby",
				},
			},
			mockSetup: func() {
				mockSvc.On("Register",
					mock.Anything,
					"existing-display",
					mock.Anything,
				).Return(nil, display.ErrExists{Name: "existing-display"})
			},
			wantStatus: http.StatusConflict,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock
			mockSvc.Mock = mock.Mock{}

			// Setup mock expectations
			if tt.mockSetup != nil {
				tt.mockSetup()
			}

			// Create request
			body, err := json.Marshal(tt.input)
			assert.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/api/v1alpha1/displays", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			rec := httptest.NewRecorder()

			// Add chi context
			handler := mockChiContext(http.HandlerFunc(handler.RegisterDisplay))
			handler.ServeHTTP(rec, req)

			// Check status code
			assert.Equal(t, tt.wantStatus, rec.Code)

			// Verify mock
			mockSvc.AssertExpectations(t)
		})
	}
}
