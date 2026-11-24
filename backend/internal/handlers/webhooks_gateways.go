package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/cache"
	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
	"github.com/renankeny/conversion-tracker/backend/internal/wshub"
)

func (h *Webhooks) enqueueGenericPurchase(ctx context.Context, in purchaseInput, rawInput *domain.WebhookPurchaseInput) (map[string]any, int) {
	in.EventName = orDefaultEvent(in.EventName)
	in.Currency = domain.NormalizeWhatsAppCurrency(orDefaultCurrency(in.Currency))
	in.Phone = util.NormalizePhone(in.Phone)
	in.Email = util.NormalizeEmail(in.Email)

	resolvedQueryPixel, status, reason := h.resolveWebhookAccount(ctx, &in)
	if status != 0 {
		h.logWebhook(ctx, in, rawInput, in.AccountID, "", in.ExternalID, "", "failed", reason)
		return map[string]any{"received": true, "error": reason}, status
	}

	user, err := h.findUser(ctx, in)
	if err != nil {
		fallbackPixel := h.selectFallbackPixel(ctx, resolvedQueryPixel, in.AccountID)
		if fallbackPixel == nil {
			slog.Warn("webhook user not found", "gateway", in.GatewayName, "external_id", in.ExternalID, "phone", in.Phone, "email", in.Email, "account_id", in.AccountID)
			h.logWebhook(ctx, in, rawInput, in.AccountID, "", in.ExternalID, "", "failed", "no tracked user")
			return map[string]any{"received": true, "matched_user": false, "reason": "no tracked user"}, http.StatusOK
		}
		targetAccountID := fallbackPixel.AccountID
		if !h.verifyDeferredWebhookAuth(ctx, targetAccountID) {
			return map[string]any{"reason": "invalid platform token"}, http.StatusUnauthorized
		}
		syntheticExternalID := util.FirstNonEmpty(in.ExternalID, util.GenerateID10())
		newUser := &domain.TrackedUser{
			ExternalID: syntheticExternalID, IDWpp: util.GenerateID10(), PixelName: fallbackPixel.Name,
			AccountID: targetAccountID, FirstName: in.FirstName, LastName: in.LastName,
			Email: in.Email, Phone: in.Phone, FBC: in.FBC, FBP: in.FBP,
		}
		applyWebhookTrackingEnrichment(newUser, in)
		if cerr := h.users.Create(ctx, newUser); cerr != nil {
			slog.Error("create fallback user from webhook", "err", cerr, "gateway", in.GatewayName)
		}
		user = newUser
	} else if !h.verifyDeferredWebhookAuth(ctx, user.AccountID) {
		return map[string]any{"reason": "invalid platform token"}, http.StatusUnauthorized
	}

	user.FirstName = util.FirstNonEmpty(in.FirstName, user.FirstName)
	user.LastName = util.FirstNonEmpty(in.LastName, user.LastName)
	user.Email = util.FirstNonEmpty(in.Email, user.Email)
	user.Phone = util.FirstNonEmpty(in.Phone, user.Phone)
	user.FBC = util.FirstNonEmpty(user.FBC, in.FBC)
	user.FBP = util.FirstNonEmpty(user.FBP, in.FBP)
	applyWebhookTrackingEnrichment(user, in)
	if user.PixelName == "" && resolvedQueryPixel != nil {
		user.PixelName = resolvedQueryPixel.Name
	}
	if err := h.users.Upsert(ctx, user); err != nil {
		slog.Error("update user from webhook", "err", err, "gateway", in.GatewayName)
	}
	if linked, lerr := h.users.ResolveAndLink(ctx, user.AccountID, user.ExternalID, user.Phone, user.Email); lerr == nil && linked != nil {
		user = linked
	}
	h.users.EnrichGroupFields(ctx, user)

	var pixel *domain.Pixel
	if resolvedQueryPixel != nil {
		pixel = resolvedQueryPixel
	} else {
		pixel, err = h.pixels.FindByName(ctx, user.PixelName)
		if err != nil {
			slog.Error("webhook pixel not found", "err", err, "pixel_name", user.PixelName, "gateway", in.GatewayName)
			return map[string]any{"received": true, "error": "pixel not found: " + user.PixelName}, http.StatusInternalServerError
		}
	}
	if !pixel.Enabled || pixel.AccountID != user.AccountID {
		return map[string]any{"received": true, "matched_user": true, "queued": false, "reason": "pixel disabled"}, http.StatusOK
	}

	val := in.Value
	if in.EventName == domain.EventPurchase && val <= 0 && len(pixel.TrackerConfig.ProductValue) > 0 && pixel.TrackerConfig.ProductValue[0] > 0 {
		val = pixel.TrackerConfig.ProductValue[0]
	}
	if in.EventName == domain.EventPurchase && val <= 0 {
		h.logWebhook(ctx, in, rawInput, pixel.AccountID, pixel.Name, user.ExternalID, "", "failed", "valor da venda zero ou ausente")
		return map[string]any{"received": true, "matched_user": true, "queued": false, "reason": "invalid purchase value"}, http.StatusOK
	}

	fbc := util.FirstNonEmpty(user.FBC, in.FBC)
	fbp := util.FirstNonEmpty(user.FBP, in.FBP)
	rawURL := util.FirstNonEmpty(user.URL, user.FirstURL, in.SourceURL)
	sourceURL := ""
	if rawURL != "" {
		if h.urlMapper != nil {
			sourceURL = h.urlMapper.ApplyForAccount(pixel.AccountID, rawURL)
		} else {
			sourceURL = rawURL
		}
	}

	itemType := in.OfferType
	if itemType == "" {
		if in.IsOrderBump {
			itemType = "order_bump"
		} else if in.IsUpsell {
			itemType = "upsell"
		}
	}
	eventID := in.ReplayEventID
	if eventID == "" {
		eventID = webhookEventIDForEvent(in.EventName, in.OrderID, itemType, in.ProductID)
	}
	dedupItemKey := util.FirstNonEmpty(itemType, "main") + ":" + in.ProductID
	if !in.IsReplay && in.OrderID != "" {
		if claimed, ok := cache.ClaimWebhookDedup(ctx, h.rdb, pixel.AccountID, in.GatewayName, in.OrderID, in.EventName, dedupItemKey); ok && !claimed {
			return map[string]any{"received": true, "matched_user": true, "deduplicated": true, "event_id": eventID}, http.StatusOK
		}
	}

	if !in.IsReplay && in.EventName == domain.EventPurchase && itemType != "order_bump" && itemType != "upsell" {
		now := time.Now()
		dupInFlight := cache.HasInFlightMatchingPurchase(
			ctx, h.rdb, pixel.AccountID, pixel.Name, user.GroupID, user.ID,
			user.ExternalID, user.Phone, user.Email, in.OrderID, in.OrderID == "", val, now,
		)
		dupDB := false
		if !dupInFlight && h.sentEvents != nil {
			dupDB, _ = h.sentEvents.HasDuplicateExternalPurchase(ctx, pixel.AccountID, user.ID, user.ExternalID, pixel.Name, in.OrderID, val, now)
		}
		if dupInFlight || dupDB {
			if h.crmRepo != nil && user.ID > 0 {
				if leadID, synced := h.crmRepo.SyncOrCreateLeadOnPurchase(ctx, pixel.AccountID, user.ID, val, false); synced && h.rdb != nil {
					wshub.PublishCustom(ctx, h.rdb, pixel.AccountID, "crm_updated", map[string]any{"action": "webhook_purchase_synced", "lead_id": leadID, "event": domain.EventPurchase})
				}
			}
			h.logWebhook(ctx, in, rawInput, pixel.AccountID, pixel.Name, user.ExternalID, eventID, "deduplicated", "venda já enviada anteriormente por outro canal/CRM")
			return map[string]any{"received": true, "matched_user": true, "deduplicated": true, "event_id": eventID}, http.StatusOK
		}
	}

	actionSource, msgChannel, ctwaClid, pageID := h.ctwaAttribution(ctx, pixel.AccountID, pixel, user, fbc, sourceURL)
	if actionSource == "business_messaging" {
		sourceURL = ""
	}
	evTime := time.Now().Unix()
	if in.IsReplay && in.ReplayEventTime > 0 {
		evTime = util.ClampEventTime(in.ReplayEventTime, time.Now())
	}

	customerGroupKey := ""
	if user.GroupID != nil && *user.GroupID > 0 {
		customerGroupKey = "g:" + strconv.FormatInt(*user.GroupID, 10)
	} else if user.ID > 0 {
		customerGroupKey = "u:" + strconv.FormatInt(user.ID, 10)
	}

	send := domain.SendEventInput{
		PixelName: pixel.Name, EventName: in.EventName, EventID: eventID, EventTime: evTime,
		SourceURL: sourceURL, ContentID: util.FirstNonEmpty(in.ProductID, pixel.Name), ContentName: in.ProductName,
		ContentType: "product", Value: &val, Currency: in.Currency, OrderID: in.OrderID,
		CustomerGroupKey: customerGroupKey,
		ItemType: itemType, Gateway: in.GatewayName, ActionSource: actionSource, MessagingChannel: msgChannel,
		CtwaClid: ctwaClid, PageID: pageID, ExternalID: user.ExternalID, ClientIP: user.ClientIP,
		ClientUserAgent: user.ClientUserAgent, FBP: fbp, FBC: fbc, Email: user.Email, Phone: user.Phone,
		FirstName: user.FirstName, LastName: user.LastName, Country: user.Country, State: user.State,
		City: user.City, PostalCode: user.PostalCode, TTClid: user.TTClid, TTP: user.TTP,
		GCLID: user.GCLID, WBraid: user.WBraid, GBraid: user.GBraid, GA4ClientID: user.GA4ClientID,
		GA4SessionID: user.GA4SessionID, IsReplay: in.IsReplay, CRMSynced: true,
	}

	aggregated := false
	if !in.IsReplay && pixel.AggregationEnabled && in.EventName == domain.EventPurchase && h.rdb != nil {
		var aggErr error
		aggregated, aggErr = h.queue.AggregateOrEnqueue(ctx, pixel, send, h.rdb)
		if aggErr != nil {
			cache.ReleaseWebhookDedup(ctx, h.rdb, pixel.AccountID, in.GatewayName, in.OrderID, in.EventName, dedupItemKey)
			slog.Error("aggregate or enqueue webhook event", "err", aggErr, "gateway", in.GatewayName, "event_id", eventID)
			h.logWebhook(ctx, in, rawInput, pixel.AccountID, pixel.Name, user.ExternalID, "", "failed", "queue error: "+aggErr.Error())
			return map[string]any{"received": true, "error": "queue error"}, http.StatusInternalServerError
		}
	} else {
		if err := h.queue.EnqueueSendEvent(ctx, send); err != nil {
			cache.ReleaseWebhookDedup(ctx, h.rdb, pixel.AccountID, in.GatewayName, in.OrderID, in.EventName, dedupItemKey)
			slog.Error("enqueue webhook event", "err", err, "gateway", in.GatewayName, "event_id", eventID)
			h.logWebhook(ctx, in, rawInput, pixel.AccountID, pixel.Name, user.ExternalID, "", "failed", "queue error: "+err.Error())
			return map[string]any{"received": true, "error": "queue error"}, http.StatusInternalServerError
		}
	}
	if !in.IsReplay && in.EventName == domain.EventPurchase {
		cache.RecordInFlightPurchase(ctx, h.rdb, pixel.AccountID, pixel.Name, user.GroupID, user.ID, user.ExternalID, user.Phone, user.Email, cache.InFlightPurchase{
			EventID: eventID, OrderID: in.OrderID, IsNonOrder: in.OrderID == "", Value: val,
		})
	}

	if h.crmRepo != nil && user.ID > 0 && in.EventName == domain.EventPurchase {
		if leadID, synced := h.crmRepo.SyncOrCreateLeadOnPurchase(ctx, pixel.AccountID, user.ID, val, !in.IsReplay); synced && h.rdb != nil {
			wshub.PublishCustom(ctx, h.rdb, pixel.AccountID, "crm_updated", map[string]any{"action": "webhook_purchase_synced", "lead_id": leadID, "event": domain.EventPurchase})
		}
	}

	if aggregated {
		h.logWebhook(ctx, in, rawInput, pixel.AccountID, pixel.Name, user.ExternalID, eventID, "buffered", "compra em buffer na janela de "+strconv.Itoa(pixel.AggregationWindowMinutes)+"min")
	} else {
		h.logWebhook(ctx, in, rawInput, pixel.AccountID, pixel.Name, user.ExternalID, eventID, "queued", "")
	}
	return map[string]any{"received": true, "matched_user": true, "event_id": eventID, "event_name": in.EventName, "aggregated": aggregated}, http.StatusAccepted
}
