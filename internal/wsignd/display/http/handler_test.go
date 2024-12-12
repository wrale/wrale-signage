package http

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/wrale/wrale-signage/internal/wsignd/auth"
	"github.com/wrale/wrale-signage/internal/wsignd/display"
	"github.com/wrale/wrale-signage/internal/wsignd/display/activation"
	"github.com/wrale/wrale-signage/internal/wsignd/ratelimit"
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

func (m *mockService) GetByName(ctx context.Context, name string) (*display.Display, error) {
	args := m.Called(ctx, name)
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

func (m *mockService) Activate(ctx context.Context, id uuid.UUID) (*display.Display, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*display.Display), args.Error(1)
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

type mockActivationService struct {
	mock.Mock
}

func (m *mockActivationService) GenerateCode(ctx context.Context) (*activation.DeviceCode, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*activation.DeviceCode), args.Error(1)
}

func (m *mockActivationService) ActivateCode(ctx context.Context, code string) (uuid.UUID, error) {
	args := m.Called(ctx, code)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *mockActivationService) ValidateCode(ctx context.Context, code string) (*activation.DeviceCode, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*activation.DeviceCode), args.Error(1)
}

type mockAuthService struct {
	mock.Mock
}

func (m *mockAuthService) CreateToken(ctx context.Context, displayID uuid.UUID) (*auth.Token, error) {
	args := m.Called(ctx, displayID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*auth.Token), args.Error(1)
}

func (m *mockAuthService) ValidateAccessToken(ctx context.Context, token string) (uuid.UUID, error) {
	args := m.Called(ctx, token)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *mockAuthService) RefreshToken(ctx context.Context, refreshToken string) (*auth.Token, error) {
	args := m.Called(ctx, refreshToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*auth.Token), args.Error(1)
}

func (m *mockAuthService) RevokeTokens(ctx context.Context, displayID uuid.UUID) error {
	args := m.Called(ctx, displayID)
	return args.Error(0)
}

type mockRateLimitService struct {
	mock.Mock
}

func (m *mockRateLimitService) Allow(ctx context.Context, key ratelimit.LimitKey) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *mockRateLimitService) GetLimit(limitType string) ratelimit.Limit {
	args := m.Called(limitType)
	return args.Get(0).(ratelimit.Limit)
}

func (m *mockRateLimitService) Reset(ctx context.Context, key ratelimit.LimitKey) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

// newTestHandler creates a new handler with mock services for testing
func newTestHandler() (*Handler, *mockService) {
	mockSvc := &mockService{}
	mockActSvc := &mockActivationService{}
	mockAuthSvc := &mockAuthService{}
	mockRateLimitSvc := &mockRateLimitService{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	return NewHandler(mockSvc, mockActSvc, mockAuthSvc, mockRateLimitSvc, logger), mockSvc
}

// mockChiContext adds a chi routing context to the request
func mockChiContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chiCtx := chi.NewRouteContext()
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chiCtx))
		next.ServeHTTP(w, r)
	})
}
