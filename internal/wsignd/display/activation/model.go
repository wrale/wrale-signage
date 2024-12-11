package activation

import (
	"crypto/rand"
	"encoding/base32"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Common errors
var (
	ErrCodeExpired    = errors.New("activation code expired")
	ErrCodeNotFound   = errors.New("activation code not found")
	ErrCodeInvalid    = errors.New("invalid activation code")
	ErrAlreadyActive  = errors.New("device already activated")
	ErrInvalidRequest = errors.New("invalid request parameters")
)

// DeviceCode represents a pending device activation request
type DeviceCode struct {
	ID           uuid.UUID
	DeviceCode   string    // Opaque code for verification
	UserCode     string    // Human-readable code shown on display
	ExpiresAt    time.Time // When this code expires
	PollInterval int       // How often device should poll (seconds)
	Activated    bool      // Whether code has been activated
	ActivatedAt  *time.Time
	DisplayID    *uuid.UUID // Set when activated
}

// NewDeviceCode creates a new activation code pair
func NewDeviceCode() (*DeviceCode, error) {
	// Generate 30 random bytes for the device code
	devBytes := make([]byte, 30)
	if _, err := rand.Read(devBytes); err != nil {
		return nil, err
	}
	deviceCode := base32.StdEncoding.EncodeToString(devBytes)

	// Generate 8-character user code (e.g., "WDJC-XYZK")
	userBytes := make([]byte, 4)
	if _, err := rand.Read(userBytes); err != nil {
		return nil, err
	}
	userCode := strings.ToUpper(base32.StdEncoding.EncodeToString(userBytes)[:8])
	userCode = userCode[:4] + "-" + userCode[4:] // Insert hyphen

	return &DeviceCode{
		ID:           uuid.New(),
		DeviceCode:   deviceCode,
		UserCode:     userCode,
		ExpiresAt:    time.Now().Add(15 * time.Minute),
		PollInterval: 5,
	}, nil
}

// IsExpired checks if the code has expired
func (d *DeviceCode) IsExpired() bool {
	return time.Now().After(d.ExpiresAt)
}

// Activate marks the device code as activated
func (d *DeviceCode) Activate(displayID uuid.UUID) error {
	if d.IsExpired() {
		return ErrCodeExpired
	}
	if d.Activated {
		return ErrAlreadyActive
	}

	now := time.Now()
	d.Activated = true
	d.ActivatedAt = &now
	d.DisplayID = &displayID
	return nil
}

// Repository defines storage operations for device codes
type Repository interface {
	Save(code *DeviceCode) error
	FindByDeviceCode(code string) (*DeviceCode, error)
	FindByUserCode(code string) (*DeviceCode, error)
	Delete(id uuid.UUID) error
}
