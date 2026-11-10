package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

// findUser implementa a estratégia de lookup compartilhada por todos os gateways.
//
// Prioridade:
//  1. telefone (com variantes 55-prefix / sem) - PRIORIDADE pra garantir cookies
//  2. external_id (vindo do utm_source no formato `userId<id>`)
//  3. email (fallback)
func (h *Webhooks) findUser(ctx context.Context, in purchaseInput) (*domain.TrackedUser, error) {
	// 1. Busca por telefone PRIMEIRO (melhor match rate - tem cookies)
	if in.Phone != "" {
		variants := util.PhoneVariants(in.Phone)
		u, err := h.users.FindByPhoneVariantsByAccount(ctx, variants, in.AccountID)
		if err == nil {
			return u, nil
		}
		if !errors.Is(err, repo.ErrNotFound) {
			slog.Warn("user lookup by phone failed", "err", err)
		}
	}

	// 2. Fallback: busca por external_id (userId do tracker)
	if in.ExternalID != "" {
		u, err := h.users.FindByExternalIDByAccount(ctx, in.ExternalID, in.AccountID)
		if err == nil {
			return u, nil
		}
		if !errors.Is(err, repo.ErrNotFound) {
			slog.Warn("user lookup by external_id failed", "err", err)
		}
	}

	// 3. Fallback por email: se telefone e external_id não bateram
	if in.Email != "" {
		u, err := h.users.FindByEmailByAccount(ctx, in.Email, in.AccountID)
		if err == nil {
			return u, nil
		}
	}
	return nil, errors.New("not found")
}

func orDefaultEvent(s string) string {
	if s == "" {
		return domain.EventPurchase
	}
	return s
}

func orDefaultCurrency(s string) string {
	if s == "" {
		return "BRL"
	}
	return s
}

// ExtractAccountID lê o ID da conta a partir da query string do webhook (?acc=, ?account_id=, etc).
func ExtractAccountID(r *http.Request) int64 {
	for _, param := range []string{"acc", "account_id", "acc_id", "account"} {
		if val := r.URL.Query().Get(param); val != "" {
			if id, err := strconv.ParseInt(val, 10, 64); err == nil && id > 0 {
				return id
			}
		}
	}
	return 0
}

// logWebhook registra o resultado de um webhook em webhook_logs (se ativo).
func (h *Webhooks) logWebhook(
	ctx context.Context,
	in purchaseInput,
	rawInput *domain.WebhookPurchaseInput,
	accountID int64,
	pixelName, externalID, eventID, status, errMsg string,
) {
	if h.webhookLogs == nil {
		return
	}
	if accountID == 0 {
		accountID = 1
	}
	eventName := in.EventName
	if eventName == "" {
		eventName = domain.EventPurchase
	}
	log := &domain.WebhookLog{
		AccountID:  accountID,
		Source:     in.GatewayName,
		PixelName:  pixelName,
		ExternalID: externalID,
		EventName:  eventName,
		EventID:    eventID,
		Status:     status,
		Error:      errMsg,
		ReceivedAt: time.Now(),
	}
	if len(in.RawPayload) > 0 {
		if err := h.webhookLogs.CreateWithRawPayload(ctx, log, rawInput, in.RawPayload, in.CustomWebhookID); err != nil {
			slog.Warn("webhook_log create with raw payload failed", "err", err, "gateway", in.GatewayName)
		}
		return
	}
	if err := h.webhookLogs.Create(ctx, log, rawInput); err != nil {
		slog.Warn("webhook_log create failed", "err", err, "gateway", in.GatewayName)
	}
}

// toPurchaseInput converte WebhookPurchaseInput → purchaseInput (interno).
func toPurchaseInput(w *domain.WebhookPurchaseInput) purchaseInput {
	return purchaseInput{
		GatewayName:       w.GatewayName,
		EventName:         w.EventName,
		ExternalID:        w.ExternalID,
		OrderID:           w.OrderID,
		Value:             w.Value,
		Currency:          w.Currency,
		FirstName:         w.FirstName,
		LastName:          w.LastName,
		Email:             w.Email,
		Phone:             w.Phone,
		AccountID:         w.AccountID,
		PixelQuery:        w.PixelQuery,
		AlsoSearchByEmail: w.AlsoSearchByEmail,
		ProductName:       w.ProductName,
		ProductID:         w.ProductID,
		IsOrderBump:       w.IsOrderBump,
		IsUpsell:          w.IsUpsell,
		OfferType:         w.OfferType,
	}
}

// fromPurchaseInput converte purchaseInput → WebhookPurchaseInput (serializável).
func fromPurchaseInput(in purchaseInput) *domain.WebhookPurchaseInput {
	return &domain.WebhookPurchaseInput{
		GatewayName:       in.GatewayName,
		EventName:         in.EventName,
		ExternalID:        in.ExternalID,
		OrderID:           in.OrderID,
		Value:             in.Value,
		Currency:          in.Currency,
		FirstName:         in.FirstName,
		LastName:          in.LastName,
		Email:             in.Email,
		Phone:             in.Phone,
		AccountID:         in.AccountID,
		PixelQuery:        in.PixelQuery,
		AlsoSearchByEmail: in.AlsoSearchByEmail,
		ProductName:       in.ProductName,
		ProductID:         in.ProductID,
		IsOrderBump:       in.IsOrderBump,
		IsUpsell:          in.IsUpsell,
		OfferType:         in.OfferType,
	}
}
