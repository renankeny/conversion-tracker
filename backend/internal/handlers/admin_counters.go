package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/cache"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

// AdminCounters expõe agregados de eventos a partir de event_counters e sent_events.
type AdminCounters struct {
	repo   *repo.EventCounterRepo
	qcache *cache.QueryCache
}

func NewAdminCounters(r *repo.EventCounterRepo) *AdminCounters {
	return &AdminCounters{repo: r}
}

func (h *AdminCounters) SetQueryCache(qc *cache.QueryCache) {
	h.qcache = qc
}

// Summary GET /admin/counters/summary?from=ISO&to=ISO&pixel=NAME&event=NAME
func (h *AdminCounters) Summary(w http.ResponseWriter, r *http.Request) {
	f, err := parseSummaryFilters(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var toVal time.Time
	if f.To != nil {
		toVal = *f.To
	}
	ttl := dashboardCacheTTL(toVal, 30*time.Second, 5*time.Minute)
	writeDashboardCachedJSON(w, r, h.qcache, f.AccountID, "counters_summary", ttl, "summary error", func(ctx context.Context) (any, error) {
		rows, lerr := h.repo.SummaryByEvent(ctx, f)
		if lerr != nil {
			return nil, lerr
		}
		return map[string]any{
			"from":   f.From,
			"to":     f.To,
			"pixel":  f.PixelName,
			"event":  f.EventName,
			"events": rows,
		}, nil
	})
}

// TimeSeries GET /admin/counters/series?from=ISO&to=ISO&pixel=NAME&event=NAME&granularity=hour|day
func (h *AdminCounters) TimeSeries(w http.ResponseWriter, r *http.Request) {
	f, err := parseSummaryFilters(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	daily := r.URL.Query().Get("granularity") == "day"
	var toVal time.Time
	if f.To != nil {
		toVal = *f.To
	}
	ttl := dashboardCacheTTL(toVal, 30*time.Second, 5*time.Minute)
	writeDashboardCachedJSON(w, r, h.qcache, f.AccountID, "counters_series", ttl, "series error", func(ctx context.Context) (any, error) {
		pts, lerr := h.repo.TimeSeries(ctx, f, daily)
		if lerr != nil {
			return nil, lerr
		}
		return map[string]any{
			"from":        f.From,
			"to":          f.To,
			"pixel":       f.PixelName,
			"event":       f.EventName,
			"granularity": granularityLabel(daily),
			"points":      pts,
		}, nil
	})
}

func parseSummaryFilters(r *http.Request) (repo.SummaryFilters, error) {
	q := r.URL.Query()
	f := repo.SummaryFilters{
		PixelName:     q.Get("pixel"),
		EventName:     q.Get("event"),
		TrafficSource: q.Get("traffic_source"),
		AccountID:     TenantFromContext(r.Context()).AccountID,
	}

	if v := q.Get("from"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return f, err
		}
		f.From = &t
	}
	if v := q.Get("to"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return f, err
		}
		f.To = &t
	}
	now := time.Now()
	if f.From == nil {
		then := now.AddDate(0, 0, -30)
		f.From = &then
	}
	if f.To == nil {
		f.To = &now
	}
	return f, nil
}

func granularityLabel(daily bool) string {
	if daily {
		return "day"
	}
	return "hour"
}
