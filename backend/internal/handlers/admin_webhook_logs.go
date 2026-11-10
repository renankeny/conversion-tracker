package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

// AdminWebhookLogs expõe o histórico de webhooks recebidos e permite
// replay manual de eventos que falharam ou precisam ser reenviados.
type AdminWebhookLogs struct {
	repo     *repo.WebhookLogRepo
	webhooks *Webhooks
}

func NewAdminWebhookLogs(r *repo.WebhookLogRepo, w *Webhooks) *AdminWebhookLogs {
	return &AdminWebhookLogs{repo: r, webhooks: w}
}

// List GET /admin/webhook-logs
//
// Query params:
//
//	source   — filtra por gateway (hotmart, cakto, kirvano, kiwify, pagtrust, yampi)
//	status   — queued | failed | replayed
//	from/to  — RFC3339
//	limit    — default 50
//	offset   — default 0
func (h *AdminWebhookLogs) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := repo.WebhookLogFilter{
		Source: q.Get("source"),
		Status: q.Get("status"),
	}

	scope := TenantFromContext(r.Context())
	f.AccountID = scope.AccountID

	if v := q.Get("from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			f.From = t
		}
	}
	if v := q.Get("to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			f.To = t
		}
	}
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			f.Limit = n
		}
	}
	if v := q.Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			f.Offset = n
		}
	}

	logs, total, err := h.repo.List(r.Context(), f)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list error: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data":   logs,
		"total":  total,
		"limit":  f.Limit,
		"offset": f.Offset,
	})
}

// Replay POST /admin/webhook-logs/{id}/replay
//
// Reprocessa um webhook_log pelo ID, executando novamente o lookup de
// usuário e o enfileiramento do evento.
func (h *AdminWebhookLogs) Replay(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id inválido")
		return
	}

	var accountID int64
	scope := TenantFromContext(r.Context())
	accountID = scope.AccountID

	result, err := h.webhooks.ReplayWebhook(r.Context(), id, accountID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, result)
}
