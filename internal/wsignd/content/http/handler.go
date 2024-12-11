package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/wrale/wrale-signage/internal/wsignd/content"
)

type Handler struct {
	service *content.Service
	logger  *slog.Logger
}

func NewHandler(service *content.Service, logger *slog.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

func (h *Handler) ReportEvents(w http.ResponseWriter, r *http.Request) {
	var batch content.EventBatch
	if err := json.NewDecoder(r.Body).Decode(&batch); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.service.ReportEvents(r.Context(), batch); err != nil {
		h.logger.Error("failed to process events",
			"error", err,
			"displayId", batch.DisplayID,
		)
		http.Error(w, "failed to process events", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) GetURLHealth(w http.ResponseWriter, r *http.Request) {
	url := chi.URLParam(r, "url")
	if url == "" {
		http.Error(w, "missing url parameter", http.StatusBadRequest)
		return
	}

	health, err := h.service.GetURLHealth(r.Context(), url)
	if err != nil {
		h.logger.Error("failed to get URL health",
			"error", err,
			"url", url,
		)
		http.Error(w, "health check failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(health); err != nil {
		h.logger.Error("failed to encode response",
			"error", err,
		)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) GetURLMetrics(w http.ResponseWriter, r *http.Request) {
	url := chi.URLParam(r, "url")
	if url == "" {
		http.Error(w, "missing url parameter", http.StatusBadRequest)
		return
	}

	metrics, err := h.service.GetURLMetrics(r.Context(), url)
	if err != nil {
		h.logger.Error("failed to get URL metrics",
			"error", err,
			"url", url,
		)
		http.Error(w, "metrics retrieval failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(metrics); err != nil {
		h.logger.Error("failed to encode response",
			"error", err,
		)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
}
