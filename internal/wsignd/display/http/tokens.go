package http

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/wrale/wrale-signage/api/types/v1alpha1"
	"github.com/wrale/wrale-signage/internal/wsignd/auth"
	werrors "github.com/wrale/wrale-signage/internal/wsignd/errors"
)

// RefreshToken handles token refresh requests
func (h *Handler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetReqID(r.Context())

	// Get refresh token from Authorization header
	refreshToken := r.Header.Get("Authorization")
	if refreshToken == "" {
		h.logger.Error("missing refresh token in request",
			"requestID", reqID,
		)
		http.Error(w, "missing refresh token", http.StatusUnauthorized)
		return
	}

	// Remove "Bearer " prefix if present
	if strings.HasPrefix(refreshToken, "Bearer ") {
		refreshToken = refreshToken[7:]
	}

	// Generate new token pair
	token, err := h.auth.RefreshToken(r.Context(), refreshToken)
	if err != nil {
		status := http.StatusInternalServerError
		if err == auth.ErrTokenExpired || err == auth.ErrTokenNotFound {
			status = http.StatusUnauthorized
		}
		h.logger.Error("failed to refresh token",
			"error", err,
			"requestID", reqID,
		)
		h.writeError(w, err, status)
		return
	}

	// Return new tokens
	resp := &v1alpha1.DisplayAuthTokens{
		AccessToken:      token.AccessToken,
		RefreshToken:     token.RefreshToken,
		TokenType:        "Bearer",
		ExpiresIn:        int(time.Until(token.AccessTokenExpiry).Seconds()),
		RefreshExpiresIn: int(time.Until(token.RefreshTokenExpiry).Seconds()),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("failed to encode response",
			"error", err,
			"requestID", reqID,
		)
		h.writeError(w, werrors.NewError("ENCODING_ERROR", "failed to encode response", "RefreshToken", err), http.StatusInternalServerError)
		return
	}
}
