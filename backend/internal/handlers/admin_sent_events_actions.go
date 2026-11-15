package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/cache"
	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
	"github.com/renankeny/conversion-tracker/backend/internal/wshub"
)

type sentEventBatchRequest struct {
	Items    []repo.EventRef `json:"items"`
	EventIDs []string        `json:"event_ids"`
}

func resolveEventRefs(items []repo.EventRef, eventIDs []string) []repo.EventRef {
	if len(items) > 0 {
		out := make([]repo.EventRef, 0, len(items))
		for _, it := range items {
			eid := strings.TrimSpace(it.EventID)
			if eid == "" {
				continue
			}
			out = append(out, repo.EventRef{
				EventID:   eid,
				PixelName: strings.TrimSpace(it.PixelName),
			})
		}
		return out
	}
	out := make([]repo.EventRef, 0, len(eventIDs))
	for _, raw := range eventIDs {
		eid := strings.TrimSpace(raw)
		if eid != "" {
			out = append(out, repo.EventRef{EventID: eid})
		}
	}
	return out
}

// BulkDelete DELETE /admin/sent-events
// Remove múltiplos eventos por (event_id, pixel_name) ou event_ids legados (máximo 500 por chamada).
func (h *AdminSentEvents) BulkDelete(w http.ResponseWriter, r *http.Request) {
	var body sentEventBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "json inválido")
		return
	}
	refs := resolveEventRefs(body.Items, body.EventIDs)
	if len(refs) == 0 {
		writeError(w, http.StatusBadRequest, "nenhum evento informado")
		return
	}
	if len(refs) > 500 {
		writeError(w, http.StatusBadRequest, "máximo de 500 eventos por operação")
		return
	}

	ctx := r.Context()
	scope := TenantFromContext(ctx)
	deleted, tuples, err := h.repo.DeleteByRefs(ctx, scope.AccountID, refs)
	if err != nil {
		slog.Error("bulk delete sent events failed", "err", err, "account_id", scope.AccountID)
		writeError(w, http.StatusInternalServerError, "erro ao excluir eventos")
		return
	}

	if h.rdb != nil && len(tuples) > 0 {
		affectedAccs := make(map[int64]bool, 2)
		if scope.AccountID > 0 {
			affectedAccs[scope.AccountID] = true
		}
		for _, t := range tuples {
			cache.ReleaseEventDedup(ctx, h.rdb, t.AccountID, t.PixelName, t.EventID)
			cache.ClearDoneTargets(ctx, h.rdb, t.AccountID, t.PixelName, t.EventID)
			if t.AccountID > 0 {
				affectedAccs[t.AccountID] = true
			}
		}
		for accID := range affectedAccs {
			cache.BumpDashboardVersion(ctx, h.rdb, accID)
			wshub.PublishCustom(ctx, h.rdb, accID, "events_updated", map[string]any{"action": "deleted"})
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"deleted": deleted})
}

// Retry POST /admin/sent-events/retry
// Reenfileira eventos falhos para reprocessamento (máximo 100 por chamada).
func (h *AdminSentEvents) Retry(w http.ResponseWriter, r *http.Request) {
	if h.enqueuer == nil {
		writeError(w, http.StatusServiceUnavailable, "reenvio indisponível no momento")
		return
	}

	var body sentEventBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "json inválido")
		return
	}
	refs := resolveEventRefs(body.Items, body.EventIDs)
	if len(refs) == 0 {
		writeError(w, http.StatusBadRequest, "nenhum evento informado")
		return
	}
	if len(refs) > 100 {
		writeError(w, http.StatusBadRequest, "máximo de 100 eventos por operação")
		return
	}

	ctx := r.Context()
	scope := TenantFromContext(ctx)
	inputs, notFound, err := h.repo.FindInputsByRefs(ctx, scope.AccountID, refs)
	if err != nil {
		slog.Error("find inputs for retry failed", "err", err, "account_id", scope.AccountID)
		writeError(w, http.StatusInternalServerError, "erro ao buscar eventos para reenvio")
		return
	}

	queued := 0
	for _, raw := range inputs {
		var in domain.SendEventInput
		if err := json.Unmarshal(raw, &in); err != nil {
			slog.Warn("retry: unmarshal input failed", "err", err)
			notFound++
			continue
		}
		in.IsReplay = true
		if err := h.enqueuer.EnqueueSendEvent(ctx, in); err != nil {
			slog.Warn("retry: enqueue failed", "event_id", in.EventID, "err", err)
			continue
		}
		queued++
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"queued":    queued,
		"not_found": notFound,
		"total":     len(refs),
	})
}
