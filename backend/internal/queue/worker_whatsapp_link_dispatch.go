package queue

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

const recentLinkClickMatchWindow = 2 * time.Minute

func isChannelAllowed(allowed []string, channel string) bool {
	if len(allowed) == 0 {
		return true
	}
	for _, c := range allowed {
		if strings.EqualFold(strings.TrimSpace(c), channel) {
			return true
		}
	}
	return false
}

// resolveAssistedTrackableLinkLead trata dois cenários de Link Rastreável:
// 1) Protocolo explícito ("Ref: ABC...") -> localiza o clique exato em whatsapp_link_clicks.
// 2) Lead apagou a frase pré-preenchida e mandou "Oi" -> verifica se existe exatamente 1 clique
//    humano não consumido nos últimos 2 minutos para a mesma instância WhatsApp.
func resolveAssistedTrackableLinkLead(
	ctx context.Context,
	deps Deps,
	inst *domain.WhatsAppInstance,
	existingChat *domain.WhatsAppChat,
	parsed *ParsedWhatsAppMessage,
	protocolCode, phone, lidDigits string,
) (*domain.TrackedUser, string, *domain.TrackableLinkClick, *int64) {
	if deps.TrackableLinks == nil {
		return nil, "", nil, nil
	}

	var click *domain.TrackableLinkClick
	if protocolCode != "" {
		if c, err := deps.TrackableLinks.FindClickByProtocol(ctx, inst.AccountID, protocolCode); err == nil && c != nil {
			click = c
		}
	} else if !parsed.IsFromMe && existingChat == nil && parsed.CtwaClid == "" && parsed.SourceID == "" {
		click = deps.TrackableLinks.FindUnambiguousRecentClickForInstance(ctx, inst.ID, recentLinkClickMatchWindow)
	}

	if click == nil {
		return nil, "", nil, nil
	}

	linkIDCopy := click.LinkID
	var u *domain.TrackedUser
	if click.TrackedUserID != nil && *click.TrackedUserID > 0 && deps.Users != nil {
		u, _ = deps.Users.FindByID(ctx, *click.TrackedUserID)
	}
	if u == nil && click.ProtocolCode != "" && deps.Users != nil {
		u, _ = deps.Users.FindByProtocolByAccount(ctx, click.ProtocolCode, inst.AccountID)
	}
	if u == nil {
		return nil, "", click, &linkIDCopy
	}

	first, last := splitFullName(parsed.PushName)
	hydrateWhatsAppUserFields(u, phone, lidDigits, first, last, parsed)
	u.TrackableLinkID = &linkIDCopy
	_ = deps.Users.Upsert(ctx, u)
	if phone != "" {
		if stitched, err := deps.Users.ResolveAndLink(ctx, inst.AccountID, u.ExternalID, phone, u.Email); err == nil && stitched != nil {
			u = stitched
			u.TrackableLinkID = &linkIDCopy
		}
	}
	return u, ClassifyPortalOrigin(parsed.CtwaClid, parsed.SourceID, u), click, &linkIDCopy
}

func resolveEffectiveTrackableLinkID(
	click *domain.TrackableLinkClick,
	explicitLinkID *int64,
	existingChat *domain.WhatsAppChat,
	u *domain.TrackedUser,
) *int64 {
	if explicitLinkID != nil && *explicitLinkID > 0 {
		return explicitLinkID
	}
	if click != nil && click.LinkID > 0 {
		id := click.LinkID
		return &id
	}
	if existingChat != nil && existingChat.TrackableLinkID != nil && *existingChat.TrackableLinkID > 0 {
		return existingChat.TrackableLinkID
	}
	if u != nil && u.TrackableLinkID != nil && *u.TrackableLinkID > 0 {
		return u.TrackableLinkID
	}
	return nil
}

// finalizeTrackableLinkFirstMessage vincula o chat ao clique e dispara (uma única vez)
// o evento de 1ª mensagem nos canais marcados pelo aluno (Meta, TikTok, Google Ads, GA4).
func finalizeTrackableLinkFirstMessage(
	ctx context.Context,
	deps Deps,
	inst *domain.WhatsAppInstance,
	click *domain.TrackableLinkClick,
	linkID *int64,
	u *domain.TrackedUser,
	chatID int64,
	phone string,
	isFromMe bool,
) {
	if deps.TrackableLinks == nil {
		return
	}
	if click != nil && click.ID > 0 && chatID > 0 {
		_ = deps.TrackableLinks.AttachChatToClick(ctx, click.ID, chatID)
	}
	if isFromMe || linkID == nil || *linkID <= 0 || u == nil || deps.Enqueuer == nil {
		return
	}

	link, err := deps.TrackableLinks.GetByID(ctx, *linkID)
	if err != nil || link == nil || !link.Enabled || !link.SendEventsEnabled {
		return
	}
	if link.EventTriggerMode != domain.LinkTriggerOnFirstMessage {
		return
	}
	allowed := link.AllowedChannels()
	if len(allowed) == 0 {
		return
	}

	eventID := fmt.Sprintf("walnk_msg_%d", chatID)
	if deps.SentEvents != nil {
		if exists, _ := deps.SentEvents.ExistsSent(ctx, inst.AccountID, eventID, link.PixelName, ""); exists {
			return
		}
	}
	if click != nil && click.ID > 0 {
		if !deps.TrackableLinks.TryClaimMessageEventDispatch(ctx, click.ID) {
			return
		}
	}

	if deps.Users != nil {
		deps.Users.EnrichGroupFields(ctx, u)
	}
	linkBaseURL := util.FirstNonEmpty(deps.AppURL, deps.PublicAPIURL)
	sourceURL := util.FirstNonEmpty(u.URL, u.FirstURL, link.BuildPublicURL(linkBaseURL))

	if err := deps.Enqueuer.EnqueueSendEvent(ctx, domain.SendEventInput{
		PixelName:        link.PixelName,
		EventName:        link.EventName,
		EventID:          eventID,
		EventTime:        time.Now().Unix(),
		SourceURL:        sourceURL,
		Gateway:          "whatsapp_link",
		ActionSource:     "website",
		ExternalID:       u.ExternalID,
		IDWpp:            u.IDWpp,
		ClientIP:         u.ClientIP,
		ClientUserAgent:  u.ClientUserAgent,
		Email:            u.Email,
		Phone:            util.FirstNonEmpty(phone, u.Phone),
		FirstName:        u.FirstName,
		LastName:         u.LastName,
		Country:          u.Country,
		State:            u.State,
		City:             u.City,
		PostalCode:       u.PostalCode,
		FBP:              u.FBP,
		FBC:              u.FBC,
		GCLID:            u.GCLID,
		WBraid:           u.WBraid,
		GBraid:           u.GBraid,
		TTClid:           u.TTClid,
		TTP:              u.TTP,
		Referrer:         u.Referrer,
		GA4ClientID:      u.GA4ClientID,
		GA4SessionID:     u.GA4SessionID,
		AllowedChannels:  allowed,
		CRMSynced:        true,
	}); err != nil {
		slog.Error("enqueue trackable link first message event failed",
			"pixel_name", link.PixelName,
			"event_id", eventID,
			"error", err,
		)
	}
}
