package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

// BulkDelete DELETE /admin/sent-events
// Remove múltiplos eventos por event_id. Máximo 500 event_ids por chamada.
func (h *AdminSentEvents) BulkDelete(w http.ResponseWriter, r *http.Request) {
	var body struct {
		EventIDs []string `json:"event_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "json inválido")
		return
	}
	if len(body.EventIDs) == 0 {
		writeError(w, http.StatusBadRequest, "event_ids não pode ser vazio")
		return
	}
	if len(body.EventIDs) > 500 {
		writeError(w, http.StatusBadRequest, "máximo 500 event_ids por operação")
		return
	}

	scope := TenantFromContext(r.Context())
	deleted, err := h.repo.DeleteByEventIDs(r.Context(), scope.AccountID, body.EventIDs)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "delete error: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": deleted})
}

// Retry POST /admin/sent-events/retry
// Reenfileira eventos falhos para reprocessamento (máximo 100 por chamada).
func (h *AdminSentEvents) Retry(w http.ResponseWriter, r *http.Request) {
	if h.enqueuer == nil {
		writeError(w, http.StatusServiceUnavailable, "retry não disponível")
		return
	}

	var body struct {
		EventIDs []string `json:"event_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "json inválido")
		return
	}
	if len(body.EventIDs) == 0 {
		writeError(w, http.StatusBadRequest, "event_ids não pode ser vazio")
		return
	}
	if len(body.EventIDs) > 100 {
		writeError(w, http.StatusBadRequest, "máximo 100 event_ids por operação")
		return
	}

	scope := TenantFromContext(r.Context())
	inputs, err := h.repo.FindInputsByEventIDs(r.Context(), scope.AccountID, body.EventIDs)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao buscar inputs: "+err.Error())
		return
	}

	queued, notFound := 0, 0
	for _, eventID := range body.EventIDs {
		raw, ok := inputs[eventID]
		if !ok {
			notFound++
			continue
		}
		var in domain.SendEventInput
		if err := json.Unmarshal(raw, &in); err != nil {
			slog.Warn("retry: unmarshal input failed", "event_id", eventID, "err", err)
			notFound++
			continue
		}
		in.IsReplay = true
		if err := h.enqueuer.EnqueueSendEvent(r.Context(), in); err != nil {
			slog.Warn("retry: enqueue failed", "event_id", eventID, "err", err)
			continue
		}
		queued++
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"queued":    queued,
		"not_found": notFound,
		"total":     len(body.EventIDs),
	})
}
