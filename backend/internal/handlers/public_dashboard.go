package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/renankeny/conversion-tracker/backend/internal/cache"
	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

type PublicDashboard struct {
	pixels   *repo.PixelRepo
	stats    *repo.StatsRepo
	counters *repo.EventCounterRepo
	sent     *repo.SentEventRepo
	qcache   *cache.QueryCache
}

func NewPublicDashboard(pixels *repo.PixelRepo, stats *repo.StatsRepo, counters *repo.EventCounterRepo, sent *repo.SentEventRepo) *PublicDashboard {
	return &PublicDashboard{pixels: pixels, stats: stats, counters: counters, sent: sent}
}

func (h *PublicDashboard) SetQueryCache(qc *cache.QueryCache) {
	h.qcache = qc
}

func (h *PublicDashboard) writeCached(w http.ResponseWriter, r *http.Request, endpoint string, p *domain.Pixel, loader func(context.Context) (any, error)) {
	key := cache.HashQueryKey("pub:dash:"+endpoint, p.Name, r.URL.RawQuery)
	raw, err := h.qcache.GetOrLoadBytes(r.Context(), key, 30*time.Second, func(ctx context.Context) ([]byte, error) {
		data, lerr := loader(ctx)
		if lerr != nil {
			return nil, lerr
		}
		return json.Marshal(data)
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, endpoint+" error: "+err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write(raw)
}

// resolvePixel busca o pixel pelo token do link. 404 se o token não existe,
// foi revogado, ou o pixel foi desabilitado.
func (h *PublicDashboard) resolvePixel(r *http.Request) (*domain.Pixel, error) {
	token := chi.URLParam(r, "token")
	p, err := h.pixels.FindByDashboardShareToken(r.Context(), token)
	if err != nil {
		return nil, err
	}
	if !p.Enabled {
		return nil, repo.ErrNotFound
	}
	return p, nil
}

func parsePublicRange(r *http.Request) (time.Time, time.Time) {
	q := r.URL.Query()
	from := time.Now().AddDate(0, 0, -30)
	to := time.Now()
	if v := q.Get("from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			from = t
		}
	}
	if v := q.Get("to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			to = t
		}
	}
	return from, to
}

// Meta GET /public/dashboard/{token}/meta — nome do pixel + config da dashboard.
func (h *PublicDashboard) Meta(w http.ResponseWriter, r *http.Request) {
	p, err := h.resolvePixel(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "link inválido ou desativado")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"pixel_label":      firstNonEmpty(p.Label, p.Name),
		"dashboard_config": p.DashboardConfig,
	})
}

// Summary GET /public/dashboard/{token}/summary?from=&to=
func (h *PublicDashboard) Summary(w http.ResponseWriter, r *http.Request) {
	p, err := h.resolvePixel(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "link inválido ou desativado")
		return
	}
	from, to := parsePublicRange(r)
	h.writeCached(w, r, "summary", p, func(ctx context.Context) (any, error) {
		rows, err := h.counters.SummaryByEvent(ctx, repo.SummaryFilters{
			From: &from, To: &to, PixelName: p.Name, AccountID: p.AccountID,
		})
		if err != nil {
			return nil, err
		}
		return map[string]any{"events": rows}, nil
	})
}

// Series GET /public/dashboard/{token}/series?from=&to=&granularity=hour|day
func (h *PublicDashboard) Series(w http.ResponseWriter, r *http.Request) {
	p, err := h.resolvePixel(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "link inválido ou desativado")
		return
	}
	from, to := parsePublicRange(r)
	daily := r.URL.Query().Get("granularity") == "day"
	h.writeCached(w, r, "series", p, func(ctx context.Context) (any, error) {
		pts, err := h.counters.TimeSeries(ctx, repo.SummaryFilters{
			From: &from, To: &to, PixelName: p.Name, AccountID: p.AccountID,
		}, daily)
		if err != nil {
			return nil, err
		}
		return map[string]any{"granularity": granularityLabel(daily), "points": pts}, nil
	})
}

// Geo GET /public/dashboard/{token}/geo?from=&to=
func (h *PublicDashboard) Geo(w http.ResponseWriter, r *http.Request) {
	p, err := h.resolvePixel(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "link inválido ou desativado")
		return
	}
	from, to := parsePublicRange(r)
	h.writeCached(w, r, "geo", p, func(ctx context.Context) (any, error) {
		return h.stats.GeoDistribution(ctx, repo.StatsFilter{
			From: from, To: to, PixelName: p.Name, AccountID: p.AccountID,
		})
	})
}

// GeoByEvent GET /public/dashboard/{token}/geo-by-event?from=&to=&event=
func (h *PublicDashboard) GeoByEvent(w http.ResponseWriter, r *http.Request) {
	p, err := h.resolvePixel(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "link inválido ou desativado")
		return
	}
	from, to := parsePublicRange(r)
	eventName := r.URL.Query().Get("event")
	h.writeCached(w, r, "geo_event", p, func(ctx context.Context) (any, error) {
		return h.stats.GeoDistributionByEvent(ctx, repo.StatsFilter{
			From: from, To: to, PixelName: p.Name, AccountID: p.AccountID,
		}, eventName)
	})
}

// Purchases GET /public/dashboard/{token}/purchases?from=&to=&limit=
func (h *PublicDashboard) Purchases(w http.ResponseWriter, r *http.Request) {
	p, err := h.resolvePixel(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "link inválido ou desativado")
		return
	}
	from, to := parsePublicRange(r)
	h.writeCached(w, r, "purchases", p, func(ctx context.Context) (any, error) {
		list, total, err := h.sent.ListGrouped(ctx, repo.ListFilter{
			AccountID: p.AccountID, PixelName: p.Name, EventNames: []string{"Purchase"},
			From: from, To: to, Limit: 50, IncludeLTV: true,
		})
		if err != nil {
			return nil, err
		}
		return map[string]any{"data": list, "total": total}, nil
	})
}

// Revenue GET /public/dashboard/{token}/revenue?from=&to=
func (h *PublicDashboard) Revenue(w http.ResponseWriter, r *http.Request) {
	p, err := h.resolvePixel(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "link inválido ou desativado")
		return
	}
	from, to := parsePublicRange(r)
	h.writeCached(w, r, "revenue", p, func(ctx context.Context) (any, error) {
		overview, err := h.stats.Overview(ctx, repo.StatsFilter{
			From: from, To: to, PixelName: p.Name, AccountID: p.AccountID,
		})
		if err != nil {
			return nil, err
		}
		return map[string]any{"total_revenue": overview.TotalRevenue}, nil
	})
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
