package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"

	"github.com/renankeny/conversion-tracker/backend/internal/auth"
	"github.com/renankeny/conversion-tracker/backend/internal/cache"
	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

type AdminTrackableLinks struct {
	links        *repo.TrackableLinkRepo
	pixels       *repo.PixelRepo
	wa           *repo.WhatsAppRepo
	qcache       *cache.QueryCache
	rdb          *redis.Client
	publicAPIURL string
}

func NewAdminTrackableLinks(
	links *repo.TrackableLinkRepo,
	pixels *repo.PixelRepo,
	wa *repo.WhatsAppRepo,
	qcache *cache.QueryCache,
	rdb *redis.Client,
	publicAPIURL string,
) *AdminTrackableLinks {
	return &AdminTrackableLinks{
		links:        links,
		pixels:       pixels,
		wa:           wa,
		qcache:       qcache,
		rdb:          rdb,
		publicAPIURL: publicAPIURL,
	}
}

func (h *AdminTrackableLinks) ListByPixel(w http.ResponseWriter, r *http.Request) {
	pixel, ok := h.loadAuthorizedPixel(w, r, false)
	if !ok {
		return
	}
	overview, err := h.links.StatsByFilter(r.Context(), repo.TrackableLinkStatsFilter{
		AccountID: pixel.AccountID,
		PixelID:   pixel.ID,
	}, h.publicAPIURL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao listar links rastreáveis")
		return
	}
	writeJSON(w, http.StatusOK, overview)
}

func (h *AdminTrackableLinks) Create(w http.ResponseWriter, r *http.Request) {
	pixel, ok := h.loadAuthorizedPixel(w, r, true)
	if !ok {
		return
	}
	var in trackableLinkInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "JSON inválido")
		return
	}

	link, err := h.buildAndValidateInput(r.Context(), pixel, nil, in)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var createErr error
	for attempt := 0; attempt < 5; attempt++ {
		link.Slug = domain.GenerateTrackableLinkSlug(pixel.AccountID)
		createErr = h.links.Create(r.Context(), link)
		if createErr == nil || !isDuplicateKey(createErr) {
			break
		}
	}
	if createErr != nil {
		writeError(w, http.StatusInternalServerError, "erro ao criar link rastreável")
		return
	}

	link.TrackingHost = pixel.TrackingHost
	link.PixelName = pixel.Name
	link.PixelLabel = pixel.Label
	link.PublicURL = link.BuildPublicURL(h.publicAPIURL)
	if h.rdb != nil {
		cache.BumpDashboardVersion(r.Context(), h.rdb, pixel.AccountID)
	}
	writeJSON(w, http.StatusCreated, link)
}

func (h *AdminTrackableLinks) Update(w http.ResponseWriter, r *http.Request) {
	pixel, ok := h.loadAuthorizedPixel(w, r, true)
	if !ok {
		return
	}
	linkID, err := strconv.ParseInt(chi.URLParam(r, "linkID"), 10, 64)
	if err != nil || linkID <= 0 {
		writeError(w, http.StatusBadRequest, "ID do link inválido")
		return
	}
	existing, err := h.links.GetByID(r.Context(), linkID)
	if err != nil || existing == nil || existing.PixelID != pixel.ID || existing.AccountID != pixel.AccountID {
		writeError(w, http.StatusNotFound, "link rastreável não encontrado")
		return
	}

	var in trackableLinkInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	updated, err := h.buildAndValidateInput(r.Context(), pixel, existing, in)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.links.Update(r.Context(), updated); err != nil {
		if isDuplicateKey(err) {
			writeError(w, http.StatusConflict, "já existe um link rastreável com este slug (/w/"+updated.Slug+")")
			return
		}
		writeError(w, http.StatusInternalServerError, "erro ao salvar link rastreável")
		return
	}
	updated.TrackingHost = pixel.TrackingHost
	updated.PixelName = pixel.Name
	updated.PixelLabel = pixel.Label
	updated.PublicURL = updated.BuildPublicURL(h.publicAPIURL)
	if h.rdb != nil {
		cache.BumpDashboardVersion(r.Context(), h.rdb, pixel.AccountID)
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *AdminTrackableLinks) Delete(w http.ResponseWriter, r *http.Request) {
	pixel, ok := h.loadAuthorizedPixel(w, r, true)
	if !ok {
		return
	}
	linkID, err := strconv.ParseInt(chi.URLParam(r, "linkID"), 10, 64)
	if err != nil || linkID <= 0 {
		writeError(w, http.StatusBadRequest, "ID do link inválido")
		return
	}
	if err := h.links.Delete(r.Context(), pixel.AccountID, pixel.ID, linkID); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "link rastreável não encontrado")
			return
		}
		writeError(w, http.StatusInternalServerError, "erro ao remover link rastreável")
		return
	}
	if h.rdb != nil {
		cache.BumpDashboardVersion(r.Context(), h.rdb, pixel.AccountID)
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
}

// DashboardStats GET /api/v1/admin/trackable-links/stats?from=&to=&pixel_name=
func (h *AdminTrackableLinks) DashboardStats(w http.ResponseWriter, r *http.Request) {
	scope := TenantFromContext(r.Context())
	from, to := parseDashboardTimeBounds(r)
	var fromPtr, toPtr *time.Time
	if !from.IsZero() {
		fromPtr = &from
	}
	if !to.IsZero() {
		toPtr = &to
	}
	pixelName := strings.TrimSpace(r.URL.Query().Get("pixel_name"))
	if pixelName == "" {
		pixelName = strings.TrimSpace(r.URL.Query().Get("pixel"))
	}

	refTo := to
	if refTo.IsZero() {
		refTo = time.Now()
	}
	ttl := dashboardCacheTTL(refTo, 20*time.Second, 3*time.Minute)
	writeDashboardCachedJSON(w, r, h.qcache, scope.AccountID, "trackable_links_stats", ttl, "erro ao carregar métricas de links rastreáveis", func(ctx context.Context) (any, error) {
		return h.links.StatsByFilter(ctx, repo.TrackableLinkStatsFilter{
			AccountID: scope.AccountID,
			PixelName: pixelName,
			From:      fromPtr,
			To:        toPtr,
		}, h.publicAPIURL)
	})
}

func (h *AdminTrackableLinks) loadAuthorizedPixel(w http.ResponseWriter, r *http.Request, requireEdit bool) (*domain.Pixel, bool) {
	if requireEdit && !auth.CanEditPixels(r.Context()) {
		writeError(w, http.StatusForbidden, "você não tem permissão para editar pixels")
		return nil, false
	}
	pixelID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || pixelID <= 0 {
		writeError(w, http.StatusBadRequest, "ID de pixel inválido")
		return nil, false
	}
	pixel, err := h.pixels.FindByID(r.Context(), pixelID)
	if err != nil {
		writeError(w, http.StatusNotFound, "pixel não encontrado")
		return nil, false
	}
	scope := TenantFromContext(r.Context())
	if !scope.IsOwner && pixel.AccountID != scope.AccountID {
		writeError(w, http.StatusForbidden, "sem acesso a este pixel")
		return nil, false
	}
	return pixel, true
}
