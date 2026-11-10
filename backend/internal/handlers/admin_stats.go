package handlers

import (
	"net/http"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

// AdminStats expõe o GET /admin/stats pra dashboard do painel.
type AdminStats struct {
	repo *repo.StatsRepo
}

func NewAdminStats(r *repo.StatsRepo) *AdminStats {
	return &AdminStats{repo: r}
}

// Overview GET /admin/stats?from=&to=&pixel=
//
// `from`/`to` em RFC3339. Se omitidos, default são "últimos 30 dias".
// `pixel` filtra por pixel_name exato.
func (h *AdminStats) Overview(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := repo.StatsFilter{
		PixelName:     q.Get("pixel"),
		TrafficSource: q.Get("traffic_source"),
	}

	scope := TenantFromContext(r.Context())
	f.AccountID = scope.AccountID

	if v := q.Get("from"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid 'from' (RFC3339)")
			return
		}
		f.From = t
	} else {
		f.From = time.Now().AddDate(0, 0, -30)
	}
	if v := q.Get("to"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid 'to' (RFC3339)")
			return
		}
		f.To = t
	}

	overview, err := h.repo.Overview(r.Context(), f)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "stats error: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, overview)
}

// Geo GET /admin/stats/geo?from=&to=&pixel=
func (h *AdminStats) Geo(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := repo.StatsFilter{PixelName: q.Get("pixel")}

	scope := TenantFromContext(r.Context())
	f.AccountID = scope.AccountID

	if v := q.Get("from"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid 'from' (RFC3339)")
			return
		}
		f.From = t
	} else {
		f.From = time.Now().AddDate(0, 0, -30)
	}
	if v := q.Get("to"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid 'to' (RFC3339)")
			return
		}
		f.To = t
	}

	geo, err := h.repo.GeoDistribution(r.Context(), f)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "geo stats error: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, geo)
}

// GeoByEvent GET /admin/stats/geo-by-event?from=&to=&pixel=&event=
func (h *AdminStats) GeoByEvent(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := repo.StatsFilter{PixelName: q.Get("pixel")}
	eventName := q.Get("event") // PageView, Purchase, etc

	scope := TenantFromContext(r.Context())
	f.AccountID = scope.AccountID

	if v := q.Get("from"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid 'from' (RFC3339)")
			return
		}
		f.From = t
	} else {
		f.From = time.Now().AddDate(0, 0, -30)
	}
	if v := q.Get("to"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid 'to' (RFC3339)")
			return
		}
		f.To = t
	}

	geo, err := h.repo.GeoDistributionByEvent(r.Context(), f, eventName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "geo by event stats error: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, geo)
}

