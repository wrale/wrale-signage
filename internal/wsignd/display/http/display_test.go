package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/wrale/wrale-signage/internal/wsignd/display"
)

func TestGetDisplay(t *testing.T) {
	handler, mockSvc := newTestHandler()

	displayID := uuid.New()
	existingDisplay := &display.Display{
		ID:   displayID,
		Name: "test-display",
		Location: display.Location{
			SiteID:   "site-1",
			Zone:     "lobby",
			Position: "main",
		},
		State:    display.StateActive,
		LastSeen: time.Now(),
		Version:  1,
	}

	tests := []struct {
		name       string
		displayID  string
		mockSetup  func()
		wantStatus int
	}{
		{
			name:      "successful get",
			displayID: displayID.String(),
			mockSetup: func() {
				mockSvc.On("Get", mock.Anything, displayID).Return(existingDisplay, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:      "lookup by name",
			displayID: "not-a-uuid",
			mockSetup: func() {
				mockSvc.On("GetByName", mock.Anything, "not-a-uuid").Return(existingDisplay, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:      "display not found by name",
			displayID: "unknown-display",
			mockSetup: func() {
				mockSvc.On("GetByName", mock.Anything, "unknown-display").Return(nil, display.ErrNotFound{ID: "unknown-display"})
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:      "display not found by id",
			displayID: uuid.New().String(),
			mockSetup: func() {
				mockSvc.On("Get", mock.Anything, mock.Anything).Return(nil, display.ErrNotFound{ID: "unknown"})
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock
			mockSvc.Mock = mock.Mock{}

			// Setup mock expectations
			tt.mockSetup()

			// Create request with chi context
			req := httptest.NewRequest(http.MethodGet, "/api/v1alpha1/displays/"+tt.displayID, nil)
			chiCtx := chi.NewRouteContext()
			chiCtx.URLParams.Add("id", tt.displayID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

			// Create response recorder
			rec := httptest.NewRecorder()

			// Handle request
			handler.GetDisplay(rec, req)

			// Check status code
			assert.Equal(t, tt.wantStatus, rec.Code)

			// Verify mock
			mockSvc.AssertExpectations(t)
		})
	}
}

func TestActivateDisplay(t *testing.T) {
	handler, mockSvc := newTestHandler()

	displayID := uuid.New()
	existingDisplay := &display.Display{
		ID:   displayID,
		Name: "test-display",
		Location: display.Location{
			SiteID:   "site-1",
			Zone:     "lobby",
			Position: "main",
		},
		State:    display.StateActive,
		LastSeen: time.Now(),
		Version:  1,
	}

	tests := []struct {
		name       string
		displayID  string
		mockSetup  func()
		wantStatus int
	}{
		{
			name:      "successful activation",
			displayID: displayID.String(),
			mockSetup: func() {
				mockSvc.On("Activate", mock.Anything, displayID).Return(existingDisplay, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:      "invalid uuid",
			displayID: "not-a-uuid",
			mockSetup: func() {
				// No mock setup needed - should fail before service call
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:      "invalid state transition",
			displayID: displayID.String(),
			mockSetup: func() {
				mockSvc.On("Activate", mock.Anything, displayID).Return(nil, display.ErrInvalidState{
					Current: display.StateDisabled,
					Target:  display.StateActive,
				})
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:      "display not found",
			displayID: displayID.String(),
			mockSetup: func() {
				mockSvc.On("Activate", mock.Anything, displayID).Return(nil, display.ErrNotFound{ID: displayID.String()})
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock
			mockSvc.Mock = mock.Mock{}

			// Setup mock expectations
			tt.mockSetup()

			// Create request with chi context
			req := httptest.NewRequest(http.MethodPut, "/api/v1alpha1/displays/"+tt.displayID+"/activate", nil)
			chiCtx := chi.NewRouteContext()
			chiCtx.URLParams.Add("id", tt.displayID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

			// Create response recorder
			rec := httptest.NewRecorder()

			// Handle request
			handler.ActivateDisplay(rec, req)

			// Check status code
			assert.Equal(t, tt.wantStatus, rec.Code)

			// Verify mock
			mockSvc.AssertExpectations(t)
		})
	}
}
