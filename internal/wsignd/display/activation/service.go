package activation

import (
	"context"

	"github.com/google/uuid"
)

// Service manages device activation flow
type Service interface {
	// GenerateCode creates a new device code pair
	GenerateCode(ctx context.Context) (*DeviceCode, error)

	// ActivateCode activates a device using its code
	ActivateCode(ctx context.Context, code string) (uuid.UUID, error)

	// ValidateCode checks if a device code is valid and unused
	ValidateCode(ctx context.Context, code string) (*DeviceCode, error)
}

// DefaultService implements the Service interface
type DefaultService struct {
	repo Repository
}

// NewService creates a new activation service
func NewService(repo Repository) Service {
	return &DefaultService{repo: repo}
}

func (s *DefaultService) GenerateCode(ctx context.Context) (*DeviceCode, error) {
	code, err := NewDeviceCode()
	if err != nil {
		return nil, err
	}

	if err := s.repo.Save(code); err != nil {
		return nil, err
	}

	return code, nil
}

func (s *DefaultService) ActivateCode(ctx context.Context, code string) (uuid.UUID, error) {
	deviceCode, err := s.repo.FindByUserCode(code)
	if err != nil {
		return uuid.Nil, err
	}

	displayID := uuid.New()
	if err := deviceCode.Activate(displayID); err != nil {
		return uuid.Nil, err
	}

	if err := s.repo.Save(deviceCode); err != nil {
		return uuid.Nil, err
	}

	return displayID, nil
}

func (s *DefaultService) ValidateCode(ctx context.Context, code string) (*DeviceCode, error) {
	deviceCode, err := s.repo.FindByDeviceCode(code)
	if err != nil {
		return nil, err
	}

	if deviceCode.IsExpired() {
		return nil, ErrCodeExpired
	}

	return deviceCode, nil
}
