package queue

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/cache"
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
	if !isFromMe || strings.TrimSpace(content) == "" || deps.Enqueuer == nil || u == nil || inst == nil {
		return nil
	}

	type triggerMatch struct {
		eventName string
		phrase    string
		allowBare bool
	}
	var rawMatches []triggerMatch
	for _, rule := range inst.EffectiveEventTriggers() {
		evName := strings.TrimSpace(rule.EventName)
		if evName == "" || domain.IsInternalEvent(evName) {
			continue
		}
		if matchedPhrase, ok := matchPhraseTriggerWithTerm(rule.Trigger, content); ok {
			isLeadLike := strings.EqualFold(evName, domain.EventLead) ||
				strings.EqualFold(evName, domain.EventLeadSubmitted) ||
				strings.EqualFold(evName, domain.EventQualifiedLead) ||
				strings.EqualFold(evName, domain.EventContact)
			rawMatches = append(rawMatches, triggerMatch{
				eventName: evName,
				phrase:    matchedPhrase,
				allowBare: !isLeadLike,
			})
		}
	}
	if len(rawMatches) == 0 {
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

	seenEvents := make(map[string]bool, len(rawMatches))
	var firstErr error

	for _, tm := range rawMatches {
		if actionSource == "business_messaging" && strings.EqualFold(tm.eventName, domain.EventLead) {
			tm.eventName = domain.EventLeadSubmitted
		}
		if strings.EqualFold(tm.eventName, domain.EventPurchase) {
			tm.eventName = domain.EventPurchase
		}
		evKey := strings.ToLower(tm.eventName)
		if seenEvents[evKey] {
			continue
		}
		seenEvents[evKey] = true

		eventID := waMessageID + ":" + tm.eventName
		if deps.SentEvents != nil {
			if exists, _ := deps.SentEvents.ExistsSent(ctx, pixel.AccountID, eventID, pixel.Name, ""); exists {
				continue
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
			if valPtr == nil && inst.AverageTicket > 0 {
				avg := inst.AverageTicket
				valPtr = &avg
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
			now := time.Now()
			dupInFlight := cache.HasInFlightMatchingPurchase(ctx, deps.Redis, pixel.AccountID, pixel.Name, u.GroupID, u.ID, u.ExternalID, util.FirstNonEmpty(phone, u.Phone), u.Email, "", true, *valPtr, now)
			dupDB := false
			if !dupInFlight && deps.SentEvents != nil {
				dupDB, _ = deps.SentEvents.HasMatchingPurchaseSent(ctx, pixel.AccountID, u.ID, u.ExternalID, pixel.Name, *valPtr, now)
			}
			if dupInFlight || dupDB {
				if deps.CRM != nil && u.ID > 0 {
					if leadID, synced := deps.CRM.SyncOrCreateLeadOnPurchase(ctx, inst.AccountID, u.ID, *valPtr, false); synced && deps.Redis != nil {
						wshub.PublishCustom(ctx, deps.Redis, inst.AccountID, "crm_updated", map[string]any{"action": "whatsapp_trigger_synced", "lead_id": leadID, "event": tm.eventName})
					}
				}
				continue
			}
			cache.RecordInFlightPurchase(ctx, deps.Redis, pixel.AccountID, pixel.Name, u.GroupID, u.ID, u.ExternalID, util.FirstNonEmpty(phone, u.Phone), u.Email, cache.InFlightPurchase{
				EventID: eventID, IsNonOrder: true, Value: *valPtr,
			})
			if deps.CRM != nil && u.ID > 0 {
				if leadID, synced := deps.CRM.SyncOrCreateLeadOnPurchase(ctx, inst.AccountID, u.ID, *valPtr, true); synced && deps.Redis != nil {
					wshub.PublishCustom(ctx, deps.Redis, inst.AccountID, "crm_updated", map[string]any{"action": "whatsapp_trigger_synced", "lead_id": leadID, "event": tm.eventName})
				}
			}
		} else {
			if deps.CRM != nil && u.ID > 0 {
				if leadID, synced := deps.CRM.SyncLeadFromEventTrigger(ctx, inst.AccountID, u.ID, tm.eventName, valPtr); synced && deps.Redis != nil {
					wshub.PublishCustom(ctx, deps.Redis, inst.AccountID, "crm_updated", map[string]any{"action": "whatsapp_trigger_synced", "lead_id": leadID, "event": tm.eventName})
				}
			}
			if deps.SentEvents != nil {
				if tm.eventName == domain.EventLead || tm.eventName == domain.EventLeadSubmitted || (tm.eventName == domain.EventQualifiedLead && actionSource != "business_messaging") {
					if already, _ := deps.SentEvents.HasLeadConversionSent(ctx, pixel.AccountID, u.ID, u.ExternalID, pixel.Name); already {
						continue
					}
				} else if tm.eventName == domain.EventQualifiedLead && actionSource == "business_messaging" {
					if already, _ := deps.SentEvents.ExistsRecentEvent(ctx, pixel.AccountID, pixel.Name, u.ExternalID, domain.EventQualifiedLead, 365*24*time.Hour); already {
						continue
					}
				}
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
