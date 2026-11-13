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
)

// enqueueGenericPurchase é o fluxo único compartilhado:
//  1. Encontra o tracked user (external_id → phone (com variantes) → email opcional).
//  2. Atualiza PII recebida do gateway no tracked_users.
//  3. Aplica URL black→white na URL gravada do user.
//  4. Resolve o pixel (`pixel_name` salvo no tracked user).
//  5. Enfileira o evento (Purchase ou AddPaymentInfo) na fila Meta CAPI.
func (h *Webhooks) enqueueGenericPurchase(ctx context.Context, in purchaseInput, rawInput *domain.WebhookPurchaseInput) (map[string]any, int) {
	in.EventName = orDefaultEvent(in.EventName)
	in.Currency = orDefaultCurrency(in.Currency)
	in.Phone = util.NormalizePhone(in.Phone)
	in.Email = util.NormalizeEmail(in.Email)

	resolvedQueryPixel, status, reason := h.resolveWebhookAccount(ctx, &in)
	if status != 0 {
		h.logWebhook(ctx, in, rawInput, in.AccountID, "", in.ExternalID, "", "failed", reason)
		return map[string]any{"received": true, "error": reason}, status
	}

	user, err := h.findUser(ctx, in)
	if err != nil {
		fallbackPixel := resolvedQueryPixel
		if fallbackPixel == nil && in.AccountID > 0 {
			if list, perr := h.pixels.ListByAccount(ctx, in.AccountID); perr == nil {
				for i := range list {
					if list[i].Enabled {
						fallbackPixel = &list[i]
						break
					}
				}
			}
		}

		if fallbackPixel == nil {
			slog.Warn("webhook user not found",
				"gateway", in.GatewayName,
				"external_id", in.ExternalID,
				"phone", in.Phone,
				"email", in.Email,
				"account_id", in.AccountID,
			)
			h.logWebhook(ctx, in, rawInput, in.AccountID, "", in.ExternalID, "", "failed", "no tracked user")
			return map[string]any{
				"received":     true,
				"matched_user": false,
				"reason":       "no tracked user",
			}, http.StatusOK
		}

		targetAccountID := fallbackPixel.AccountID
		if !h.verifyDeferredWebhookAuth(ctx, targetAccountID) {
			return map[string]any{"reason": "invalid platform token"}, http.StatusUnauthorized
		}

		syntheticExternalID := in.ExternalID
		if syntheticExternalID == "" {
			syntheticExternalID = util.GenerateID10()
		}
		newUser := &domain.TrackedUser{
			ExternalID: syntheticExternalID,
			IDWpp:      util.GenerateID10(),
			PixelName:  fallbackPixel.Name,
			AccountID:  targetAccountID,
			FirstName:  in.FirstName,
			LastName:   in.LastName,
			Email:      in.Email,
			Phone:      in.Phone,
			URL:        "",
		}
		if cerr := h.users.Create(ctx, newUser); cerr != nil {
			slog.Error("create fallback user from webhook", "err", cerr, "gateway", in.GatewayName)
		}
		user = newUser
	} else {
		if !h.verifyDeferredWebhookAuth(ctx, user.AccountID) {
			return map[string]any{"reason": "invalid platform token"}, http.StatusUnauthorized
		}
	}

	// Atualiza dados PII recebidos do gateway no usuário rastreado.
	user.FirstName = util.FirstNonEmpty(in.FirstName, user.FirstName)
	user.LastName = util.FirstNonEmpty(in.LastName, user.LastName)
	user.Email = util.FirstNonEmpty(in.Email, user.Email)
	user.Phone = util.FirstNonEmpty(in.Phone, user.Phone)
	if err := h.users.Upsert(ctx, user); err != nil {
		slog.Error("update user from webhook", "err", err, "gateway", in.GatewayName)
	}

	// Unifica identidade (stitch cross-device) ligando a sessão do checkout com sessões anteriores pelo fone/email
	if linked, lerr := h.users.ResolveAndLink(ctx, user.AccountID, user.ExternalID, user.Phone, user.Email); lerr == nil && linked != nil {
		user = linked
	}

	pixel, err := h.pixels.FindByName(ctx, user.PixelName)
	if err != nil {
		slog.Error("webhook pixel not found",
			"err", err,
			"pixel_name", user.PixelName,
			"gateway", in.GatewayName,
		)
		return map[string]any{
			"received": true,
			"error":    "pixel not found: " + user.PixelName,
		}, http.StatusInternalServerError
	}
	if !pixel.Enabled || pixel.AccountID != user.AccountID {
		return map[string]any{
			"received":     true,
			"matched_user": true,
			"queued":       false,
			"reason":       "pixel disabled",
		}, http.StatusOK
	}

	// Cookies/click ids mais recentes entre TODOS os external_id ligados a essa pessoa dentro da conta
	linked, err := h.users.LinkedFieldsByAccount(ctx, pixel.AccountID, user.ExternalID)
	if err != nil {
		slog.Warn("linked fields lookup failed", "err", err, "gateway", in.GatewayName, "external_id", user.ExternalID)
		linked = domain.LinkedTrackingFields{
			FBP: user.FBP, FBC: user.FBC, GCLID: user.GCLID, WBraid: user.WBraid, GBraid: user.GBraid,
			TTClid: user.TTClid, TTP: user.TTP, GA4ClientID: user.GA4ClientID, GA4SessionID: user.GA4SessionID,
		}
	}

	val := in.Value
	sourceURL := h.urlMapper.ApplyForAccount(pixel.AccountID, user.URL)

	itemType := in.OfferType
	if itemType == "" {
		if in.IsOrderBump {
			itemType = "order_bump"
		} else if in.IsUpsell {
			itemType = "upsell"
		}
	}
	eventID := webhookEventID(in.OrderID, itemType, in.ProductID)
	dedupItemKey := util.FirstNonEmpty(itemType, "main") + ":" + in.ProductID
	if !in.IsReplay && in.OrderID != "" {
		if claimed, ok := cache.ClaimWebhookDedup(ctx, h.rdb, pixel.AccountID, in.GatewayName, in.OrderID, in.EventName, dedupItemKey); ok && !claimed {
			return map[string]any{"received": true, "matched_user": true, "deduplicated": true, "event_id": eventID}, http.StatusOK
		}
	}

	actionSource, msgChannel, ctwaClid, pageID := h.ctwaAttribution(ctx, pixel.AccountID, user, linked.FBC, sourceURL)

	send := domain.SendEventInput{
		PixelName:        pixel.Name,
		EventName:        in.EventName,
		EventID:          eventID,
		EventTime:        time.Now().Unix(),
		SourceURL:        sourceURL,
		ContentID:        util.FirstNonEmpty(in.ProductID, pixel.Name),
		ContentName:      in.ProductName,
		ContentType:      "product",
		Value:            &val,
		Currency:         in.Currency,
		OrderID:          in.OrderID,
		ItemType:         itemType,
		Gateway:          in.GatewayName,
		ActionSource:     actionSource,
		MessagingChannel: msgChannel,
		CtwaClid:         ctwaClid,
		PageID:           pageID,
		ExternalID:       user.ExternalID,
		ClientIP:        user.ClientIP,
		ClientUserAgent: user.ClientUserAgent,
		FBP:             linked.FBP,
		FBC:             linked.FBC,
		Email:           user.Email,
		Phone:           user.Phone,
		FirstName:       user.FirstName,
		LastName:        user.LastName,
		Country:         user.Country,
		State:           user.State,
		City:            user.City,
		PostalCode:      user.PostalCode,
		TTClid:          linked.TTClid,
		TTP:             linked.TTP,
		GCLID:           linked.GCLID,
		WBraid:          linked.WBraid,
		GBraid:          linked.GBraid,
		GA4ClientID:     linked.GA4ClientID,
		GA4SessionID:    linked.GA4SessionID,
	}

	aggregated := false
	if pixel.AggregationEnabled && in.EventName == domain.EventPurchase && h.rdb != nil {
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

	if aggregated {
		h.logWebhook(ctx, in, rawInput, pixel.AccountID, pixel.Name, user.ExternalID, eventID, "buffered", "compra em buffer na janela de "+strconv.Itoa(pixel.AggregationWindowMinutes)+"min")
	} else {
		h.logWebhook(ctx, in, rawInput, pixel.AccountID, pixel.Name, user.ExternalID, eventID, "queued", "")
	}

	return map[string]any{
		"received":     true,
		"matched_user": true,
		"event_id":     eventID,
		"event_name":   in.EventName,
		"aggregated":   aggregated,
	}, http.StatusAccepted
}
