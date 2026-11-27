package handlers

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/queue"
)

func (h *WebhookEvolution) HandleWAHABySecret(w http.ResponseWriter, r *http.Request) {
	secret := strings.TrimSpace(chi.URLParam(r, "secret"))
	if secret == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	inst, err := h.repo.GetInstanceByWebhookSecret(r.Context(), secret)
	if err != nil || inst == nil {
		w.WriteHeader(http.StatusOK)
		return
	}
	rawBody, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read error", http.StatusBadRequest)
		return
	}
	h.processWAHAPayload(w, r, inst, rawBody)
}

func (h *WebhookEvolution) processWAHAPayload(w http.ResponseWriter, r *http.Request, inst *domain.WhatsAppInstance, rawBody []byte) {
	if !inst.Enabled {
		w.WriteHeader(http.StatusOK)
		return
	}
	var env struct {
		Event string `json:"event"`
	}
	if json.Unmarshal(rawBody, &env) != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	ev := strings.ToLower(strings.TrimSpace(env.Event))
	var queueEvent string
	var queueData json.RawMessage

	switch ev {
	case "message.any", "message":
		if queue.IsWAHAIgnorableMessage(rawBody) {
			w.WriteHeader(http.StatusOK)
			return
		}
		queueEvent = "WAHAMessage"
		queueData = queue.StripHeavyWAHAThumbnails(rawBody)
	case "session.status":
		queueEvent = "WAHASessionStatus"
		queueData = rawBody
	default:
		w.WriteHeader(http.StatusOK)
		return
	}

	if h.enqueuer != nil {
		if err := h.enqueuer.EnqueueWhatsAppEvent(r.Context(), queue.WhatsAppEventPayload{
			AccountID:  inst.AccountID,
			InstanceID: inst.ID,
			Event:      queueEvent,
			Data:       queueData,
		}); err != nil {
			slog.Error("enqueue waha webhook event", "instance_id", inst.ID, "event", queueEvent, "err", err)
			http.Error(w, "queue error", http.StatusInternalServerError)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
}
