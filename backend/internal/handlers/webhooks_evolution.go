package handlers

import (
	"crypto/hmac"
	"encoding/json"
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
	h.processEnvelope(w, r, inst, true)
}

// HandleLegacyByName mantém retrocompatibilidade com instâncias antigas configuradas com instanceName,
// exigindo validação em tempo constante do instanceToken enviado pela Evolution Go.
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
	h.processEnvelope(w, r, inst, false)
}

func (h *WebhookEvolution) processEnvelope(w http.ResponseWriter, r *http.Request, inst *domain.WhatsAppInstance, hasSecretInPath bool) {
	var env struct {
		Event         string          `json:"event"`
		InstanceToken string          `json:"instanceToken"`
		Data          json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&env); err != nil {
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
