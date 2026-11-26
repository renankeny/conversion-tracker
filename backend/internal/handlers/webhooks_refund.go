package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/cache"
	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/notify"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
	"github.com/renankeny/conversion-tracker/backend/internal/wshub"
)

// detectRefundOrChargeback identifica eventos canônicos de reembolso ou chargeback
// enviados pelos gateways de checkout (string livre de event/status).
func detectRefundOrChargeback(fields ...string) string {
	for _, raw := range fields {
		s := strings.ToLower(strings.TrimSpace(raw))
		if s == "" {
			continue
		}
		if strings.Contains(s, "chargeback") || strings.Contains(s, "chargedback") ||
			strings.Contains(s, "charged_back") || strings.Contains(s, "disput") ||
			strings.Contains(s, "contestad") || strings.Contains(s, "mediac") {
			return domain.EventChargeback
		}
		if strings.Contains(s, "refund") || strings.Contains(s, "reembols") ||
			strings.Contains(s, "estorn") || strings.Contains(s, "devolvid") ||
			strings.Contains(s, "devoluc") || s == "returned" {
			return domain.EventRefund
		}
	}
	return ""
}

func (h *Webhooks) processGenericRefund(ctx context.Context, in purchaseInput, rawInput *domain.WebhookPurchaseInput) (map[string]any, int) {
	resolvedQueryPixel, status, reason := h.resolveWebhookAccount(ctx, &in)
	if status != 0 {
		h.logWebhook(ctx, in, rawInput, in.AccountID, "", in.ExternalID, "", "failed", reason)
		return map[string]any{"received": true, "error": reason}, status
	}
	if !h.verifyDeferredWebhookAuth(ctx, in.AccountID) {
		return map[string]any{"reason": "invalid platform token"}, http.StatusUnauthorized
	}

	var origPixel, origExtID, origEventID string
	var origVal float64
	if h.refunds != nil && in.OrderID != "" {
		origPixel, origExtID, origEventID, origVal = h.refunds.FindOriginalSaleByOrder(ctx, in.AccountID, in.OrderID)
	}
	if in.ExternalID == "" && origExtID != "" {
		in.ExternalID = origExtID
	}

	user, _ := h.findUser(ctx, in)
	if user != nil {
		h.users.EnrichGroupFields(ctx, user)
	}

	pixelName := ""
	switch {
	case resolvedQueryPixel != nil:
		pixelName = resolvedQueryPixel.Name
	case origPixel != "":
		pixelName = origPixel
	case user != nil && user.PixelName != "":
		pixelName = user.PixelName
	default:
		if fb := h.selectFallbackPixel(ctx, nil, in.AccountID); fb != nil {
			pixelName = fb.Name
		}
	}

	amount := in.Value
	if amount <= 0 && origVal > 0 {
		amount = origVal
	}
	orderID := strings.TrimSpace(in.OrderID)
	if orderID == "" {
		orderID = util.GenerateID10()
	}

	refundType := domain.NormalizeRefundType(in.EventName)
	eventID := strings.ToLower(in.EventName) + ":" + orderID

	if h.refunds == nil {
		h.logWebhook(ctx, in, rawInput, in.AccountID, pixelName, in.ExternalID, eventID, "failed", "refund repo não configurado")
		return map[string]any{"received": true, "error": "refund storage unavailable"}, http.StatusInternalServerError
	}

	refundedAt := time.Now()
	if ts := strings.TrimSpace(in.RefundedAtISO); ts != "" {
		if parsed, perr := time.Parse(time.RFC3339Nano, ts); perr == nil {
			refundedAt = parsed
		} else if parsed, perr := time.Parse(time.RFC3339, ts); perr == nil {
			refundedAt = parsed
		}
	}

	rec := &domain.SaleRefund{
		AccountID:       in.AccountID,
		PixelName:       pixelName,
		Gateway:         in.GatewayName,
		OrderID:         orderID,
		RefundType:      refundType,
		Amount:          amount,
		Currency:        in.Currency,
		OriginalEventID: util.FirstNonEmpty(origEventID, orderID),
		ProductName:     in.ProductName,
		RawPayload:      in.RawPayload,
		RefundedAt:      refundedAt,
	}
	if user != nil && user.ID > 0 {
		uid := user.ID
		rec.TrackedUserID = &uid
		rec.GroupID = user.GroupID
	}

	inserted, err := h.refunds.Upsert(ctx, rec)
	if err != nil {
		slog.Error("upsert sale_refund", "gateway", in.GatewayName, "order_id", orderID, "err", err)
		h.logWebhook(ctx, in, rawInput, in.AccountID, pixelName, in.ExternalID, eventID, "failed", err.Error())
		return map[string]any{"received": true, "error": "failed to persist refund"}, http.StatusInternalServerError
	}

	if inserted {
		if h.crmRepo != nil && user != nil && user.ID > 0 {
			notePrefix := "⚠️ Reembolso"
			if refundType == domain.RefundTypeChargeback {
				notePrefix = "🚨 Chargeback"
			}
			note := fmt.Sprintf("%s via %s (Pedido #%s) — %s",
				notePrefix, notify.FriendlyGatewayOrigin(in.GatewayName), orderID, notify.FormatMoney(amount, in.Currency))
			if reason := strings.TrimSpace(in.RefundReason); reason != "" {
				note += " | Motivo: " + reason
			}
			if leadID, updated := h.crmRepo.ApplyRefundToLead(ctx, in.AccountID, user.ID, refundType, amount, note); updated {
				h.refunds.SetCRMLeadID(ctx, rec.ID, leadID)
				if h.rdb != nil {
					wshub.PublishCustom(ctx, h.rdb, in.AccountID, "crm_updated", map[string]any{
						"action": "webhook_refund_synced", "lead_id": leadID, "event": in.EventName,
					})
				}
			}
		}
		if h.queue != nil {
			_ = h.queue.EnqueueRefundAdjustment(ctx, rec.ID)
		}
		cache.BumpDashboardVersion(ctx, h.rdb, in.AccountID)
	}

	logStatus := "processed"
	if !inserted {
		logStatus = "deduplicated"
	}
	extID := in.ExternalID
	if extID == "" && user != nil {
		extID = user.ExternalID
	}
	h.logWebhook(ctx, in, rawInput, in.AccountID, pixelName, extID, eventID, logStatus, "")
	return map[string]any{
		"received":     true,
		"refund_id":    rec.ID,
		"refund_type":  refundType,
		"deduplicated": !inserted,
		"event_name":   in.EventName,
		"order_id":     orderID,
	}, http.StatusOK
}
