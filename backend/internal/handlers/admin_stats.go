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
	f := repo.StatsFilter{PixelName: q.Get("pixel")}

	if scope := TenantFromContext(r.Context()); !scope.IsOwner {
		f.AccountID = scope.AccountID
	}

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
