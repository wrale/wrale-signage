package http

import (
	"github.com/go-chi/chi/v5"
	"net/http"
)

func NewRouter(h *Handler) chi.Router {
	r := chi.NewRouter()

	// Content management endpoints
	r.Post("/", h.CreateContent)
	r.Get("/", h.ListContent)

	// Content event reporting
	r.Post("/events", h.ReportEvents)

	// Content monitoring
	r.Get("/health/{url}", h.GetURLHealth)
	r.Get("/metrics/{url}", h.GetURLMetrics)

	return r
}
