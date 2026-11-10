package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/renankeny/conversion-tracker/backend/internal/cache"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

type AdminURLMappings struct {
	repo *repo.URLMappingRepo
	urls *util.URLMapper
	pool *pgxpool.Pool
	rdb  *redis.Client
}

func NewAdminURLMappings(r *repo.URLMappingRepo, urls *util.URLMapper) *AdminURLMappings {
	return &AdminURLMappings{repo: r, urls: urls}
}

func (h *AdminURLMappings) SetPool(pool *pgxpool.Pool) {
	h.pool = pool
}

func (h *AdminURLMappings) SetRedis(rdb *redis.Client) {
	h.rdb = rdb
}

type urlMappingInput struct {
	Source    string `json:"source"`
	Target    string `json:"target"`
	Enabled   *bool  `json:"enabled"`
	AccountID *int64 `json:"account_id"` // owners podem definir; members ignorado
}

func (h *AdminURLMappings) List(w http.ResponseWriter, r *http.Request) {
	scope := TenantFromContext(r.Context())
	var accountID int64 = scope.AccountID
	list, err := h.repo.List(r.Context(), accountID)
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
	targetAccountID, err := ResolveTargetAccountID(r.Context(), h.pool, in.AccountID)
	if err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	m := &repo.URLMapping{
		AccountID: targetAccountID,
		Source:    in.Source,
		Target:    in.Target,
		Enabled:   in.Enabled == nil || *in.Enabled,
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
	existing, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "load error")
		return
	}
	scope := TenantFromContext(r.Context())
	if !scope.IsOwner && existing.AccountID != scope.AccountID {
		writeError(w, http.StatusForbidden, "sem acesso")
		return
	}
	var in urlMappingInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	m := &repo.URLMapping{ID: id, AccountID: existing.AccountID, Source: in.Source, Target: in.Target}
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
	existing, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "load error")
		return
	}
	scope := TenantFromContext(r.Context())
	if !scope.IsOwner && existing.AccountID != scope.AccountID {
		writeError(w, http.StatusForbidden, "sem acesso")
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
	byAcc, err := h.repo.ActiveByAccountMap(ctx)
	if err != nil {
		slog.Warn("url_mappings cache refresh failed", "err", err)
		return
	}
	h.urls.SetByAccount(byAcc)
	cache.PublishInvalidate(ctx, h.rdb, cache.InvalidateEvent{Kind: "url_mappings"})
}
