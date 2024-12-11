package http

import (
	"github.com/go-chi/chi/v5"
)

func NewRouter(h *Handler) chi.Router {
	r := chi.NewRouter()

	r.Post("/events", h.ReportEvents)
	r.Get("/health/{url}", h.GetURLHealth)
	r.Get("/metrics/{url}", h.GetURLMetrics)

	return r
}