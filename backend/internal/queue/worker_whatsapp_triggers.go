package queue

import (
	"context"
	"log/slog"
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
	isFromMe bool,
) error {
	if !isFromMe || strings.TrimSpace(content) == "" || deps.Enqueuer == nil || u == nil {
		return nil
	}

	matchedPurchase, isPurchase := matchPhraseTriggerWithTerm(inst.PurchaseTrigger, content)
	matchedLead, isLead := matchPhraseTriggerWithTerm(inst.LeadTrigger, content)
	matchedCustom, isCustom := "", false
	if strings.TrimSpace(inst.CustomEventName) != "" {
		matchedCustom, isCustom = matchPhraseTriggerWithTerm(inst.CustomEventTrigger, content)
	}
	if !isPurchase && !isLead && !isCustom {
		return nil
	}

	pixel, err := deps.Pixels.FindByID(ctx, inst.PixelID)
	if err != nil || pixel == nil || !pixel.Enabled {
		return nil
	}
	if deps.Users != nil {
		deps.Users.EnrichGroupFields(ctx, u)
	}

	chat, _ := deps.WhatsApp.GetChatByID(ctx, inst.AccountID, chatID)
	ctwaClid := u.CtwaClid
	pageID := ""
	if chat != nil {
		ctwaClid = util.FirstNonEmpty(chat.CtwaClid, ctwaClid)
		pageID = chat.PageID
	}
	if ctwaClid != "" && pageID == "" && deps.WhatsApp != nil {
		pageID = deps.WhatsApp.FindPageIDByCtwaClid(ctx, inst.AccountID, ctwaClid)
		if pageID == "" && u.ID > 0 {
			pageID = deps.WhatsApp.FindPageIDByTrackedUserID(ctx, inst.AccountID, u.ID)
		}
	}
	if ctwaClid != "" && pageID == "" {
		if oauthAcc, _ := resolvePixelMetaOAuth(ctx, deps, pixel); oauthAcc != nil && oauthAcc.DefaultPageID != "" {
			pageID = oauthAcc.DefaultPageID
		}
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

	var firstErr error
	for _, tm := range matches {
		eventID := waMessageID + ":" + tm.eventName
		if deps.SentEvents != nil {
			if exists, _ := deps.SentEvents.ExistsSent(ctx, pixel.AccountID, eventID, pixel.Name, ""); exists {
				continue
			}
			if tm.eventName == domain.EventLead || tm.eventName == domain.EventLeadSubmitted || tm.eventName == domain.EventQualifiedLead {
				if already, _ := deps.SentEvents.HasLeadConversionSent(ctx, pixel.AccountID, u.ID, u.ExternalID, pixel.Name); already {
					continue
				}
			}
		}

		var valPtr *float64
		if v, ok := util.ExtractMoneyFromMessage(content, tm.phrase, tm.allowBare); ok && v > 0 {
			valPtr = &v
		}
		if tm.eventName == domain.EventPurchase && valPtr == nil {
			if deps.CRM != nil && u.ID > 0 {
				if crmVal := deps.CRM.GetLeadValueByTrackedUser(ctx, inst.AccountID, u.ID); crmVal > 0 {
					valPtr = &crmVal
				}
			}
			if valPtr == nil && len(pixel.TrackerConfig.ProductValue) > 0 && pixel.TrackerConfig.ProductValue[0] > 0 {
				pv := pixel.TrackerConfig.ProductValue[0]
				valPtr = &pv
			}
		}

		if tm.eventName == domain.EventPurchase {
			if valPtr == nil || *valPtr <= 0 {
				slog.Warn("skipping whatsapp trigger Purchase without positive value", "pixel_name", pixel.Name, "chat_id", chatID)
				continue
			}
			if deps.SentEvents != nil {
				if dup, _ := deps.SentEvents.HasMatchingPurchaseSent(ctx, pixel.AccountID, u.ID, u.ExternalID, pixel.Name, *valPtr, time.Now()); dup {
					if deps.CRM != nil && u.ID > 0 {
						if leadID, synced := deps.CRM.SyncOrCreateLeadOnPurchase(ctx, inst.AccountID, u.ID, *valPtr, false); synced && deps.Redis != nil {
							wshub.PublishCustom(ctx, deps.Redis, inst.AccountID, "crm_updated", map[string]any{"action": "whatsapp_trigger_synced", "lead_id": leadID, "event": tm.eventName})
						}
					}
					continue
				}
			}
			if deps.CRM != nil && u.ID > 0 {
				if leadID, synced := deps.CRM.SyncOrCreateLeadOnPurchase(ctx, inst.AccountID, u.ID, *valPtr, true); synced && deps.Redis != nil {
					wshub.PublishCustom(ctx, deps.Redis, inst.AccountID, "crm_updated", map[string]any{"action": "whatsapp_trigger_synced", "lead_id": leadID, "event": tm.eventName})
				}
			}
		} else if deps.CRM != nil && u.ID > 0 {
			if leadID, synced := deps.CRM.SyncLeadFromEventTrigger(ctx, inst.AccountID, u.ID, tm.eventName, valPtr); synced && deps.Redis != nil {
				wshub.PublishCustom(ctx, deps.Redis, inst.AccountID, "crm_updated", map[string]any{"action": "whatsapp_trigger_synced", "lead_id": leadID, "event": tm.eventName})
			}
		}

		if err := deps.Enqueuer.EnqueueSendEvent(ctx, domain.SendEventInput{
			PixelName: pixel.Name, EventName: tm.eventName, EventID: eventID, EventTime: time.Now().Unix(),
			SourceURL: sourceURL, Gateway: "whatsapp_trigger", ActionSource: actionSource, MessagingChannel: msgChannel,
			CtwaClid: ctwaClid, PageID: pageID, ExternalID: u.ExternalID, IDWpp: u.IDWpp,
			ClientIP: u.ClientIP, ClientUserAgent: u.ClientUserAgent, Email: u.Email,
			Phone: util.FirstNonEmpty(phone, u.Phone), FirstName: u.FirstName, LastName: u.LastName,
			Country: u.Country, State: u.State, City: u.City, PostalCode: u.PostalCode,
			FBP: u.FBP, FBC: u.FBC, GCLID: u.GCLID, WBraid: u.WBraid, GBraid: u.GBraid,
			TTClid: u.TTClid, TTP: u.TTP, GA4ClientID: u.GA4ClientID, GA4SessionID: u.GA4SessionID,
			Value: valPtr, Currency: domain.NormalizeWhatsAppCurrency(inst.Currency), CRMSynced: true,
		}); err != nil {
			slog.Error("enqueue whatsapp trigger event failed", "pixel_name", pixel.Name, "event_id", eventID, "error", err)
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

// matchPhraseTriggerWithTerm suporta múltiplos termos separados por vírgula ou ponto-e-vírgula,
// ignorando maiúsculas/minúsculas e acentuação e escapando metacaracteres de regex.
func matchPhraseTriggerWithTerm(trigger, text string) (string, bool) {
	trigger = strings.TrimSpace(trigger)
	text = strings.TrimSpace(text)
	if trigger == "" || text == "" {
		return "", false
	}

	normText := foldAccentsLower(text)
	parts := splitTriggerTerms(trigger)

	for _, rawPart := range parts {
		part := strings.TrimSpace(rawPart)
		if part == "" {
			continue
		}
		normPart := foldAccentsLower(part)
		if normPart != "" && strings.Contains(normText, normPart) {
			return part, true
		}
		if rx, err := regexp.Compile("(?i)" + regexp.QuoteMeta(part)); err == nil && rx.MatchString(text) {
			return part, true
		}
	}
	return "", false
}

func splitTriggerTerms(s string) []string {
	rs := []rune(s)
	var parts []string
	start := 0
	for i, r := range rs {
		isSep := r == ';'
		if r == ',' {
			decimalComma := i > 0 && i+1 < len(rs) && unicode.IsDigit(rs[i-1]) && unicode.IsDigit(rs[i+1])
			isSep = !decimalComma
		}
		if isSep {
			parts = append(parts, string(rs[start:i]))
			start = i + 1
		}
	}
	parts = append(parts, string(rs[start:]))
	return parts
}

func foldAccentsLower(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	res, _, err := transform.String(t, strings.ToLower(s))
	if err != nil {
		return strings.ToLower(s)
	}
	return res
}
