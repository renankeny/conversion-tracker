package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/renankeny/conversion-tracker/backend/internal/maintenance"
)

// AdminMaintenance dispara rotinas de manutenção sob demanda.
// Útil pro admin testar a regra de retenção sem esperar o cron.
type AdminMaintenance struct {
	pool                       *pgxpool.Pool
	sentEventsRetentionDays    int
	eventCountersRetentionDays int
}

func NewAdminMaintenance(pool *pgxpool.Pool, sentDays, countersDays int) *AdminMaintenance {
	return &AdminMaintenance{
		pool:                       pool,
		sentEventsRetentionDays:    sentDays,
		eventCountersRetentionDays: countersDays,
	}
}

// Status retorna a configuração atual de retenção.
func (h *AdminMaintenance) Status(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"sent_events_retention_days":    h.sentEventsRetentionDays,
		"event_counters_retention_days": h.eventCountersRetentionDays,
		"now":                           time.Now().Format(time.RFC3339),
	})
}

// PruneNow executa o prune sincronamente. Use override `?days=N` se quiser
// rodar com uma retenção diferente da configurada (ex: limpar mais agressivo
// uma vez só).
func (h *AdminMaintenance) PruneNow(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	sentRetention := h.sentEventsRetentionDays
	countersRetention := h.eventCountersRetentionDays

	// Override opcional via query param.
	if v := r.URL.Query().Get("sent_events_days"); v != "" {
		if n, err := parsePositiveInt(v); err == nil {
			sentRetention = n
		}
	}
	if v := r.URL.Query().Get("event_counters_days"); v != "" {
		if n, err := parsePositiveInt(v); err == nil {
			countersRetention = n
		}
	}

	results := []maintenance.PruneResult{}

	if r1, err := maintenance.PruneOldSentEvents(ctx, h.pool, sentRetention); err != nil {
		writeError(w, http.StatusInternalServerError, "sent_events: "+err.Error())
		return
	} else {
		results = append(results, r1)
	}

	if r2, err := maintenance.PruneOldEventCounters(ctx, h.pool, countersRetention); err != nil {
		writeError(w, http.StatusInternalServerError, "event_counters: "+err.Error())
		return
	} else {
		results = append(results, r2)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"results": results,
	})
}

func parsePositiveInt(s string) (int, error) {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, errInvalidInt
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}

var errInvalidInt = &simpleError{msg: "invalid int"}

type simpleError struct{ msg string }

func (e *simpleError) Error() string { return e.msg }
