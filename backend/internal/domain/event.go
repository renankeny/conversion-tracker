package domain

// Eventos suportados pela API.
//
// Categorias:
//   - "Init"           → handshake do pixel.js (não envia evento à Meta, só hidrata contexto).
//   - "UpdateUserData" → atualiza fbp/fbc do tracked user (não envia evento à Meta).
//   - Standard         → enviado como track normal pra Meta CAPI.
//   - Custom           → enviado como trackCustom pra Meta CAPI (Scroll, Timer, ViewVideo).
const (
	// Internos
	EventInit           = "Init"
	EventUpdateUserData = "UpdateUserData"

	// Standard (Meta CAPI track)
	EventPageView             = "PageView"
	EventViewContent          = "ViewContent"
	EventLead                 = "Lead"
	EventContact              = "Contact"
	EventInitiateCheckout     = "InitiateCheckout"
	EventAddPaymentInfo       = "AddPaymentInfo"
	EventAddShippingInfo      = "AddShippingInfo"
	EventAddToCart            = "AddToCart"
	EventAddToWishlist        = "AddToWishlist"
	EventCompleteRegistration = "CompleteRegistration"
	EventPurchase             = "Purchase"

	// Business Messaging (action_source=business_messaging — CTWA/WhatsApp).
	// "Lead" não é um evento suportado nessa superfície; o nome correto
	// documentado pela Meta é "LeadSubmitted".
	EventLeadSubmitted = "LeadSubmitted"
	EventSchedule      = "Schedule"

	// Custom (Meta CAPI trackCustom)
	EventScroll25    = "Scroll_25"
	EventScroll50    = "Scroll_50"
	EventScroll75    = "Scroll_75"
	EventScroll90    = "Scroll_90"
	EventTimer1Min   = "Timer_1min"
	EventPlayVideo   = "PlayVideo"
	EventViewVideo25 = "ViewVideo_25"
	EventViewVideo50 = "ViewVideo_50"
	EventViewVideo75 = "ViewVideo_75"
	EventViewVideo90 = "ViewVideo_90"
)

// IsValidEventType valida se o eventType é aceito pelo /events/collect.
// Cobre eventos internos, standard e custom.
func IsValidEventType(name string) bool {
	switch name {
	case EventInit, EventUpdateUserData,
		EventPageView, EventViewContent, EventLead, EventContact,
		EventInitiateCheckout, EventAddPaymentInfo, EventAddShippingInfo, EventAddToCart,
		EventAddToWishlist, EventCompleteRegistration, EventPurchase,
		EventLeadSubmitted, EventSchedule,
		EventScroll25, EventScroll50, EventScroll75, EventScroll90,
		EventTimer1Min, EventPlayVideo,
		EventViewVideo25, EventViewVideo50, EventViewVideo75, EventViewVideo90:
		return true
	}
	return false
}

// IsInternalEvent retorna true se o evento NÃO deve ser enviado pra Meta CAPI
// (Init e UpdateUserData fazem só side effects no servidor).
func IsInternalEvent(name string) bool {
	return name == EventInit || name == EventUpdateUserData
}

// SendEventInput é o payload normalizado pronto pra ser enviado pra Meta.
//
// JSON tags são usadas na serialização do task do asynq (passa entre API e
// worker). Mantém snake_case por consistência.
type SendEventInput struct {
	PixelName string `json:"pixel_name"`
	EventName string `json:"event_name"`
	EventID   string `json:"event_id"`
	EventTime int64  `json:"event_time"`
	SourceURL string `json:"source_url"`

	// Custom data (Meta CAPI)
	ContentID   string   `json:"content_id"`             // mapeado pra content_ids[0]
	ContentType string   `json:"content_type,omitempty"` // "product" ou "product_group"
	ContentName string   `json:"content_name,omitempty"`
	Value       *float64 `json:"value,omitempty"`
	Currency    string   `json:"currency,omitempty"`
	OrderID     string   `json:"order_id,omitempty"` // recomendado pra Purchase
	NumItems    *int     `json:"num_items,omitempty"`

	// Business Messaging (WhatsApp via Evolution GO). Quando preenchidos,
	// buildPayload troca action_source de "website" pra ActionSource e
	// anexa messaging_channel/ctwa_clid/page_id ao payload da Meta CAPI.
	ActionSource     string `json:"action_source,omitempty"`
	MessagingChannel string `json:"messaging_channel,omitempty"`
	CtwaClid         string `json:"ctwa_clid,omitempty"`
	PageID           string `json:"page_id,omitempty"`

	// User data
	ExternalID      string `json:"external_id"`
	IDWpp           string `json:"id_wpp,omitempty"`     // ID WhatsApp (subscription_id no Meta)
	ClientIP        string `json:"client_ip"`
	ClientUserAgent string `json:"client_user_agent"`
	FBP             string `json:"fbp"`
	FBC             string `json:"fbc"`
	Email           string `json:"email"`
	Phone           string `json:"phone"`
	FirstName       string `json:"first_name"`
	LastName        string `json:"last_name"`
	Country         string `json:"country"`
	State           string `json:"state"`
	City            string `json:"city"`
	PostalCode      string `json:"postal_code"`

	// TikTok-specific user data (não é hasheado; tokens browser do TikTok)
	TTClid string `json:"ttclid,omitempty"` // TikTok Click ID (?ttclid=...)
	TTP    string `json:"ttp,omitempty"`    // Cookie _ttp do tracker TikTok
	// Referrer é usado pelo TikTok no campo page.referrer
	Referrer string `json:"referrer,omitempty"`

	// Google Ads click identifiers - exatamente um vai com a conversao
	// (Google escolhe gclid > gbraid > wbraid pela ordem de prioridade).
	GCLID  string `json:"gclid,omitempty"`
	WBraid string `json:"wbraid,omitempty"`
	GBraid string `json:"gbraid,omitempty"`

	// GA4 Measurement Protocol
	GA4ClientID  string `json:"ga4_client_id,omitempty"`
	GA4SessionID string `json:"ga4_session_id,omitempty"`

	// CRM Overrides
	OverrideGadsAccountID *int64 `json:"override_gads_account_id,omitempty"`
	OverrideGadsActionID  string `json:"override_gads_action_id,omitempty"`

	// Agrupamento de Conversões (Order Bump & Upsell)
	Gateway         string           `json:"gateway,omitempty"`   // gateway de origem (ex: "hotmart", "kiwify")
	ItemType        string           `json:"item_type,omitempty"` // "main", "order_bump", "upsell"
	AggregatedCount int              `json:"aggregated_count,omitempty"`
	AggregatedItems []AggregatedItem `json:"aggregated_items,omitempty"`
}

// AggregatedItem representa um item ou venda individual agrupado dentro de uma janela de tempo.
type AggregatedItem struct {
	OrderID      string  `json:"order_id,omitempty"`
	ProductTitle string  `json:"product_title,omitempty"`
	ItemType     string  `json:"item_type,omitempty"` // "main", "order_bump", "upsell"
	Value        float64 `json:"value"`
	Gateway      string  `json:"gateway,omitempty"`
	CreatedAt    int64   `json:"created_at"`
}
