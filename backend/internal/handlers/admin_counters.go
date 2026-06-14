package handlers

import (
	"net/http"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

// AdminCounters expõe agregados de eventos a partir de event_counters.
// É a fonte do dashboard "1300 visitas, 900 InitiateCheckout, 30 Purchases".
type AdminCounters struct {
	repo *repo.EventCounterRepo
}

func NewAdminCounters(r *repo.EventCounterRepo) *AdminCounters {
	return &AdminCounters{repo: r}
}

// Summary retorna totais por evento dentro do intervalo.
//   GET /admin/counters/summary?from=ISO&to=ISO&pixel=NAME&event=NAME
//
// Padrão (sem from/to): últimos 30 dias.
func (h *AdminCounters) Summary(w http.ResponseWriter, r *http.Request) {
	f, err := parseSummaryFilters(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	rows, err := h.repo.SummaryByEvent(r.Context(), f)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "summary error: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"from":   f.From,
		"to":     f.To,
		"pixel":  f.PixelName,
		"event":  f.EventName,
		"events": rows,
	})
}

// TimeSeries retorna a evolução temporal por hora ou dia.
//   GET /admin/counters/series?from=ISO&to=ISO&pixel=NAME&event=NAME&granularity=hour|day
func (h *AdminCounters) TimeSeries(w http.ResponseWriter, r *http.Request) {
	f, err := parseSummaryFilters(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	daily := r.URL.Query().Get("granularity") == "day"
	pts, err := h.repo.TimeSeries(r.Context(), f, daily)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "series error: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"from":        f.From,
		"to":          f.To,
		"pixel":       f.PixelName,
		"event":       f.EventName,
		"granularity": granularityLabel(daily),
		"points":      pts,
	})
}

func parseSummaryFilters(r *http.Request) (repo.SummaryFilters, error) {
	q := r.URL.Query()
	f := repo.SummaryFilters{
		PixelName: q.Get("pixel"),
		EventName: q.Get("event"),
	}

	// Aplica filtro multi-tenant: members só veem a própria conta.
	scope := TenantFromContext(r.Context())
	if !scope.IsOwner {
		f.AccountID = scope.AccountID
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
	// Defaults razoáveis: últimos 30 dias se nada informado.
	if f.From == nil && f.To == nil {
		now := time.Now()
		then := now.AddDate(0, 0, -30)
		f.From = &then
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
