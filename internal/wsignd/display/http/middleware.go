package http

import (
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

// requestIDHeaderMiddleware copies request ID from context to response header
func requestIDHeaderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if reqID := middleware.GetReqID(r.Context()); reqID != "" {
			w.Header().Set("Request-ID", reqID)
		}
		next.ServeHTTP(w, r)
	})
}
