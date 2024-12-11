package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/wrale/wrale-signage/api/types/v1alpha1"
	"github.com/wrale/wrale-signage/internal/wsignd/content"
)

func (h *Handler) handleReportEvents(w http.ResponseWriter, r *http.Request) {
	var batch v1alpha1.ContentEventBatch
	if err := json.NewDecoder(r.Body).Decode(&batch); err != nil {
		h.respondError(w, ErrInvalidRequest("invalid request body"))
		return
	}

	eventBatch := content.EventBatch{
		DisplayID: batch.DisplayID,
		Events:    convertAPIEvents(batch.Events),
	}

	if err := h.service.ReportEvents(r.Context(), eventBatch); err != nil {
		h.respondError(w, err)
		return
	}

	h.respondJSON(w, http.StatusAccepted, nil)
}

func (h *Handler) handleGetURLHealth(w http.ResponseWriter, r *http.Request) {
	url := chi.URLParam(r, "url")
	if url == "" {
		h.respondError(w, ErrInvalidRequest("url parameter is required"))
		return
	}

	status, err := h.service.GetURLHealth(r.Context(), url)
	if err != nil {
		h.respondError(w, err)
		return
	}

	h.respondJSON(w, http.StatusOK, status)
}

func (h *Handler) handleGetURLMetrics(w http.ResponseWriter, r *http.Request) {
	url := chi.URLParam(r, "url")
	if url == "" {
		h.respondError(w, ErrInvalidRequest("url parameter is required"))
		return
	}

	metrics, err := h.service.GetURLMetrics(r.Context(), url)
	if err != nil {
		h.respondError(w, err)
		return
	}

	h.respondJSON(w, http.StatusOK, metrics)
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
