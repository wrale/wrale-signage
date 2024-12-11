package http

import (
	"github.com/go-chi/chi/v5"
)

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1alpha1/content", func(r chi.Router) {
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
	})
}
