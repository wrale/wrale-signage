package http

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/wrale/wrale-signage/api/types/v1alpha1"
	"github.com/wrale/wrale-signage/internal/wsignd/display"
)

type mockService struct {
	mock.Mock
}

func (m *mockService) Register(ctx context.Context, name string, location display.Location) (*display.Display, error) {
	args := m.Called(ctx, name, location)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*display.Display), args.Error(1)
}

func (m *mockService) Get(ctx context.Context, id uuid.UUID) (*display.Display, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*display.Display), args.Error(1)
}

func (m *mockService) List(ctx context.Context, filter display.DisplayFilter) ([]*display.Display, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]*display.Display), args.Error(1)
}

func (m *mockService) UpdateLocation(ctx context.Context, id uuid.UUID, location display.Location) error {
	args := m.Called(ctx, id, location)
	return args.Error(0)
}

func (m *mockService) Activate(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockService) Disable(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockService) UpdateLastSeen(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockService) SetProperty(ctx context.Context, id uuid.UUID, key, value string) error {
	args := m.Called(ctx, id, key, value)
	return args.Error(0)
}

// Helper function to mock chi routing context
func mockChiContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chiCtx := chi.NewRouteContext()
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chiCtx))
		next.ServeHTTP(w, r)
	})
}

func TestRegisterDisplay(t *testing.T) {
	mockSvc := &mockService{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := NewHandler(mockSvc, logger)

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
				mockSvc.On("Register",
					mock.Anything,
					"",
					mock.Anything,
				).Return(nil, display.ErrInvalidName{Name: "", Reason: "name cannot be empty"})
			},
			wantStatus: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock
			mockSvc.Mock = mock.Mock{}

			// Setup mock expectations
			tt.mockSetup()

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

func TestGetDisplay(t *testing.T) {
	mockSvc := &mockService{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := NewHandler(mockSvc, logger)

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
			name:      "invalid uuid",
			displayID: "not-a-uuid",
			mockSetup: func() {
				// No mock setup needed - should fail before service call
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:      "display not found",
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
	mockSvc := &mockService{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := NewHandler(mockSvc, logger)

	displayID := uuid.New()

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
				mockSvc.On("Activate", mock.Anything, displayID).Return(nil)
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
			name:      "activation error",
			displayID: displayID.String(),
			mockSetup: func() {
				mockSvc.On("Activate", mock.Anything, displayID).Return(display.ErrInvalidState{
					Current: display.StateDisabled,
					Target:  display.StateActive,
				})
			},
			wantStatus: http.StatusInternalServerError,
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
