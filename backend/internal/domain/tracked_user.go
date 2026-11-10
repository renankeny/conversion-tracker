package domain

import (
	"encoding/json"
	"time"
)

// TrackedUser representa um visitante/lead identificado pelo external_id
// (gerado no client com 10 caracteres [A-Z0-9], igual ao sistema legado).
//
// JSON tags em snake_case pra consistência com o resto da API e com
// as convenções esperadas pelo frontend.
type TrackedUser struct {
	ID         int64  `json:"id"`
	ExternalID string `json:"external_id"`
	IDWpp      string `json:"id_wpp"`
	PixelName  string `json:"pixel_name"`
	// AccountID vincula o tracked_user à conta do pixel (multi-tenant).
	AccountID int64 `json:"account_id"`
	// GroupID liga esse registro a outros external_id da mesma pessoa
	// (telefone/email batendo — navegador diferente, mesmo cliente). NULL
	// até um vínculo ser detectado. Uso interno só — nunca serializado.
	GroupID *int64 `json:"-"`

	FBP             string `json:"fbp"`
	FBC             string `json:"fbc"`
	GCLID           string `json:"gclid"`
	WBraid          string `json:"wbraid"`
	GBraid          string `json:"gbraid"`
	TTClid          string `json:"ttclid"`
	TTP             string `json:"ttp"`
	ClientIP        string `json:"client_ip"`
	ClientUserAgent string `json:"client_user_agent"`

	// GA4
	GA4ClientID  string `json:"ga4_client_id"`
	GA4SessionID string `json:"ga4_session_id"`

	Country    string `json:"country"`
	State      string `json:"state"`
	City       string `json:"city"`
	PostalCode string `json:"postal_code"`

	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`

	URL      string `json:"url"`
	FirstURL string `json:"first_url"`

	FirstUTMSource   string `json:"first_utm_source"`
	FirstUTMMedium   string `json:"first_utm_medium"`
	FirstUTMCampaign string `json:"first_utm_campaign"`
	FirstUTMContent  string `json:"first_utm_content"`
	FirstUTMTerm     string `json:"first_utm_term"`

	LastUTMSource   string `json:"last_utm_source"`
	LastUTMMedium   string `json:"last_utm_medium"`
	LastUTMCampaign string `json:"last_utm_campaign"`
	LastUTMContent  string `json:"last_utm_content"`
	LastUTMTerm     string `json:"last_utm_term"`

	// Campos WhatsApp (uazAPI Premium)
	CtwaClid    string `json:"ctwa_clid"`     // Click to WhatsApp ID
	WaChatID    string `json:"wa_chat_id"`    // ID do chat
	WaName      string `json:"wa_name"`       // Nome do contato
	WaAdID      string `json:"wa_ad_id"`      // ID do criativo
	WaSourceApp string `json:"wa_source_app"` // fb_messenger, instagram, etc
	WaLid       string `json:"wa_lid"`        // Linked ID (JID @lid) — fallback quando o WhatsApp não expõe o telefone real

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Contadores de Engajamento e Atividade
	PageViewCount int       `json:"pageview_count"`
	CheckoutCount int       `json:"checkout_count"`
	LastSeenAt    time.Time `json:"last_seen_at"`

	// Classificação de Tráfego e Auditoria Heurística
	Referrer        string          `json:"referrer"`
	TrafficType     string          `json:"traffic_type"`
	TrafficScore    int             `json:"traffic_score"`
	TrafficCategory string          `json:"traffic_category"`
	TrafficRules    json.RawMessage `json:"traffic_rules"`

	// Campos Calculados Dinamicamente
	LTV            float64 `json:"ltv"`
	TimeToPurchase string  `json:"time_to_purchase"`
}

// LinkedTrackingFields agrupa os identificadores de canal (cookies/clicks)
// resolvidos a partir do valor mais recente não-vazio entre TODOS os
// tracked_users do mesmo GroupID de um external_id — não só do próprio
// registro. Usado na hora de montar o envio pra Meta/Google/TikTok/GA4,
// pra não perder um fbc/gclid que ficou salvo num navegador diferente do
// que gerou o evento atual. Ver UserRepo.LinkedFields.
type LinkedTrackingFields struct {
	FBP          string
	FBC          string
	GCLID        string
	WBraid       string
	GBraid       string
	TTClid       string
	TTP          string
	GA4ClientID  string
	GA4SessionID string
}
