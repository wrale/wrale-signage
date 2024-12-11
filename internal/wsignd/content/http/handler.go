package http

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/wrale/wrale-signage/api/types/v1alpha1"
	"github.com/wrale/wrale-signage/internal/wsignd/content"
)

type Handler struct {
	service content.Service
}

func NewHandler(service content.Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) CreateContent(w http.ResponseWriter, r *http.Request) {
	var req v1alpha1.ContentSource
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if err := req.Spec.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate URL is accessible
	if err := h.service.ValidateContent(r.Context(), req.Spec.URL); err != nil {
		http.Error(w, "content validation failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Update status fields
	req.Status.LastValidated = time.Now()
	req.Status.IsHealthy = true
	req.Status.Version = 1

	// Return created content
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(req); err != nil {
		http.Error(w, "error encoding response", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) ReportEvents(w http.ResponseWriter, r *http.Request) {
	var batch v1alpha1.ContentEventBatch
	if err := json.NewDecoder(r.Body).Decode(&batch); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	eventBatch := content.EventBatch{
		DisplayID: batch.DisplayID,
		Events:    convertAPIEvents(batch.Events),
	}

	if err := h.service.ReportEvents(r.Context(), eventBatch); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) GetURLHealth(w http.ResponseWriter, r *http.Request) {
	url := chi.URLParam(r, "url")
	if url == "" {
		http.Error(w, "url parameter is required", http.StatusBadRequest)
		return
	}

	status, err := h.service.GetURLHealth(r.Context(), url)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(status); err != nil {
		http.Error(w, "error encoding response", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) GetURLMetrics(w http.ResponseWriter, r *http.Request) {
	url := chi.URLParam(r, "url")
	if url == "" {
		http.Error(w, "url parameter is required", http.StatusBadRequest)
		return
	}

	metrics, err := h.service.GetURLMetrics(r.Context(), url)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(metrics); err != nil {
		http.Error(w, "error encoding response", http.StatusInternalServerError)
		return
	}
}

func convertAPIEvents(apiEvents []v1alpha1.ContentEvent) []content.Event {
	events := make([]content.Event, len(apiEvents))
	for i, e := range apiEvents {
		events[i] = content.Event{
			ID:        e.ID,
			DisplayID: e.DisplayID,
			Type:      content.EventType(e.Type),
			URL:       e.URL,
			Timestamp: e.Timestamp,
		}
		if e.Error != nil {
			events[i].Error = &content.EventError{
				Code:    e.Error.Code,
				Message: e.Error.Message,
				Details: e.Error.Details,
			}
		}
		if e.Metrics != nil {
			events[i].Metrics = &content.EventMetrics{
				LoadTime:        e.Metrics.LoadTime,
				RenderTime:      e.Metrics.RenderTime,
				InteractiveTime: e.Metrics.InteractiveTime,
			}
		}
		if e.Context != nil {
			events[i].Context = e.Context
		}
	}
	return events
}
