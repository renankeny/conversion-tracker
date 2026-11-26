package whatsappai

import (
	"context"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/cache"
	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
	"github.com/renankeny/conversion-tracker/backend/internal/wshub"
)

type EventEnqueuer interface {
	EnqueueSendEvent(ctx context.Context, in domain.SendEventInput) error
}

// resolveAIQualificationEvent determina o evento de conversão para a qualificação pela IA:
// 1. Se a coluna de destino do CRM tem trigger próprio (ex: Schedule, AddToCart, QualifiedLead), respeita-o.
// 2. Caso contrário, em leads CTWA (WhatsApp) dispara QualifiedLead (distinto do LeadSubmitted inicial),
//    e em leads web/LP dispara Lead.
func resolveAIQualificationEvent(qualStage *domain.CRMStage, isCTWA bool) string {
	if qualStage != nil {
		if ev := strings.TrimSpace(domain.EffectiveStageTrigger(qualStage, isCTWA)); ev != "" && !strings.EqualFold(ev, domain.EventPurchase) {
			return ev
		}
	}
	if isCTWA {
		return domain.EventQualifiedLead
	}
	return domain.EventLead
}

// resolveMetaDefaultPageID respeita estritamente o pixel.MetaAccountID explícito.
func resolveMetaDefaultPageID(ctx context.Context, metaRepo *repo.MetaAccountRepo, pixel *domain.Pixel) string {
	if metaRepo == nil || pixel == nil || pixel.MetaAccountID == nil || *pixel.MetaAccountID <= 0 {
		return ""
	}
	if acc, err := metaRepo.FindByID(ctx, *pixel.MetaAccountID); err == nil && acc != nil && acc.AccountID == pixel.AccountID && acc.Enabled && acc.IsConnected() {
		return acc.DefaultPageID
	}
	return ""
}

func buildAISendEventInput(
	ctx context.Context,
	usersRepo *repo.UserRepo,
	metaRepo *repo.MetaAccountRepo,
	pixel *domain.Pixel,
	stage *domain.CRMStage,
	u *domain.TrackedUser,
	cand repo.AIChatCandidate,
	eventName string,
	eventID string,
	evTime time.Time,
	value *float64,
	currency string,
) domain.SendEventInput {
	if usersRepo != nil && u != nil {
		usersRepo.EnrichGroupFields(ctx, u)
	}
	ctwaClid := util.FirstNonEmpty(cand.CtwaClid, u.CtwaClid)
	pageID := cand.PageID
	if ctwaClid != "" && pageID == "" {
		pageID = resolveMetaDefaultPageID(ctx, metaRepo, pixel)
	}

	sourceURL := util.FirstNonEmpty(u.URL, u.FirstURL)
	actionSource := "website"
	msgChannel := ""
	if ctwaClid != "" {
		if pageID != "" {
			actionSource = "business_messaging"
			msgChannel = "whatsapp"
			sourceURL = ""
		} else {
			actionSource = "system_generated"
			ctwaClid = ""
		}
	} else if sourceURL == "" {
		actionSource = "system_generated"
	}

	if actionSource == "business_messaging" {
		if eventName == domain.EventLead {
			eventName = domain.EventLeadSubmitted
		}
	} else if eventName == domain.EventLeadSubmitted || eventName == domain.EventQualifiedLead {
		eventName = domain.EventLead
	}

	ts := time.Now().Unix()
	if !evTime.IsZero() {
		ts = util.ClampEventTime(evTime.Unix(), time.Now())
	}

	var overrideGadsAcc *int64
	var overrideGadsAct string
	if stage != nil {
		overrideGadsAcc = stage.GoogleAdsAccountID
		overrideGadsAct = strings.TrimSpace(stage.GoogleAdsActionID)
	}

	return domain.SendEventInput{
		PixelName:             pixel.Name,
		EventName:             eventName,
		EventID:               eventID,
		EventTime:             ts,
		SourceURL:             sourceURL,
		Gateway:               "whatsapp_ai",
		ActionSource:          actionSource,
		MessagingChannel:      msgChannel,
		CtwaClid:              ctwaClid,
		PageID:                pageID,
		ExternalID:            u.ExternalID,
		IDWpp:                 u.IDWpp,
		ClientIP:              u.ClientIP,
		ClientUserAgent:       u.ClientUserAgent,
		Email:                 u.Email,
		Phone:                 util.FirstNonEmpty(cand.Phone, u.Phone),
		FirstName:             u.FirstName,
		LastName:              u.LastName,
		Country:               u.Country,
		State:                 u.State,
		City:                  u.City,
		PostalCode:            u.PostalCode,
		FBP:                   u.FBP,
		FBC:                   u.FBC,
		GCLID:                 u.GCLID,
		WBraid:                u.WBraid,
		GBraid:                u.GBraid,
		TTClid:                u.TTClid,
		TTP:                   u.TTP,
		GA4ClientID:           u.GA4ClientID,
		GA4SessionID:          u.GA4SessionID,
		Value:                 value,
		Currency:              domain.NormalizeWhatsAppCurrency(util.FirstNonEmpty(currency, cand.Currency)),
		OverrideGadsAccountID: overrideGadsAcc,
		OverrideGadsActionID:  overrideGadsAct,
		CRMSynced:             true,
	}
}

func resolveValidMsgID(msgID, fallbackID int64, msgByID map[int64]*domain.WhatsAppMessage) int64 {
	if msgID > 0 {
		if _, ok := msgByID[msgID]; ok {
			return msgID
		}
	}
	return fallbackID
}

func (a *Analyzer) publishUpdates(ctx context.Context, accountID int64, ev *domain.WhatsAppAIEvent, crmChanged bool) {
	if a.Redis == nil {
		return
	}
	if crmChanged || ev.DispatchedEventID != "" {
		cache.BumpDashboardVersion(ctx, a.Redis, accountID)
	}
	wshub.PublishCustom(ctx, a.Redis, accountID, "whatsapp_ai_event", map[string]any{
		"chat_id": ev.ChatID, "crm_lead_id": ev.CRMLeadID, "tracked_user_id": ev.TrackedUserID, "ai_event": ev,
	})
	if crmChanged {
		wshub.PublishCustom(ctx, a.Redis, accountID, "crm_updated", map[string]any{
			"action": "whatsapp_ai_synced", "lead_id": ev.CRMLeadID, "event": ev.EventType,
		})
	}
}
