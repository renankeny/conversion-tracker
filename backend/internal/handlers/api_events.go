package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/redis/go-redis/v9"

	"github.com/renankeny/conversion-tracker/backend/internal/geoip"
	"github.com/renankeny/conversion-tracker/backend/internal/queue"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

type APIEvents struct {
	apiKeys    *repo.APIKeyRepo
	pixels     *repo.PixelRepo
	users      *repo.UserRepo
	sentEvents *repo.SentEventRepo
	geo        *geoip.Lookup
	queue      *queue.Enqueuer
	urls       *util.URLMapper
	crmRepo    *repo.CRMRepo
	rdb        *redis.Client
}

func NewAPIEvents(
	ak *repo.APIKeyRepo,
	p *repo.PixelRepo,
	u *repo.UserRepo,
	s *repo.SentEventRepo,
	g *geoip.Lookup,
	q *queue.Enqueuer,
	urls *util.URLMapper,
	crm *repo.CRMRepo,
) *APIEvents {
	return &APIEvents{
		apiKeys:    ak,
		pixels:     p,
		users:      u,
		sentEvents: s,
		geo:        g,
		queue:      q,
		urls:       urls,
		crmRepo:    crm,
	}
}

func (h *APIEvents) SetRedis(rdb *redis.Client) {
	h.rdb = rdb
}

// apiEventRequest representa o payload de um evento enviado via API externa.
// Usa snake_case para consistência com a convenção REST da API pública.
type apiEventRequest struct {
	// Obrigatórios
	PixelName  string `json:"pixel_name"`  // Nome do pixel cadastrado
	EventName  string `json:"event_name"`  // PageView, Purchase, Lead, AddToCart, etc
	ExternalID string `json:"external_id"` // ID único do usuário (obrigatório)

	// Identificadores opcionais
	EventID   string `json:"event_id,omitempty"`   // ID do evento (gerado se omitido)
	IDWpp     string `json:"id_wpp,omitempty"`     // ID WhatsApp (se aplicável)
	EventTime *int64 `json:"event_time,omitempty"` // Unix timestamp (padrão: now)

	// Dados do produto/conversão
	ContentID   string   `json:"content_id,omitempty"`   // ID do produto
	ContentName string   `json:"content_name,omitempty"` // Nome do produto
	ContentType string   `json:"content_type,omitempty"` // "product" ou "product_group"
	Value       *float64 `json:"value,omitempty"`        // Valor monetário
	Currency    string   `json:"currency,omitempty"`     // BRL, USD, etc
	OrderID     string   `json:"order_id,omitempty"`     // ID do pedido (recomendado Purchase)
	NumItems    *int     `json:"num_items,omitempty"`    // Quantidade de itens

	// Dados do usuário (PII)
	Email     string `json:"email,omitempty"`
	Phone     string `json:"phone,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`

	// Localização (inferida do IP se omitida)
	Country    string `json:"country,omitempty"`
	State      string `json:"state,omitempty"`
	City       string `json:"city,omitempty"`
	PostalCode string `json:"postal_code,omitempty"`
	ClientIP   string `json:"client_ip,omitempty"` // IP do usuário final (padrão: request IP)

	// Identificadores de rastreamento
	FBP          string `json:"fbp,omitempty"`            // Cookie _fbp do Meta Pixel
	FBC          string `json:"fbc,omitempty"`            // Cookie _fbc do Meta Pixel
	TTClid       string `json:"ttclid,omitempty"`         // TikTok Click ID
	TTP          string `json:"ttp,omitempty"`            // Cookie _ttp do TikTok
	GCLID        string `json:"gclid,omitempty"`          // Google Click ID
	WBraid       string `json:"wbraid,omitempty"`         // Google Wbraid
	GBraid       string `json:"gbraid,omitempty"`         // Google Gbraid
	GA4ClientID  string `json:"ga4_client_id,omitempty"`  // Cookie _ga Client ID
	GA4SessionID string `json:"ga4_session_id,omitempty"` // Cookie _ga_<id> Session ID

	// URLs e contexto
	SourceURL string `json:"source_url,omitempty"` // URL da página de origem
	Referrer  string `json:"referrer,omitempty"`   // Referrer HTTP

	// User Agent (inferido do request se omitido)
	ClientUserAgent string `json:"client_user_agent,omitempty"`

	// UTMs (primeira origem)
	UTMSource   string `json:"utm_source,omitempty"`
	UTMCampaign string `json:"utm_campaign,omitempty"`
	UTMMedium   string `json:"utm_medium,omitempty"`
	UTMContent  string `json:"utm_content,omitempty"`
	UTMTerm     string `json:"utm_term,omitempty"`
}

// apiBatchRequest permite enviar múltiplos eventos em uma única requisição.
type apiBatchRequest struct {
	Events []apiEventRequest `json:"events"`
}

// SendEvent recebe um único evento via API autenticada.
// POST /api/v1/send-event
func (h *APIEvents) SendEvent(w http.ResponseWriter, r *http.Request) {
	accountID, err := h.authenticate(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid or missing API key")
		return
	}

	var req apiEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	result, httpStatus, errMsg := h.processEvent(r.Context(), req, accountID, r)
	if errMsg != "" {
		writeError(w, httpStatus, errMsg)
		return
	}

	writeJSON(w, httpStatus, result)
}

// SendEventsBatch recebe múltiplos eventos em lote.
// POST /api/v1/send-events-batch
func (h *APIEvents) SendEventsBatch(w http.ResponseWriter, r *http.Request) {
	accountID, err := h.authenticate(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid or missing API key")
		return
	}

	var req apiBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	if len(req.Events) == 0 {
		writeError(w, http.StatusBadRequest, "events array cannot be empty")
		return
	}

	if len(req.Events) > 100 {
		writeError(w, http.StatusBadRequest, "maximum 100 events per batch")
		return
	}

	results := make([]map[string]any, 0, len(req.Events))
	successCount := 0
	failureCount := 0

	for i, event := range req.Events {
		result, _, errMsg := h.processEvent(r.Context(), event, accountID, r)
		eventResult := map[string]any{"index": i}

		if errMsg != "" {
			eventResult["success"] = false
			eventResult["error"] = errMsg
			failureCount++
		} else {
			eventResult["success"] = true
			eventResult["event_id"] = result["event_id"]
			eventResult["queued"] = result["queued"]
			successCount++
		}

		results = append(results, eventResult)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"total":    len(req.Events),
		"success":  successCount,
		"failures": failureCount,
		"results":  results,
	})
}

// authenticate valida a API Key e retorna accountID.
func (h *APIEvents) authenticate(r *http.Request) (accountID int64, err error) {
	apiKey := r.Header.Get("X-API-Key")
	if apiKey == "" {
		return 0, errors.New("missing X-API-Key header")
	}

	key, err := h.apiKeys.FindActiveByPlaintext(r.Context(), apiKey)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return 0, errors.New("invalid API key")
		}
		slog.Error("find api key", "err", err)
		return 0, errors.New("auth error")
	}

	if !key.IsActive() {
		return 0, errors.New("API key revoked")
	}

	_ = h.apiKeys.TouchLastUsed(r.Context(), key.ID)
	return key.AccountID, nil
}
