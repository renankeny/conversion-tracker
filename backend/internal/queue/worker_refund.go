package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/hibiken/asynq"

	"github.com/renankeny/conversion-tracker/backend/internal/cache"
	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/notify"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

func makeProcessRefundHandler(deps Deps) func(context.Context, *asynq.Task) error {
	return func(ctx context.Context, t *asynq.Task) error {
		var p RefundAdjustmentPayload
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			return fmt.Errorf("unmarshal refund payload: %w", err)
		}
		if deps.Refunds == nil || p.RefundID <= 0 {
			return nil
		}
		ref, err := deps.Refunds.FindByID(ctx, p.RefundID)
		if err != nil || ref == nil {
			return nil
		}

		var pixel *domain.Pixel
		if deps.Pixels != nil && ref.PixelName != "" {
			pixel, _ = deps.Pixels.FindByName(ctx, ref.PixelName)
		}
		var user *domain.TrackedUser
		if deps.Users != nil && ref.TrackedUserID != nil && *ref.TrackedUserID > 0 {
			user, _ = deps.Users.FindByID(ctx, *ref.TrackedUserID)
		}

		dispatchGoogleAdsRetraction(ctx, deps, ref, pixel)
		dispatchGA4Refund(ctx, deps, ref, pixel, user)
		emitRefundNotification(deps, ref, pixel, user)
		cache.BumpDashboardVersion(ctx, deps.Redis, ref.AccountID)
		return nil
	}
}

func dispatchGoogleAdsRetraction(ctx context.Context, deps Deps, ref *domain.SaleRefund, pixel *domain.Pixel) {
	if ref.GoogleRetractionStatus == domain.AdjustmentStatusSent {
		return
	}
	if deps.GoogleAds == nil || deps.GoogleAdsAcc == nil || pixel == nil || !pixel.HasGoogleAds() || strings.TrimSpace(ref.OrderID) == "" {
		_ = deps.Refunds.UpdateAdjustmentStatus(ctx, ref.ID, "google", domain.AdjustmentStatusSkipped, "")
		return
	}
	gAcc, err := deps.GoogleAdsAcc.FindByID(ctx, *pixel.GoogleAdsAccountID)
	if err != nil || gAcc == nil || !gAcc.Enabled || !gAcc.HasCredentials() {
		_ = deps.Refunds.UpdateAdjustmentStatus(ctx, ref.ID, "google", domain.AdjustmentStatusSkipped, "conta Google Ads inativa")
		return
	}
	action := gAcc.FindActionForEvent(domain.EventPurchase)
	if action == nil {
		_ = deps.Refunds.UpdateAdjustmentStatus(ctx, ref.ID, "google", domain.AdjustmentStatusSkipped, "sem ação Purchase no Google Ads")
		return
	}
	if _, err := deps.GoogleAds.RetractConversion(ctx, gAcc, action, ref.OrderID, ref.RefundedAt); err != nil {
		slog.Warn("google ads retraction failed", "refund_id", ref.ID, "order_id", ref.OrderID, "err", err)
		_ = deps.Refunds.UpdateAdjustmentStatus(ctx, ref.ID, "google", domain.AdjustmentStatusFailed, err.Error())
		return
	}
	_ = deps.Refunds.UpdateAdjustmentStatus(ctx, ref.ID, "google", domain.AdjustmentStatusSent, "")
}

func dispatchGA4Refund(ctx context.Context, deps Deps, ref *domain.SaleRefund, pixel *domain.Pixel, user *domain.TrackedUser) {
	if ref.GA4RefundStatus == domain.AdjustmentStatusSent {
		return
	}
	if deps.GA4 == nil || pixel == nil || strings.TrimSpace(pixel.GA4MeasurementID) == "" || strings.TrimSpace(pixel.GA4APISecret) == "" || strings.TrimSpace(ref.OrderID) == "" {
		_ = deps.Refunds.UpdateAdjustmentStatus(ctx, ref.ID, "ga4", domain.AdjustmentStatusSkipped, "")
		return
	}
	var ga4ClientID, extID string
	if user != nil {
		ga4ClientID = user.GA4ClientID
		extID = user.ExternalID
	}
	res := deps.GA4.SendRefund(
		ctx, pixel.GA4APISecret, pixel.GA4MeasurementID,
		ga4ClientID, extID, ref.OrderID, ref.Currency, ref.ProductName, ref.Amount, ref.RefundedAt,
	)
	if res.Err != nil {
		slog.Warn("ga4 refund dispatch failed", "refund_id", ref.ID, "order_id", ref.OrderID, "err", res.Err)
		_ = deps.Refunds.UpdateAdjustmentStatus(ctx, ref.ID, "ga4", domain.AdjustmentStatusFailed, res.Err.Error())
		return
	}
	_ = deps.Refunds.UpdateAdjustmentStatus(ctx, ref.ID, "ga4", domain.AdjustmentStatusSent, "")
}

func emitRefundNotification(deps Deps, ref *domain.SaleRefund, pixel *domain.Pixel, user *domain.TrackedUser) {
	if deps.Notify == nil {
		return
	}
	label := "Reembolso"
	if ref.RefundType == domain.RefundTypeChargeback {
		label = "Chargeback"
	}
	money := notify.FormatMoney(ref.Amount, ref.Currency)
	origin := notify.FriendlyGatewayOrigin(ref.Gateway)
	pxLabel := ref.PixelName
	if pixel != nil {
		pxLabel = util.FirstNonEmpty(pixel.Label, pixel.Name)
	}
	parts := []string{"Origem: " + origin}
	if ref.OrderID != "" {
		parts = append(parts, "Pedido: #"+ref.OrderID)
	}
	if pxLabel != "" {
		parts = append(parts, "Pixel: "+pxLabel)
	}
	if user != nil {
		buyer := strings.TrimSpace(user.FirstName + " " + user.LastName)
		if buyer == "" {
			buyer = util.FirstNonEmpty(user.Phone, user.Email)
		}
		if buyer != "" {
			parts = append(parts, "Cliente: "+buyer)
		}
	}
	deps.Notify.DispatchAsync(notify.Event{
		AccountID:   ref.AccountID,
		Category:    domain.NotifSaleRefunded,
		Title:       fmt.Sprintf("%s: %s", label, money),
		Body:        strings.Join(parts, " • ") + ".",
		ActionURL:   "/campanhas",
		CooldownKey: fmt.Sprintf("refund:%d:%s:%s:%s", ref.AccountID, ref.Gateway, ref.OrderID, ref.RefundType),
		CooldownTTL: 24 * time.Hour,
	})
}
