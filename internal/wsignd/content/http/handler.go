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

// Router returns a router pre-configured with all content endpoints mounted at the API root
func (h *Handler) Router() chi.Router {
	r := chi.NewRouter()
	h.RegisterRoutes(r)
	return r
}

// RegisterRoutes mounts all API endpoints on the provided router
func (h *Handler) RegisterRoutes(r chi.Router) {
	// Content management endpoints
	r.Post("/", h.handleCreateContent)
	r.Get("/", h.handleListContent)
	r.Route("/{name}", func(r chi.Router) {
		r.Get("/", h.handleGetContent)
		r.Put("/", h.handleUpdateContent)
		r.Delete("/", h.handleDeleteContent)
	})

	// Content event reporting
	r.Post("/events", h.handleReportEvents)

	// Content monitoring
	r.Get("/health/{url}", h.handleGetURLHealth)
	r.Get("/metrics/{url}", h.handleGetURLMetrics)
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
