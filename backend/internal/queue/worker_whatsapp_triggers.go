package queue

import (
	"context"
	"regexp"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
	"github.com/renankeny/conversion-tracker/backend/internal/wshub"
)

func evaluateAttendantTriggers(
	ctx context.Context,
	deps Deps,
	inst *domain.WhatsAppInstance,
	u *domain.TrackedUser,
	chatID int64,
	phone, waMessageID, content string,
) {
	if strings.TrimSpace(content) == "" || deps.Enqueuer == nil || u == nil {
		return
	}

	matchedPurchase, isPurchase := matchPhraseTriggerWithTerm(inst.PurchaseTrigger, content)
	matchedLead, isLead := matchPhraseTriggerWithTerm(inst.LeadTrigger, content)
	matchedCustom, isCustom := "", false
	if strings.TrimSpace(inst.CustomEventName) != "" {
		matchedCustom, isCustom = matchPhraseTriggerWithTerm(inst.CustomEventTrigger, content)
	}
	if !isPurchase && !isLead && !isCustom {
		return
	}

	pixel, err := deps.Pixels.FindByID(ctx, inst.PixelID)
	if err != nil || pixel == nil || !pixel.Enabled {
		return
	}

	chat, _ := deps.WhatsApp.GetChatByID(ctx, inst.AccountID, chatID)
	ctwaClid := u.CtwaClid
	pageID := ""
	if chat != nil {
		ctwaClid = util.FirstNonEmpty(chat.CtwaClid, ctwaClid)
		pageID = chat.PageID
	}

	actionSource := "website"
	msgChannel := ""
	if ctwaClid != "" {
		actionSource = "business_messaging"
		msgChannel = "whatsapp"
	}

	type triggerMatch struct {
		eventName string
		phrase    string
		allowBare bool
	}
	var matches []triggerMatch
	if isPurchase {
		matches = append(matches, triggerMatch{eventName: domain.EventPurchase, phrase: matchedPurchase, allowBare: true})
	}
	if isLead {
		leadEv := domain.EventLead
		if actionSource == "business_messaging" {
			leadEv = domain.EventLeadSubmitted
		}
		matches = append(matches, triggerMatch{eventName: leadEv, phrase: matchedLead, allowBare: false})
	}
	if isCustom {
		matches = append(matches, triggerMatch{eventName: strings.TrimSpace(inst.CustomEventName), phrase: matchedCustom, allowBare: true})
	}

	for _, tm := range matches {
		eventID := waMessageID + ":" + tm.eventName
		if exists, _ := deps.SentEvents.ExistsSent(ctx, eventID, pixel.Name); exists {
			continue
		}

		var valPtr *float64
		if v, ok := util.ExtractMoneyFromMessage(content, tm.phrase, tm.allowBare); ok && v > 0 {
			valPtr = &v
		}

		if deps.CRM != nil && u.ID > 0 {
			if leadID, synced := deps.CRM.SyncLeadFromEventTrigger(ctx, inst.AccountID, u.ID, tm.eventName, valPtr); synced && deps.Redis != nil {
				wshub.PublishCustom(ctx, deps.Redis, inst.AccountID, "crm_updated", map[string]any{
					"action":  "whatsapp_trigger_synced",
					"lead_id": leadID,
					"event":   tm.eventName,
				})
			}
		}

		sourceURL := util.FirstNonEmpty(u.URL, u.FirstURL)
		if actionSource == "business_messaging" {
			sourceURL = ""
		}

		_ = deps.Enqueuer.EnqueueSendEvent(ctx, domain.SendEventInput{
			PixelName:        pixel.Name,
			EventName:        tm.eventName,
			EventID:          eventID,
			EventTime:        time.Now().Unix(),
			SourceURL:        sourceURL,
			Gateway:          "whatsapp",
			ActionSource:     actionSource,
			MessagingChannel: msgChannel,
			CtwaClid:         ctwaClid,
			PageID:           pageID,
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
			GA4ClientID:      u.GA4ClientID,
			GA4SessionID:     u.GA4SessionID,
			Value:            valPtr,
			Currency:         "BRL",
		})
	}
}

// matchPhraseTriggerWithTerm suporta múltiplos termos separados por vírgula ou ponto-e-vírgula,
// ignorando maiúsculas/minúsculas e acentuação (ex: "pagamento confirmado, pix recebido").
func matchPhraseTriggerWithTerm(trigger, text string) (string, bool) {
	trigger = strings.TrimSpace(trigger)
	text = strings.TrimSpace(text)
	if trigger == "" || text == "" {
		return "", false
	}

	normText := foldAccentsLower(text)
	parts := strings.FieldsFunc(trigger, func(r rune) bool { return r == ',' || r == ';' })
	if len(parts) == 0 {
		parts = []string{trigger}
	}

	for _, rawPart := range parts {
		part := strings.TrimSpace(rawPart)
		if part == "" {
			continue
		}
		normPart := foldAccentsLower(part)
		if normPart != "" && strings.Contains(normText, normPart) {
			return part, true
		}
		if rx, err := regexp.Compile("(?i)" + part); err == nil && rx.MatchString(text) {
			return part, true
		}
	}
	return "", false
}

func foldAccentsLower(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	res, _, err := transform.String(t, strings.ToLower(s))
	if err != nil {
		return strings.ToLower(s)
	}
	return res
}

