package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/wrale/wrale-signage/api/types/v1alpha1"
)

func (h *Handler) handleCreateContent(w http.ResponseWriter, r *http.Request) {
	source, err := h.decodeContentSource(r)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to decode content source")
		h.respondError(w, err)
		return
	}

	if err := h.service.ValidateContent(r.Context(), source.Spec.URL); err != nil {
		h.logger.Error().Err(err).Str("url", source.Spec.URL).Msg("content validation failed")
		h.respondError(w, err)
		return
	}

	if err := h.service.CreateContent(r.Context(), source); err != nil {
		h.logger.Error().Err(err).Str("name", source.ObjectMeta.Name).Msg("failed to create content")
		h.respondError(w, err)
		return
	}

	h.logger.Info().Str("name", source.ObjectMeta.Name).Msg("content created successfully")
	h.respondJSON(w, http.StatusCreated, source)
}

func (h *Handler) handleUpdateContent(w http.ResponseWriter, r *http.Request) {
	var update v1alpha1.ContentSourceUpdate
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		h.logger.Error().Err(err).Msg("failed to decode content update")
		h.respondError(w, ErrInvalidRequest("invalid request body"))
		return
	}

	name := chi.URLParam(r, "name")
	source := &v1alpha1.ContentSource{
		ObjectMeta: v1alpha1.ObjectMeta{Name: name},
		Spec: v1alpha1.ContentSourceSpec{
			Properties: update.Properties,
		},
	}

	if update.URL != nil {
		source.Spec.URL = *update.URL
	}
	if update.PlaybackDuration != nil {
		source.Spec.PlaybackDuration = *update.PlaybackDuration
	}

	if err := h.service.UpdateContent(r.Context(), source); err != nil {
		h.logger.Error().Err(err).Str("name", name).Msg("failed to update content")
		h.respondError(w, err)
		return
	}

	h.logger.Info().Str("name", name).Msg("content updated successfully")
	h.respondJSON(w, http.StatusOK, source)
}

func (h *Handler) handleDeleteContent(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	if err := h.service.DeleteContent(r.Context(), name); err != nil {
		h.logger.Error().Err(err).Str("name", name).Msg("failed to delete content")
		h.respondError(w, err)
		return
	}

	h.logger.Info().Str("name", name).Msg("content deleted successfully")
	h.respondJSON(w, http.StatusNoContent, nil)
}

func (h *Handler) handleListContent(w http.ResponseWriter, r *http.Request) {
	sources, err := h.service.ListContent(r.Context())
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to list content")
		h.respondError(w, err)
		return
	}

	list := v1alpha1.ContentSourceList{
		TypeMeta: v1alpha1.TypeMeta{
			APIVersion: "wrale.io/v1alpha1",
			Kind:       "ContentSourceList",
		},
		Items: sources,
	}

	h.logger.Debug().Int("count", len(sources)).Msg("listing content sources")
	h.respondJSON(w, http.StatusOK, list)
}

func (h *Handler) handleGetContent(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	source, err := h.service.GetContent(r.Context(), name)
	if err != nil {
		h.logger.Error().Err(err).Str("name", name).Msg("failed to get content")
		h.respondError(w, err)
		return
	}

	h.logger.Debug().Str("name", name).Msg("retrieved content")
	h.respondJSON(w, http.StatusOK, source)
}
