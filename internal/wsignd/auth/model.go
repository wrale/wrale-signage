package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrTokenNotFound = errors.New("token not found")
	ErrTokenExpired  = errors.New("token expired")
	ErrTokenInvalid  = errors.New("invalid token")
	ErrTokenMismatch = errors.New("token mismatch")
)

// Token represents a display's authentication tokens
type Token struct {
	ID                 uuid.UUID
	DisplayID          uuid.UUID
	AccessToken        string // Plain text, only populated on creation
	RefreshToken       string // Plain text, only populated on creation
	AccessTokenHash    []byte
	RefreshTokenHash   []byte
	AccessTokenExpiry  time.Time
	RefreshTokenExpiry time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// NewToken creates a new token pair for a display
func NewToken(displayID uuid.UUID) (*Token, error) {
	// Generate random tokens using crypto/rand wrapped by uuid package
	accessTokenBytes := uuid.New().String() + uuid.New().String()
	refreshTokenBytes := uuid.New().String() + uuid.New().String()

	accessToken := base64.RawURLEncoding.EncodeToString([]byte(accessTokenBytes))
	refreshToken := base64.RawURLEncoding.EncodeToString([]byte(refreshTokenBytes))

	// Hash tokens for storage
	accessHash := sha256.Sum256([]byte(accessToken))
	refreshHash := sha256.Sum256([]byte(refreshToken))

	now := time.Now()
	return &Token{
		ID:                 uuid.New(),
		DisplayID:          displayID,
		AccessToken:        accessToken,  // Only available at creation
		RefreshToken:       refreshToken, // Only available at creation
		AccessTokenHash:    accessHash[:],
		RefreshTokenHash:   refreshHash[:],
		AccessTokenExpiry:  now.Add(1 * time.Hour),       // 1 hour
		RefreshTokenExpiry: now.Add(90 * 24 * time.Hour), // 90 days
		CreatedAt:          now,
		UpdatedAt:          now,
	}, nil
}

// IsAccessTokenExpired checks if the access token is expired
func (t *Token) IsAccessTokenExpired() bool {
	return time.Now().After(t.AccessTokenExpiry)
}

// IsRefreshTokenExpired checks if the refresh token is expired
func (t *Token) IsRefreshTokenExpired() bool {
	return time.Now().After(t.RefreshTokenExpiry)
}

// ValidateAccessToken validates an access token
func (t *Token) ValidateAccessToken(token string) error {
	if t.IsAccessTokenExpired() {
		return ErrTokenExpired
	}

	hash := sha256.Sum256([]byte(token))
	if !byteSlicesEqual(hash[:], t.AccessTokenHash) {
		return ErrTokenInvalid
	}

	return nil
}

// ValidateRefreshToken validates a refresh token
func (t *Token) ValidateRefreshToken(token string) error {
	if t.IsRefreshTokenExpired() {
		return ErrTokenExpired
	}

	hash := sha256.Sum256([]byte(token))
	if !byteSlicesEqual(hash[:], t.RefreshTokenHash) {
		return ErrTokenInvalid
	}

	return nil
}

// Helper for constant-time comparison
func byteSlicesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}

	var result byte
	for i := 0; i < len(a); i++ {
		result |= a[i] ^ b[i]
	}
	return result == 0
}
