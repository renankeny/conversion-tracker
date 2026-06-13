package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/renankeny/conversion-tracker/backend/internal/repo"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

// AdminURLMappings expõe o CRUD de mapeamentos black->white de URLs no painel admin.
//
// Após qualquer mutação, recarrega o cache em memória (URLMapper) sincronamente
// para que o efeito seja imediato sem esperar o ticker periódico.
type AdminURLMappings struct {
	repo *repo.URLMappingRepo
	urls *util.URLMapper
}

func NewAdminURLMappings(r *repo.URLMappingRepo, urls *util.URLMapper) *AdminURLMappings {
	return &AdminURLMappings{repo: r, urls: urls}
}

type urlMappingInput struct {
	Source  string `json:"source"`
	Target  string `json:"target"`
	Enabled *bool  `json:"enabled"`
}

func (h *AdminURLMappings) List(w http.ResponseWriter, r *http.Request) {
	list, err := h.repo.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": list})
}

func (h *AdminURLMappings) Create(w http.ResponseWriter, r *http.Request) {
	var in urlMappingInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if in.Source == "" || in.Target == "" {
		writeError(w, http.StatusBadRequest, "source and target are required")
		return
	}
	m := &repo.URLMapping{
		Source:  in.Source,
		Target:  in.Target,
		Enabled: in.Enabled == nil || *in.Enabled,
	}
	if err := h.repo.Create(r.Context(), m); err != nil {
		writeError(w, http.StatusInternalServerError, "create error: "+err.Error())
		return
	}
	h.refreshCache(r.Context())
	writeJSON(w, http.StatusCreated, m)
}

func (h *AdminURLMappings) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var in urlMappingInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	m := &repo.URLMapping{ID: id, Source: in.Source, Target: in.Target}
	if in.Enabled != nil {
		m.Enabled = *in.Enabled
	} else {
		m.Enabled = true
	}
	if err := h.repo.Update(r.Context(), m); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "update error: "+err.Error())
		return
	}
	h.refreshCache(r.Context())
	writeJSON(w, http.StatusOK, m)
}

func (h *AdminURLMappings) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "delete error")
		return
	}
	h.refreshCache(r.Context())
	w.WriteHeader(http.StatusNoContent)
}

// refreshCache lê o estado atual do banco e atualiza o URLMapper em memória.
// Erros aqui apenas geram log: o ticker periódico vai recuperar.
func (h *AdminURLMappings) refreshCache(ctx context.Context) {
	m, err := h.repo.ActiveAsMap(ctx)
	if err != nil {
		slog.Warn("url_mappings cache refresh failed", "err", err)
		return
	}
	h.urls.Set(m)
}
