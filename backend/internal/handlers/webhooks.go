package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/redis/go-redis/v9"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/queue"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

// GatewaySecrets agrupa segredos opcionais usados na validação de assinatura
// de webhooks. Hoje vazia — Cakto, Hotmart e Kirvano não exigem
// HMAC. Mantido por extensibilidade.
type GatewaySecrets struct{}

// Webhooks expõe um handler por gateway. O processamento real é compartilhado
// em processGenericPurchase para garantir que todos os 22 gateways apliquem
// a mesma regra (lookup do tracked user, atualização de PII, mapeamento de
// URL, enfileiramento do evento).
type Webhooks struct {
	pixels      *repo.PixelRepo
	users       *repo.UserRepo
	queue       *queue.Enqueuer
	urlMapper   *util.URLMapper
	secrets     GatewaySecrets
	tokens      *repo.PlatformTokenRepo
	webhookLogs *repo.WebhookLogRepo // opcional: nil = sem logging
	whatsapp    *repo.WhatsAppRepo   // opcional: resolve page_id em vendas CTWA
	rdb         *redis.Client
}

func NewWebhooks(
	p *repo.PixelRepo,
	u *repo.UserRepo,
	q *queue.Enqueuer,
	urls *util.URLMapper,
	secrets GatewaySecrets,
	tokens *repo.PlatformTokenRepo,
) *Webhooks {
	return &Webhooks{
		pixels:    p,
		users:     u,
		queue:     q,
		urlMapper: urls,
		secrets:   secrets,
		tokens:    tokens,
	}
}

// SetWebhookLogs ativa o logging de webhooks recebidos.
func (h *Webhooks) SetWebhookLogs(wl *repo.WebhookLogRepo) { h.webhookLogs = wl }

// SetRedis injeta o cliente Redis para buffer de agregação de compras.
func (h *Webhooks) SetRedis(r *redis.Client) { h.rdb = r }

// SetWhatsAppRepo permite enriquecer conversões de vendas fechadas após anúncio CTWA.
func (h *Webhooks) SetWhatsAppRepo(wa *repo.WhatsAppRepo) { h.whatsapp = wa }

// ReplayWebhook reprocessa um webhook_log pelo ID.
// Chamado pelo painel admin no endpoint POST /admin/webhook-logs/:id/replay.
func (h *Webhooks) ReplayWebhook(ctx context.Context, id int64, accountID int64) (map[string]any, error) {
	if h.webhookLogs == nil {
		return nil, errors.New("webhook logging não ativo")
	}
	wpi, log, err := h.webhookLogs.FindInputByID(ctx, id, accountID)
	if err != nil {
		return nil, err
	}
	if wpi == nil {
		return nil, errors.New("input_json indisponível (webhook falhou antes do enfileiramento)")
	}
	in := toPurchaseInput(wpi)
	in.IsReplay = true
	result, _ := h.enqueueGenericPurchase(ctx, in, nil)
	// Marca como replayed
	status := "replayed"
	errMsg := ""
	if e, ok := result["error"].(string); ok && e != "" {
		status = "failed"
		errMsg = e
	}
	_ = h.webhookLogs.UpdateStatus(ctx, log.ID, status, errMsg)
	return result, nil
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

	// Identificação de produto e tipo de oferta (Order Bump / Upsell)
	ProductName string
	ProductID   string
	IsOrderBump bool
	IsUpsell    bool
	OfferType   string // "main", "order_bump", "upsell"

	// Fallback por PixelQuery (ex: ?pixel=slug ou ?token=xyz na URL do webhook)
	PixelQuery string

	// AccountID extraído da URL (?acc= ou ?account_id=) para isolamento multi-tenant seguro
	AccountID int64

	// Estratégia de fallback de lookup: além de external_id e phone, também
	// tenta email (agora suportado nativamente para todos os gateways).
	AlsoSearchByEmail bool

	// Opcional: payload cru e ID do webhook customizado para auditoria unificada.
	RawPayload      []byte
	CustomWebhookID string
	IsReplay        bool
}

// processGenericPurchase é o fluxo single-item: processa UM purchaseInput,
// loga o resultado em webhook_logs (se ativo) e escreve a resposta HTTP.
func (h *Webhooks) processGenericPurchase(ctx context.Context, w http.ResponseWriter, in purchaseInput, rawInput *domain.WebhookPurchaseInput) {
	body, status := h.enqueueGenericPurchase(ctx, in, rawInput)
	writeJSON(w, status, body)
}
