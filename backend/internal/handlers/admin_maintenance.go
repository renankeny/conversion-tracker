package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/renankeny/conversion-tracker/backend/internal/maintenance"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

// AdminMaintenance dispara rotinas de manutenção sob demanda.
type AdminMaintenance struct {
	pool                       *pgxpool.Pool
	accounts                   *repo.AccountRepo
	sentEventsRetentionDays    int
	eventCountersRetentionDays int
}

func NewAdminMaintenance(pool *pgxpool.Pool, accounts *repo.AccountRepo, sentDays, countersDays int) *AdminMaintenance {
	return &AdminMaintenance{
		pool:                       pool,
		accounts:                   accounts,
		sentEventsRetentionDays:    sentDays,
		eventCountersRetentionDays: countersDays,
	}
}

// Status retorna a configuração atual de retenção.
func (h *AdminMaintenance) Status(w http.ResponseWriter, r *http.Request) {
	out := map[string]any{
		"sent_events_retention_days":    h.sentEventsRetentionDays,
		"event_counters_retention_days": h.eventCountersRetentionDays,
		"now":                           time.Now().Format(time.RFC3339),
	}

	scope := TenantFromContext(r.Context())
	accountID := scope.EffectiveAccountID(0)
	if accountID > 0 {
		acc, err := h.accounts.FindByID(r.Context(), accountID)
		if err == nil && acc != nil {
			out["account_id"] = acc.ID
			out["account_sent_events_retention_days"] = acc.SentEventsRetentionDays
			out["account_event_counters_retention_days"] = acc.EventCountersRetentionDays
			out["effective_sent_events_retention_days"] = pickRetention(acc.SentEventsRetentionDays, h.sentEventsRetentionDays)
			out["effective_event_counters_retention_days"] = pickRetention(acc.EventCountersRetentionDays, h.eventCountersRetentionDays)
		}
	}

	writeJSON(w, http.StatusOK, out)
}

// PruneNow executa o prune sincronamente para a conta atual ou global.
func (h *AdminMaintenance) PruneNow(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	scope := TenantFromContext(r.Context())
	sentDays := h.sentEventsRetentionDays
	countersDays := h.eventCountersRetentionDays

	accountID := scope.EffectiveAccountID(0)
	if accountID > 0 && h.accounts != nil {
		if acc, err := h.accounts.FindByID(ctx, accountID); err == nil && acc != nil {
			sentDays = pickRetention(acc.SentEventsRetentionDays, sentDays)
			countersDays = pickRetention(acc.EventCountersRetentionDays, countersDays)
		}
	}

	if v := r.URL.Query().Get("sent_events_days"); v != "" {
		if n, err := parsePositiveInt(v); err == nil {
			sentDays = n
		}
	}
	if v := r.URL.Query().Get("event_counters_days"); v != "" {
		if n, err := parsePositiveInt(v); err == nil {
			countersDays = n
		}
	}

	results := []maintenance.PruneResult{}

	if scope.IsOwner && accountID == 0 {
		r1, err := maintenance.PruneOldSentEvents(ctx, h.pool, sentDays)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "sent_events: "+err.Error())
			return
		}
		results = append(results, r1)
		r2, err := maintenance.PruneOldEventCounters(ctx, h.pool, countersDays)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "event_counters: "+err.Error())
			return
		}
		results = append(results, r2)
	} else {
		if accountID <= 0 {
			writeError(w, http.StatusBadRequest, "account_id obrigatório")
			return
		}
		r1, err := maintenance.PruneOldSentEventsForAccount(ctx, h.pool, accountID, sentDays)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "sent_events: "+err.Error())
			return
		}
		results = append(results, r1)
		r2, err := maintenance.PruneOldEventCountersForAccount(ctx, h.pool, accountID, countersDays)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "event_counters: "+err.Error())
			return
		}
		results = append(results, r2)
	}

	writeJSON(w, http.StatusOK, map[string]any{"results": results})
}

type updateRetentionInput struct {
	SentEventsRetentionDays    *int   `json:"sent_events_retention_days"`
	EventCountersRetentionDays *int   `json:"event_counters_retention_days"`
	AccountID                  *int64 `json:"account_id"`
}

// UpdateRetention salva o override de retenção da conta.
func (h *AdminMaintenance) UpdateRetention(w http.ResponseWriter, r *http.Request) {
	if h.accounts == nil {
		writeError(w, http.StatusServiceUnavailable, "accounts repo indisponível")
		return
	}
	var in updateRetentionInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	scope := TenantFromContext(r.Context())
	override := int64(0)
	if in.AccountID != nil {
		override = *in.AccountID
	}
	accountID := scope.EffectiveAccountID(override)
	if accountID == 0 {
		writeError(w, http.StatusBadRequest, "account_id obrigatório (owner global precisa especificar)")
		return
	}
	if !scope.IsOwner && accountID != scope.AccountID {
		writeError(w, http.StatusForbidden, "sem acesso a esta conta")
		return
	}

	if in.SentEventsRetentionDays != nil && *in.SentEventsRetentionDays < 0 {
		writeError(w, http.StatusBadRequest, "sent_events_retention_days não pode ser negativo")
		return
	}
	if in.EventCountersRetentionDays != nil && *in.EventCountersRetentionDays < 0 {
		writeError(w, http.StatusBadRequest, "event_counters_retention_days não pode ser negativo")
		return
	}

	if err := h.accounts.UpdateRetention(r.Context(), accountID, in.SentEventsRetentionDays, in.EventCountersRetentionDays); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "conta não encontrada")
			return
		}
		writeError(w, http.StatusInternalServerError, "update error: "+err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type resetInput struct {
	Confirm   bool   `json:"confirm"`
	AccountID *int64 `json:"account_id"`
}

// Reset apaga TODOS os sent_events e event_counters da conta (exige `confirm: true`).
func (h *AdminMaintenance) Reset(w http.ResponseWriter, r *http.Request) {
	var in resetInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if !in.Confirm {
		writeError(w, http.StatusConflict, "operação destrutiva: envie {\"confirm\": true}")
		return
	}
	scope := TenantFromContext(r.Context())
	override := int64(0)
	if in.AccountID != nil {
		override = *in.AccountID
	}
	accountID := scope.EffectiveAccountID(override)
	if accountID == 0 {
		writeError(w, http.StatusBadRequest, "account_id obrigatório (owner global precisa especificar)")
		return
	}
	if !scope.IsOwner && accountID != scope.AccountID {
		writeError(w, http.StatusForbidden, "sem acesso a esta conta")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()
	sent, counters, err := maintenance.ResetAccountData(ctx, h.pool, accountID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "reset error: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"account_id":             accountID,
		"deleted_sent_events":    sent,
		"deleted_event_counters": counters,
	})
}

func pickRetention(override *int, fallback int) int {
	if override != nil {
		return *override
	}
	return fallback
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
