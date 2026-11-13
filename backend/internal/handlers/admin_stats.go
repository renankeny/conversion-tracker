package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/cache"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

// AdminStats expõe o GET /admin/stats pra dashboard do painel.
type AdminStats struct {
	repo   *repo.StatsRepo
	qcache *cache.QueryCache
}

func NewAdminStats(r *repo.StatsRepo) *AdminStats {
	return &AdminStats{repo: r}
}

func (h *AdminStats) SetQueryCache(qc *cache.QueryCache) {
	h.qcache = qc
}

func parseStatsFilter(r *http.Request) (repo.StatsFilter, error) {
	q := r.URL.Query()
	now := time.Now()
	f := repo.StatsFilter{
		PixelName:     q.Get("pixel"),
		TrafficSource: q.Get("traffic_source"),
		AccountID:     TenantFromContext(r.Context()).AccountID,
		From:          now.AddDate(0, 0, -30),
		To:            now,
	}
	if v := q.Get("from"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return f, errors.New("invalid 'from' (RFC3339)")
		}
		f.From = t
	}
	if v := q.Get("to"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return f, errors.New("invalid 'to' (RFC3339)")
		}
		f.To = t
	}
	return f, nil
}

// Overview GET /admin/stats?from=&to=&pixel=
func (h *AdminStats) Overview(w http.ResponseWriter, r *http.Request) {
	f, err := parseStatsFilter(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	ttl := dashboardCacheTTL(f.To, 30*time.Second, 5*time.Minute)
	writeDashboardCachedJSON(w, r, h.qcache, f.AccountID, "stats_overview", ttl, "stats error", func(ctx context.Context) (any, error) {
		return h.repo.Overview(ctx, f)
	})
}

// Geo GET /admin/stats/geo?from=&to=&pixel=
func (h *AdminStats) Geo(w http.ResponseWriter, r *http.Request) {
	f, err := parseStatsFilter(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	ttl := dashboardCacheTTL(f.To, 60*time.Second, 10*time.Minute)
	writeDashboardCachedJSON(w, r, h.qcache, f.AccountID, "stats_geo", ttl, "geo stats error", func(ctx context.Context) (any, error) {
		return h.repo.GeoDistribution(ctx, f)
	})
}

// GeoByEvent GET /admin/stats/geo-by-event?from=&to=&pixel=&event=
func (h *AdminStats) GeoByEvent(w http.ResponseWriter, r *http.Request) {
	f, err := parseStatsFilter(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	eventName := r.URL.Query().Get("event")
	ttl := dashboardCacheTTL(f.To, 60*time.Second, 10*time.Minute)
	writeDashboardCachedJSON(w, r, h.qcache, f.AccountID, "stats_geo_event", ttl, "geo by event stats error", func(ctx context.Context) (any, error) {
		return h.repo.GeoDistributionByEvent(ctx, f, eventName)
	})
}
