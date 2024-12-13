package http

import (
	"github.com/go-chi/chi/v5"
)

// NewRouter creates a new HTTP router for display endpoints
// Deprecated: Use Handler.Router() instead for consistent middleware and route configuration
func NewRouter(h *Handler) chi.Router {
	return h.Router()
}
