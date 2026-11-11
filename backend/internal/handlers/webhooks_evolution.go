package handlers

import (
	"bytes"
	"crypto/hmac"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/queue"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

type WebhookEvolution struct {
	repo     *repo.WhatsAppRepo
	enqueuer *queue.Enqueuer
}

func NewWebhookEvolution(repo *repo.WhatsAppRepo, enqueuer *queue.Enqueuer) *WebhookEvolution {
	return &WebhookEvolution{
		repo:     repo,
		enqueuer: enqueuer,
	}
}

var acceptedEvolutionEvents = map[string]bool{
	"Message":        true,
	"SendMessage":    true,
	"Connected":      true,
	"PairSuccess":    true,
	"QRSuccess":      true,
	"ProfileUpdated": true,
	"Disconnected":   true,
	"LoggedOut":      true,
	"ConnectFailure": true,
	"TemporaryBan":   true,
	"QRCode":         true,
	"QRTimeout":      true,
}

func (h *WebhookEvolution) HandleBySecret(w http.ResponseWriter, r *http.Request) {
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
	if inst.IsExternalUazapi() || looksLikeUazapiPayload(rawBody) {
		h.processUazapiPayload(w, r, inst, rawBody)
		return
	}
	h.processEnvelopeBytes(w, r, inst, rawBody, true)
}

func (h *WebhookEvolution) HandleUazapiBySecret(w http.ResponseWriter, r *http.Request) {
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
	h.processUazapiPayload(w, r, inst, rawBody)
}

// HandleLegacyByName mantém retrocompatibilidade com instâncias antigas configuradas com instanceName.
func (h *WebhookEvolution) HandleLegacyByName(w http.ResponseWriter, r *http.Request) {
	instanceName := strings.TrimSpace(chi.URLParam(r, "instanceName"))
	if instanceName == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	inst, err := h.repo.GetInstanceByName(r.Context(), instanceName)
	if err != nil || inst == nil {
		w.WriteHeader(http.StatusOK)
		return
	}
	rawBody, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read error", http.StatusBadRequest)
		return
	}
	h.processEnvelopeBytes(w, r, inst, rawBody, false)
}

func looksLikeUazapiPayload(raw []byte) bool {
	unwrapped := queue.UnwrapUazapiEnvelope(raw)
	return bytes.Contains(unwrapped, []byte(`"EventType"`)) || bytes.Contains(unwrapped, []byte(`"wa_chatid"`))
}

func (h *WebhookEvolution) processUazapiPayload(w http.ResponseWriter, r *http.Request, inst *domain.WhatsAppInstance, rawBody []byte) {
	if !inst.Enabled {
		w.WriteHeader(http.StatusOK)
		return
	}
	unwrapped := queue.UnwrapUazapiEnvelope(rawBody)
	meta := queue.ParseUazapiWebhookMeta(unwrapped)
	evType := strings.ToLower(meta.EventType)
	if evType == "" && bytes.Contains(unwrapped, []byte(`"message"`)) {
		evType = "messages"
	}

	var queueEvent string
	var queueData json.RawMessage

	switch evType {
	case "messages", "message":
		if queue.IsUazapiIgnorableMessage(unwrapped) {
			w.WriteHeader(http.StatusOK)
			return
		}
		queueEvent = "UazapiMessage"
		queueData = queue.StripHeavyUazapiThumbnails(unwrapped)
	case "connection":
		queueEvent = "UazapiConnection"
		queueData = unwrapped
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
			slog.Error("enqueue uazapi webhook event", "instance_id", inst.ID, "event", queueEvent, "err", err)
			http.Error(w, "queue error", http.StatusInternalServerError)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
}

func (h *WebhookEvolution) processEnvelopeBytes(w http.ResponseWriter, r *http.Request, inst *domain.WhatsAppInstance, rawBody []byte, hasSecretInPath bool) {
	var env struct {
		Event         string          `json:"event"`
		InstanceToken string          `json:"instanceToken"`
		Data          json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(rawBody, &env); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	if env.InstanceToken != "" && inst.InstanceToken != "" {
		if !hmac.Equal([]byte(env.InstanceToken), []byte(inst.InstanceToken)) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	} else if !hasSecretInPath {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if !inst.Enabled || !acceptedEvolutionEvents[env.Event] {
		w.WriteHeader(http.StatusOK)
		return
	}

	if env.Event == "Message" || env.Event == "SendMessage" {
		if isQuickIgnorableGroupOrStatus(env.Data) {
			w.WriteHeader(http.StatusOK)
			return
		}
		env.Data = queue.StripHeavyBase64FromEventData(env.Data)
	}

	if h.enqueuer != nil {
		if err := h.enqueuer.EnqueueWhatsAppEvent(r.Context(), queue.WhatsAppEventPayload{
			AccountID:  inst.AccountID,
			InstanceID: inst.ID,
			Event:      env.Event,
			Data:       env.Data,
		}); err != nil {
			slog.Error("enqueue whatsapp webhook event", "instance_id", inst.ID, "event", env.Event, "err", err)
			http.Error(w, "queue error", http.StatusInternalServerError)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
}

func isQuickIgnorableGroupOrStatus(raw json.RawMessage) bool {
	var quick struct {
		Info struct {
			Chat    string `json:"Chat"`
			IsGroup bool   `json:"IsGroup"`
		} `json:"Info"`
	}
	if json.Unmarshal(raw, &quick) != nil {
		return false
	}
	if quick.Info.IsGroup {
		return true
	}
	c := strings.ToLower(strings.TrimSpace(quick.Info.Chat))
	return strings.HasSuffix(c, "@g.us") ||
		strings.HasSuffix(c, "@broadcast") ||
		strings.HasSuffix(c, "@newsletter") ||
		c == "status@broadcast"
}
