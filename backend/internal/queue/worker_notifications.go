package queue

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/notify"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

func emitDispatchNotifications(
	ctx context.Context,
	deps Deps,
	pixel *domain.Pixel,
	in domain.SendEventInput,
	results []dispatchResult,
	hasErr bool,
	retryCount int,
) {
	if deps.Notify == nil || pixel == nil || in.IsReplay {
		return
	}

	if in.EventName == domain.EventPurchase && retryCount == 0 {
		dedupID := strings.TrimSpace(in.EventID)
		if dedupID == "" {
			dedupID = strings.TrimSpace(in.OrderID)
		}
		val := purchaseValue(in)
		moneyStr := notify.FormatMoney(val, in.Currency)
		origin := notify.FriendlyGatewayOrigin(in.Gateway)
		pxLabel := util.FirstNonEmpty(pixel.Label, pixel.Name)
		buyer := strings.TrimSpace(strings.TrimSpace(in.FirstName + " " + in.LastName))
		if buyer == "" {
			buyer = util.FirstNonEmpty(in.Phone, in.Email)
		}

		parts := []string{"Origem: " + origin}
		if prod := strings.TrimSpace(in.ContentName); prod != "" {
			parts = append(parts, "Produto: "+prod)
		}
		parts = append(parts, "Pixel: "+pxLabel)
		if buyer != "" {
			parts = append(parts, "Cliente: "+buyer)
		}
		cdKey := ""
		if dedupID != "" {
			cdKey = fmt.Sprintf("sale:%d:%s", pixel.AccountID, dedupID)
		}
		deps.Notify.DispatchAsync(notify.Event{
			AccountID:   pixel.AccountID,
			Category:    domain.NotifSaleApproved,
			Title:       "Nova Venda: " + moneyStr,
			Body:        strings.Join(parts, " • ") + ".",
			ActionURL:   "/events",
			CooldownKey: cdKey,
			CooldownTTL: 24 * time.Hour,
		})
	}

	if hasErr && retryCount == 0 {
		for _, r := range collapseChannelResults(results) {
			if r.Err == nil {
				continue
			}
			chName := strings.ToUpper(r.Channel)
			switch r.Channel {
			case "meta":
				chName = "Meta CAPI"
			case "google_ads":
				chName = "Google Ads"
			case "tiktok":
				chName = "TikTok Events"
			case "ga4":
				chName = "Google Analytics 4"
			}
			pxLabel := util.FirstNonEmpty(pixel.Label, pixel.Name)
			statusTxt := ""
			if r.StatusCode > 0 {
				statusTxt = fmt.Sprintf(" (HTTP %d)", r.StatusCode)
			}
			deps.Notify.DispatchAsync(notify.Event{
				AccountID:   pixel.AccountID,
				Category:    domain.NotifEventDispatchError,
				Title:       "Erro no Envio de Evento (" + chName + ")",
				Body:        fmt.Sprintf("Falha ao enviar %s no Pixel %s para %s%s.", in.EventName, pxLabel, chName, statusTxt),
				ActionURL:   "/events",
				CooldownKey: fmt.Sprintf("ev_err:%d:%s:%s", pixel.AccountID, pixel.Name, r.Channel),
				CooldownTTL: 10 * time.Minute,
			})
			break
		}
	}
	_ = ctx
}

func notifyWhatsAppStateChange(deps Deps, inst *domain.WhatsAppInstance, nowConnected bool) {
	if deps.Notify == nil || inst == nil {
		return
	}
	prov := notify.FriendlyWhatsAppProvider(inst.Provider)
	title := "WhatsApp Desconectado"
	verb := "desconectou e precisa de reconexão"
	stateKey := "down"
	if nowConnected {
		title = "WhatsApp Conectado"
		verb = "está online e operando normalmente"
		stateKey = "up"
	}
	deps.Notify.DispatchAsync(notify.Event{
		AccountID:   inst.AccountID,
		Category:    domain.NotifWhatsAppConnection,
		Title:       title,
		Body:        fmt.Sprintf("A conexão \"%s\" (%s) %s.", inst.Name, prov, verb),
		ActionURL:   "/integracoes/whatsapp",
		CooldownKey: fmt.Sprintf("wa_conn:%d:%s", inst.ID, stateKey),
		CooldownTTL: 20 * time.Second,
	})
}

func maybeNotifyNewWhatsAppLead(
	deps Deps,
	inst *domain.WhatsAppInstance,
	existingChat *domain.WhatsAppChat,
	isFromMe bool,
	u *domain.TrackedUser,
	contactName, phone, originType string,
) {
	if deps.Notify == nil || inst == nil || existingChat != nil || isFromMe || u == nil {
		return
	}
	who := util.FirstNonEmpty(strings.TrimSpace(contactName), strings.TrimSpace(u.FirstName+" "+u.LastName), phone, "Novo contato")
	src := "WhatsApp"
	switch {
	case strings.HasPrefix(originType, "ctwa"):
		src = "Anúncio Click-to-WhatsApp"
	case strings.HasPrefix(originType, "bio_link"):
		src = "Link Rastreável na Bio"
	case strings.HasPrefix(originType, "website"):
		src = "Botão da Landing Page"
	}
	deps.Notify.DispatchAsync(notify.Event{
		AccountID:   inst.AccountID,
		Category:    domain.NotifLeadCreated,
		Title:       "Novo Lead no WhatsApp",
		Body:        fmt.Sprintf("%s iniciou conversa via %s • Conexão %s.", who, src, inst.Name),
		ActionURL:   "/crm",
		CooldownKey: fmt.Sprintf("lead:%d:%s", inst.AccountID, u.ExternalID),
		CooldownTTL: 15 * time.Minute,
	})
}

func enrichInboundWhatsAppContact(
	ctx context.Context,
	deps Deps,
	existingChat *domain.WhatsAppChat,
	parsed *ParsedWhatsAppMessage,
	trackedUser *domain.TrackedUser,
	phone string,
) (string, *string) {
	if phone != "" && trackedUser.Phone == "" {
		trackedUser.Phone = phone
		_ = deps.Users.Upsert(ctx, trackedUser)
	}
	if !parsed.IsFromMe {
		if rawEmail := util.ExtractEmail(parsed.Content); rawEmail != "" && trackedUser.Email != rawEmail {
			trackedUser.Email = rawEmail
			_ = deps.Users.UpdateEmail(ctx, trackedUser.ID, rawEmail)
			_, _ = deps.Users.ResolveAndLink(ctx, trackedUser.AccountID, trackedUser.ExternalID, trackedUser.Phone, rawEmail)
			slog.Info("enriched tracked user with email from received whatsapp message", "tracked_user_id", trackedUser.ID, "email", rawEmail)
		}
	}
	contactName := ""
	if !parsed.IsFromMe {
		contactName = parsed.PushName
	} else if existingChat != nil && existingChat.ContactName != nil && *existingChat.ContactName != "" {
		contactName = *existingChat.ContactName
	} else if trackedUser.FirstName != "" {
		contactName = strings.TrimSpace(trackedUser.FirstName + " " + trackedUser.LastName)
	}
	if contactName == "" {
		return "", nil
	}
	return contactName, &contactName
}
