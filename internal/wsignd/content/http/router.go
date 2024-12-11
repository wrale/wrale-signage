package http

import (
	"github.com/go-chi/chi/v5"
	"net/http"
)

func NewRouter(h *Handler) chi.Router {
	r := chi.NewRouter()

	r.Post("/events", h.ReportEvents)
	r.Get("/health/{url}", h.GetURLHealth)
	r.Get("/metrics/{url}", h.GetURLMetrics)

	// Serve static content from demo directory
	contentFS := http.Dir("demo/0001_basic_setup_and_content/content")
	fileServer(r, "/content", contentFS)

	return r
}

// fileServer conveniently sets up a http.FileServer handler to serve static files
func fileServer(r chi.Router, path string, root http.FileSystem) {
	fs := http.StripPrefix(path, http.FileServer(root))

	r.Get(path+"/*", func(w http.ResponseWriter, r *http.Request) {
		fs.ServeHTTP(w, r)
	})
}
