package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/queue"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

// GatewaySecrets agrupa segredos opcionais usados na validação de assinatura
// de webhooks. Hoje vazia — Cakto, Hotmart, Kirvano e RKCursos não exigem
// HMAC. Mantido por extensibilidade.
type GatewaySecrets struct{}

// Webhooks expõe um handler por gateway. O processamento real é compartilhado
// em processGenericPurchase para garantir que todos os 22 gateways apliquem
// a mesma regra (lookup do tracked user, atualização de PII, mapeamento de
// URL, enfileiramento do evento).
type Webhooks struct {
	pixels    *repo.PixelRepo
	users     *repo.UserRepo
	queue     *queue.Enqueuer
	urlMapper *util.URLMapper
	secrets   GatewaySecrets
}

func NewWebhooks(
	p *repo.PixelRepo,
	u *repo.UserRepo,
	q *queue.Enqueuer,
	urls *util.URLMapper,
	secrets GatewaySecrets,
) *Webhooks {
	return &Webhooks{
		pixels:    p,
		users:     u,
		queue:     q,
		urlMapper: urls,
		secrets:   secrets,
	}
}

// purchaseInput é o resultado da extração de dados de QUALQUER gateway.
// Os parsers específicos de cada gateway (webhook_<nome>.go) preenchem essa
// struct e chamam processGenericPurchase para o restante do fluxo.
type purchaseInput struct {
	GatewayName string  // identificador legível para logs (ex: "cakto")
	EventName   string  // domain.EventPurchase ou domain.EventAddPaymentInfo
	ExternalID  string  // userId extraído do utm_source (pode vir vazio)
	OrderID     string  // id da venda no gateway, usado como event_id pra dedup natural
	Value       float64 // valor da venda (já em moeda real, não centavos)
	Currency    string

	// PII recebida do gateway
	FirstName string
	LastName  string
	Email     string
	Phone     string

	// Estratégia de fallback de lookup: além de external_id e phone, também
	// tenta email (RkCursos faz isso porque não recebe utm_source).
	AlsoSearchByEmail bool
}

// processGenericPurchase é o fluxo único compartilhado pelos 22 webhooks:
//
//  1. Encontra o tracked user (external_id → phone (com variantes) → email opcional).
//  2. Atualiza PII recebida do gateway no tracked_users.
//  3. Aplica URL black→white na URL gravada do user.
//  4. Resolve o pixel (`pixel_name` salvo no tracked user).
//  5. Enfileira o evento (Purchase ou AddPaymentInfo) na fila Meta CAPI.
func (h *Webhooks) processGenericPurchase(ctx context.Context, w http.ResponseWriter, in purchaseInput) {
	in.EventName = orDefaultEvent(in.EventName)
	in.Currency = orDefaultCurrency(in.Currency)
	in.Phone = util.NormalizePhone(in.Phone)
	in.Email = util.NormalizeEmail(in.Email)

	user, err := h.findUser(ctx, in)
	if err != nil {
		slog.Warn("webhook user not found",
			"gateway", in.GatewayName,
			"external_id", in.ExternalID,
			"phone", in.Phone,
			"email", in.Email,
		)
		// 200 OK pra não fazer o gateway re-tentar indefinidamente — o evento
		// de fato não pôde ser correlacionado a nenhum visitante rastreado.
		writeJSON(w, http.StatusOK, map[string]any{
			"received":     true,
			"matched_user": false,
			"reason":       "no tracked user",
		})
		return
	}

	// Atualiza dados PII recebidos do gateway no usuário rastreado.
	user.FirstName = util.FirstNonEmpty(in.FirstName, user.FirstName)
	user.LastName = util.FirstNonEmpty(in.LastName, user.LastName)
	user.Email = util.FirstNonEmpty(in.Email, user.Email)
	user.Phone = util.FirstNonEmpty(in.Phone, user.Phone)
	if err := h.users.Upsert(ctx, user); err != nil {
		slog.Error("update user from webhook", "err", err, "gateway", in.GatewayName)
	}

	pixel, err := h.pixels.FindByName(ctx, user.PixelName)
	if err != nil {
		slog.Error("webhook pixel not found",
			"err", err,
			"pixel_name", user.PixelName,
			"gateway", in.GatewayName,
		)
		writeError(w, http.StatusInternalServerError, "pixel not found: "+user.PixelName)
		return
	}
	if !pixel.Enabled {
		writeJSON(w, http.StatusOK, map[string]any{
			"received":     true,
			"matched_user": true,
			"queued":       false,
			"reason":       "pixel disabled",
		})
		return
	}

	val := in.Value
	eventID := util.FirstNonEmpty(in.OrderID, util.GenerateID10())
	sourceURL := h.urlMapper.Apply(user.URL)

	send := domain.SendEventInput{
		PixelName:       pixel.Name,
		EventName:       in.EventName,
		EventID:         eventID,
		EventTime:       time.Now().Unix(),
		SourceURL:       sourceURL,
		ContentID:       pixel.Name,
		ContentType:     "product",
		Value:           &val,
		Currency:        in.Currency,
		OrderID:         in.OrderID,
		ExternalID:      user.ExternalID,
		ClientIP:        user.ClientIP,
		ClientUserAgent: user.ClientUserAgent,
		FBP:             user.FBP,
		FBC:             user.FBC,
		Email:           user.Email,
		Phone:           user.Phone,
		FirstName:       user.FirstName,
		LastName:        user.LastName,
		Country:         user.Country,
		State:           user.State,
		City:            user.City,
		PostalCode:      user.PostalCode,
	}
	if err := h.queue.EnqueueSendEvent(ctx, send); err != nil {
		slog.Error("enqueue webhook event", "err", err, "gateway", in.GatewayName, "event_id", eventID)
		writeError(w, http.StatusInternalServerError, "queue error")
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{
		"received":     true,
		"matched_user": true,
		"event_id":     eventID,
		"event_name":   in.EventName,
	})
}

// findUser implementa a estratégia de lookup compartilhada por todos os gateways.
//
// Prioridade:
//  1. external_id (vindo do utm_source no formato `userId<id>`)
//  2. telefone (com variantes 55-prefix / sem)
//  3. email (apenas se in.AlsoSearchByEmail = true; caso do RkCursos)
func (h *Webhooks) findUser(ctx context.Context, in purchaseInput) (*domain.TrackedUser, error) {
	if in.ExternalID != "" {
		u, err := h.users.FindByExternalID(ctx, in.ExternalID)
		if err == nil {
			return u, nil
		}
		if !errors.Is(err, repo.ErrNotFound) {
			slog.Warn("user lookup by external_id failed", "err", err)
		}
	}
	if in.Phone != "" {
		variants := util.PhoneVariants(in.Phone)
		u, err := h.users.FindByPhoneVariants(ctx, variants)
		if err == nil {
			return u, nil
		}
		if !errors.Is(err, repo.ErrNotFound) {
			slog.Warn("user lookup by phone failed", "err", err)
		}
	}
	if in.AlsoSearchByEmail && in.Email != "" {
		u, err := h.users.FindByEmail(ctx, in.Email)
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
