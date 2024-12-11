package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/wrale/wrale-signage/api/types/v1alpha1"
	"github.com/wrale/wrale-signage/internal/wsignd/content"
)

type Handler struct {
	service content.Service
	logger  zerolog.Logger
}

func NewHandler(service content.Service, logger zerolog.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger.With().Str("component", "content-http").Logger(),
	}
}

func (h *Handler) decodeContentSource(r *http.Request) (*v1alpha1.ContentSource, error) {
	var source v1alpha1.ContentSource
	if err := json.NewDecoder(r.Body).Decode(&source); err != nil {
		return nil, ErrInvalidRequest("invalid request body")
	}
	return &source, nil
}

func (h *Handler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			h.logger.Error().Err(err).Msg("failed to encode response")
		}
	}
}

func (h *Handler) respondError(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	msg := "internal server error"

	if he, ok := err.(HTTPError); ok {
		code = he.StatusCode()
		msg = he.Error()
	}

	h.respondJSON(w, code, map[string]string{"error": msg})
}
