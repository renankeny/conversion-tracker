package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
	"github.com/renankeny/conversion-tracker/backend/internal/waha"
	"github.com/renankeny/conversion-tracker/backend/internal/wshub"
)

func handleWAHASessionStatus(ctx context.Context, deps Deps, inst *domain.WhatsAppInstance, raw json.RawMessage) {
	var env wahaWebhookEnvelope
	if json.Unmarshal(raw, &env) != nil {
		return
	}
	var body struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	_ = json.Unmarshal(env.Payload, &body)
	status := strings.ToUpper(strings.TrimSpace(body.Status))

	switch status {
	case "WORKING":
		clearPendingWhatsAppDisconnect(ctx, deps.Redis, inst.ID)
		ownerBare := resolveWAHAMeOwnerJID(env.Me)
		profName := ""
		if env.Me != nil {
			profName = strings.TrimSpace(env.Me.PushName)
		}
		profPic := ""
		if inst.ServerURL != "" && (ownerBare == "" || profName == "" || strings.TrimSpace(inst.ProfilePicURL) == "") {
			cli := waha.NewClient(inst.ServerURL, inst.ServerAPIKey)
			if prof, err := cli.GetProfile(ctx, inst.InstanceName); err == nil && prof != nil {
				ownerBare = util.FirstNonEmpty(ownerBare, NormalizeWAHAJID(prof.ID))
				profName = util.FirstNonEmpty(profName, prof.Name)
				profPic = util.FirstNonEmpty(profPic, prof.Picture)
			}
		}
		ownerBare = util.FirstNonEmpty(ownerBare, inst.OwnerJID)
		profName = util.FirstNonEmpty(profName, inst.ProfileName)
		profPic = util.FirstNonEmpty(profPic, inst.ProfilePicURL)

		wasOpen := strings.EqualFold(inst.ConnectionStatus, "OPEN")
		if deps.WhatsApp != nil {
			_ = deps.WhatsApp.UpdateInstanceOwnerAndProfile(ctx, inst.ID, "OPEN", ownerBare, profName, profPic)
		}
		inst.ConnectionStatus = "OPEN"
		inst.OwnerJID = ownerBare
		inst.ProfileName = profName
		inst.ProfilePicURL = profPic
		wshub.PublishCustom(ctx, deps.Redis, inst.AccountID, "whatsapp_status", map[string]any{
			"instance_id":     inst.ID,
			"status":          "OPEN",
			"owner_jid":       ownerBare,
			"profile_name":    profName,
			"profile_pic_url": profPic,
		})
		if !wasOpen {
			notifyWhatsAppStateChange(deps, inst, true)
		}

	case "STARTING", "SCAN_QR_CODE":
		if strings.EqualFold(inst.ConnectionStatus, "CONNECTING") {
			return
		}
		if deps.WhatsApp != nil {
			_ = deps.WhatsApp.UpdateInstanceStatus(ctx, inst.ID, "CONNECTING")
		}
		inst.ConnectionStatus = "CONNECTING"
		wshub.PublishCustom(ctx, deps.Redis, inst.AccountID, "whatsapp_status", map[string]any{
			"instance_id": inst.ID,
			"status":      "CONNECTING",
		})

	case "STOPPED":
		applyConfirmedWhatsAppDisconnect(ctx, deps, inst, "WAHAStopped", true)

	case "FAILED":
		handleWAHADisconnectSignal(ctx, deps, inst, "WAHAFailed")
	}
}

func handleWAHADisconnectSignal(ctx context.Context, deps Deps, inst *domain.WhatsAppInstance, event string) {
	if strings.EqualFold(inst.ConnectionStatus, "CLOSED") || strings.EqualFold(inst.ConnectionStatus, "DISCONNECTED") {
		return
	}
	nowUnix := time.Now().Unix()
	if deps.Enqueuer != nil {
		if !markPendingWhatsAppDisconnect(ctx, deps.Redis, inst.ID, nowUnix) {
			return
		}
		err := deps.Enqueuer.EnqueueWhatsAppDisconnectCheck(ctx, WhatsAppVerifyDisconnectPayload{
			AccountID:       inst.AccountID,
			InstanceID:      inst.ID,
			Event:           event,
			TriggeredAtUnix: nowUnix,
		}, whatsappDisconnectGracePeriod)
		if err == nil {
			slog.Debug("waha transient disconnect queued for confirmation", "instance_id", inst.ID, "event", event)
			return
		}
		clearPendingWhatsAppDisconnect(ctx, deps.Redis, inst.ID)
	}
	_ = verifyAndConfirmWAHADisconnect(ctx, deps, inst, event, true)
}

func verifyAndConfirmWAHADisconnect(ctx context.Context, deps Deps, inst *domain.WhatsAppInstance, event string, confirmOnNetErr bool) error {
	if inst.ServerURL == "" || inst.InstanceName == "" {
		applyConfirmedWhatsAppDisconnect(ctx, deps, inst, event, false)
		return nil
	}
	cli := waha.NewClient(inst.ServerURL, inst.ServerAPIKey)
	sess, err := cli.GetSession(ctx, inst.InstanceName)
	if err != nil {
		if errors.Is(err, waha.ErrSessionNotFound) {
			applyConfirmedWhatsAppDisconnect(ctx, deps, inst, event, true)
			return nil
		}
		if !confirmOnNetErr {
			return fmt.Errorf("probe waha session %d: %w", inst.ID, err)
		}
		applyConfirmedWhatsAppDisconnect(ctx, deps, inst, event, false)
		return nil
	}
	status := strings.ToUpper(strings.TrimSpace(sess.Status))
	if status == "WORKING" {
		clearPendingWhatsAppDisconnect(ctx, deps.Redis, inst.ID)
		ownerBare := inst.OwnerJID
		profName := inst.ProfileName
		if sess.Me != nil {
			ownerBare = util.FirstNonEmpty(NormalizeWAHAJID(sess.Me.JID), NormalizeWAHAJID(sess.Me.ID), ownerBare)
			profName = util.FirstNonEmpty(strings.TrimSpace(sess.Me.PushName), profName)
		}
		if deps.WhatsApp != nil {
			_ = deps.WhatsApp.UpdateInstanceOwnerAndProfile(ctx, inst.ID, "OPEN", ownerBare, profName, inst.ProfilePicURL)
		}
		return nil
	}
	resetSession := status == "STOPPED" || status == "SCAN_QR_CODE" || sess.Me == nil
	applyConfirmedWhatsAppDisconnect(ctx, deps, inst, event, resetSession)
	return nil
}

// lookupPhoneByLIDWithWAHAFallback busca no banco e, em instâncias WAHA, faz fallback na API /lids/{lid}.
func lookupPhoneByLIDWithWAHAFallback(ctx context.Context, deps Deps, inst *domain.WhatsAppInstance, lidJID string) string {
	if p := deps.WhatsApp.ResolvePhoneByLID(ctx, inst.AccountID, lidJID); p != "" {
		return p
	}
	if resolved := resolveWAHALIDRemote(ctx, inst, lidJID); resolved != "" {
		_ = deps.WhatsApp.UpsertLIDMapping(ctx, inst.AccountID, lidJID, resolved)
		return resolved
	}
	return ""
}

// resolveWAHALIDRemote tenta resolver @lid consultando a API da WAHA caso ainda não esteja em cache no Postgres.
func resolveWAHALIDRemote(ctx context.Context, inst *domain.WhatsAppInstance, lidJID string) string {
	if inst == nil || !inst.IsWAHA() || inst.ServerURL == "" || inst.InstanceName == "" {
		return ""
	}
	probeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	pn := waha.NewClient(inst.ServerURL, inst.ServerAPIKey).ResolveLID(probeCtx, inst.InstanceName, lidJID)
	return NormalizeWAHAJID(pn)
}

// fetchWAHALeadProfilePic busca a foto de perfil do lead na WAHA apenas quando ainda não salva no chat.
func fetchWAHALeadProfilePic(ctx context.Context, inst *domain.WhatsAppInstance, existingChat *domain.WhatsAppChat, contactID string) *string {
	if inst == nil || !inst.IsWAHA() || inst.ServerURL == "" || inst.InstanceName == "" || contactID == "" {
		return nil
	}
	if existingChat != nil && existingChat.ProfilePic != nil && strings.TrimSpace(*existingChat.ProfilePic) != "" {
		return existingChat.ProfilePic
	}
	probeCtx, cancel := context.WithTimeout(ctx, 2500*time.Millisecond)
	defer cancel()
	pic := waha.NewClient(inst.ServerURL, inst.ServerAPIKey).GetContactProfilePicture(probeCtx, inst.InstanceName, contactID)
	if pic == "" {
		return nil
	}
	return &pic
}
