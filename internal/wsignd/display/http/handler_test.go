package http

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
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

// newTestHandler creates a new handler with mock service for testing
func newTestHandler() (*Handler, *mockService) {
	mockSvc := &mockService{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	return NewHandler(mockSvc, logger), mockSvc
}

// mockChiContext adds a chi routing context to the request
func mockChiContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chiCtx := chi.NewRouteContext()
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chiCtx))
		next.ServeHTTP(w, r)
	})
}
