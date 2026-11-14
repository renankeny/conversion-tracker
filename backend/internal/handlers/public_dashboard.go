package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
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
	metaH    *AdminDashboardMeta
	googleH  *AdminDashboardGoogle
	qcache   *cache.QueryCache
}

func NewPublicDashboard(pixels *repo.PixelRepo, stats *repo.StatsRepo, counters *repo.EventCounterRepo, sent *repo.SentEventRepo) *PublicDashboard {
	return &PublicDashboard{pixels: pixels, stats: stats, counters: counters, sent: sent}
}

func (h *PublicDashboard) SetQueryCache(qc *cache.QueryCache) { h.qcache = qc }

func (h *PublicDashboard) writeCached(w http.ResponseWriter, r *http.Request, endpoint string, p *domain.Pixel, to time.Time, loader func(context.Context) (any, error)) {
	ver := h.qcache.DashboardVersion(r.Context(), p.AccountID)
	ttl := dashboardCacheTTL(to, 30*time.Second, 5*time.Minute)
	key := cache.HashQueryKey("pub:dash:v3:"+endpoint, strconv.FormatInt(p.AccountID, 10), ver, p.Name, r.URL.RawQuery)
	raw, err := h.qcache.GetOrLoadBytes(r.Context(), key, ttl, func(ctx context.Context) ([]byte, error) {
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
	from, to := parseDashboardTimeBounds(r)
	if from.IsZero() {
		from = time.Now().AddDate(0, 0, -30)
	}
	if to.IsZero() {
		to = time.Now()
	}
	return from, to
}

func (h *PublicDashboard) Meta(w http.ResponseWriter, r *http.Request) {
	p, err := h.resolvePixel(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "link inválido ou desativado")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"pixel_label":      firstNonEmpty(p.Label, p.Name),
		"currency":         firstNonEmpty(p.TrackerConfig.ProductCurrency, "BRL"),
		"dashboard_config": p.DashboardConfig,
	})
}

func (h *PublicDashboard) Summary(w http.ResponseWriter, r *http.Request) {
	p, err := h.resolvePixel(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "link inválido ou desativado")
		return
	}
	from, to := parsePublicRange(r)
	src := r.URL.Query().Get("traffic_source")
	h.writeCached(w, r, "summary", p, to, func(ctx context.Context) (any, error) {
		rows, err := h.counters.SummaryByEvent(ctx, repo.SummaryFilters{
			From: &from, To: &to, PixelName: p.Name, AccountID: p.AccountID, TrafficSource: src,
		})
		if err != nil {
			return nil, err
		}
		return map[string]any{"events": rows}, nil
	})
}

func (h *PublicDashboard) Series(w http.ResponseWriter, r *http.Request) {
	p, err := h.resolvePixel(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "link inválido ou desativado")
		return
	}
	from, to := parsePublicRange(r)
	daily := r.URL.Query().Get("granularity") == "day"
	src := r.URL.Query().Get("traffic_source")
	h.writeCached(w, r, "series", p, to, func(ctx context.Context) (any, error) {
		pts, err := h.counters.TimeSeries(ctx, repo.SummaryFilters{
			From: &from, To: &to, PixelName: p.Name, AccountID: p.AccountID, TrafficSource: src,
		}, daily)
		if err != nil {
			return nil, err
		}
		return map[string]any{"granularity": granularityLabel(daily), "points": pts}, nil
	})
}

func (h *PublicDashboard) Geo(w http.ResponseWriter, r *http.Request) {
	p, err := h.resolvePixel(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "link inválido ou desativado")
		return
	}
	from, to := parsePublicRange(r)
	src := r.URL.Query().Get("traffic_source")
	h.writeCached(w, r, "geo", p, to, func(ctx context.Context) (any, error) {
		return h.stats.GeoDistribution(ctx, repo.StatsFilter{
			From: from, To: to, PixelName: p.Name, AccountID: p.AccountID, TrafficSource: src,
		})
	})
}

func (h *PublicDashboard) GeoByEvent(w http.ResponseWriter, r *http.Request) {
	p, err := h.resolvePixel(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "link inválido ou desativado")
		return
	}
	from, to := parsePublicRange(r)
	eventName := r.URL.Query().Get("event")
	src := r.URL.Query().Get("traffic_source")
	h.writeCached(w, r, "geo_event", p, to, func(ctx context.Context) (any, error) {
		return h.stats.GeoDistributionByEvent(ctx, repo.StatsFilter{
			From: from, To: to, PixelName: p.Name, AccountID: p.AccountID, TrafficSource: src,
		}, eventName)
	})
}

func (h *PublicDashboard) Purchases(w http.ResponseWriter, r *http.Request) {
	p, err := h.resolvePixel(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "link inválido ou desativado")
		return
	}
	from, to := parsePublicRange(r)
	src := r.URL.Query().Get("traffic_source")
	h.writeCached(w, r, "purchases", p, to, func(ctx context.Context) (any, error) {
		// Mesmo filtro do painel administrativo (sem filtrar apenas status="sent",
		// mantendo paridade exata com Últimas Compras, Vendas e Total Vendido).
		list, total, err := h.sent.ListGrouped(ctx, repo.ListFilter{
			AccountID: p.AccountID, PixelName: p.Name, EventNames: []string{"Purchase"},
			From: from, To: to, Limit: 50, IncludeLTV: true, IncludeBuyer: true, TrafficSource: src,
		})
		if err != nil {
			return nil, err
		}
		return map[string]any{"data": list, "total": total}, nil
	})
}

func (h *PublicDashboard) Revenue(w http.ResponseWriter, r *http.Request) {
	p, err := h.resolvePixel(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "link inválido ou desativado")
		return
	}
	from, to := parsePublicRange(r)
	src := r.URL.Query().Get("traffic_source")
	h.writeCached(w, r, "revenue", p, to, func(ctx context.Context) (any, error) {
		overview, err := h.stats.Overview(ctx, repo.StatsFilter{
			From: from, To: to, PixelName: p.Name, AccountID: p.AccountID, TrafficSource: src,
		})
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"total_revenue":    overview.TotalRevenue,
			"traffic_quality":  overview.TrafficQuality,
			"source_breakdown": overview.SourceBreakdown,
			"panel_audit":      overview.PanelAudit,
			"previous_period":  overview.PreviousPeriod,
			"heatmap":          overview.Heatmap,
		}, nil
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
