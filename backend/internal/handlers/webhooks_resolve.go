package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

// resolveWebhookAccount resolve e valida a conta/pixel alvo do webhook ANTES de qualquer lookup.
//
// Regras de isolamento multi-conta:
//   - ?pixel= de outra conta que não a do ?acc= é rejeitado (403).
//   - Sem conta resolvida, só aceita se a instalação tiver exatamente uma conta (400 caso contrário).
func (h *Webhooks) resolveWebhookAccount(ctx context.Context, in *purchaseInput) (*domain.Pixel, int, string) {
	var resolved *domain.Pixel
	if in.PixelQuery != "" && h.pixels != nil {
		if p, err := h.pixels.FindByNameOrToken(ctx, in.PixelQuery); err == nil && p != nil && p.Enabled {
			resolved = p
		}
	}

	if resolved != nil {
		if in.AccountID > 0 && in.AccountID != resolved.AccountID {
			slog.Warn("webhook pixel/account mismatch",
				"gateway", in.GatewayName, "account_id", in.AccountID, "pixel_account_id", resolved.AccountID)
			return nil, http.StatusForbidden, "pixel does not belong to account"
		}
		in.AccountID = resolved.AccountID
	}

	if in.AccountID <= 0 {
		if sole, ok := h.users.SoleAccountID(ctx); ok {
			in.AccountID = sole
		} else {
			slog.Warn("webhook without account context", "gateway", in.GatewayName)
			return nil, http.StatusBadRequest, "missing account context: add ?acc= or ?pixel= to the webhook url"
		}
	}
	return resolved, 0, ""
}

// webhookEventID gera um event_id único por item: order bump/upsell recebem sufixo
// para não serem deduplicados contra o item principal (worker + Meta).
func webhookEventID(orderID, itemType, productID string) string {
	if orderID == "" {
		return util.GenerateID10()
	}
	if itemType == "" || itemType == "main" {
		return orderID
	}
	return orderID + ":" + itemType + ":" + productID
}

// ctwaAttribution decide se a venda vai como business_messaging (CTWA) ou website.
// business_messaging só é usado quando há ctwa_clid E page_id resolvido; caso contrário
// ctwa_clid não é enviado (evita 400 permanente da Meta).
func (h *Webhooks) ctwaAttribution(ctx context.Context, accountID int64, user *domain.TrackedUser, fbc, sourceURL string) (actionSource, channel, ctwaClid, pageID string) {
	actionSource = "website"
	if user.CtwaClid == "" || fbc != "" || sourceURL != "" || h.whatsapp == nil {
		return
	}
	pid := h.whatsapp.FindPageIDByCtwaClid(ctx, accountID, user.CtwaClid)
	if pid == "" {
		return
	}
	return "business_messaging", "whatsapp", user.CtwaClid, pid
}

