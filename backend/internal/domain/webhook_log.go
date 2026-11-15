package domain

import "time"

// WebhookLog é o registro de um webhook recebido de um gateway de pagamento.
// Armazena o purchaseInput normalizado (pré-lookup de usuário) para
// possibilitar replay manual pelo painel admin.
type WebhookLog struct {
	ID         int64     `json:"id"`
	AccountID  int64     `json:"account_id"`
	Source     string    `json:"source"`
	PixelName  string    `json:"pixel_name"`
	ExternalID string    `json:"external_id"`
	EventName  string    `json:"event_name"`
	EventID    string    `json:"event_id"`
	Status     string    `json:"status"` // 'queued' | 'buffered' | 'failed' | 'replayed'
	Error      string    `json:"error,omitempty"`
	CanReplay  bool      `json:"can_replay"`
	ReceivedAt time.Time `json:"received_at"`
}

// WebhookPurchaseInput é a versão serializável do purchaseInput interno
// dos handlers de webhook. Armazenada em webhook_logs.input_json para
// possibilitar replay sem re-parsing do payload original.
type WebhookPurchaseInput struct {
	GatewayName       string  `json:"gateway_name"`
	EventName         string  `json:"event_name,omitempty"`
	ExternalID        string  `json:"external_id,omitempty"`
	OrderID           string  `json:"order_id,omitempty"`
	Value             float64 `json:"value,omitempty"`
	Currency          string  `json:"currency,omitempty"`
	FirstName         string  `json:"first_name,omitempty"`
	LastName          string  `json:"last_name,omitempty"`
	Email             string  `json:"email,omitempty"`
	Phone             string  `json:"phone,omitempty"`
	FBC               string  `json:"fbc,omitempty"`
	FBP               string  `json:"fbp,omitempty"`
	AccountID         int64   `json:"account_id,omitempty"`
	PixelQuery        string  `json:"pixel_query,omitempty"`
	AlsoSearchByEmail bool    `json:"also_search_by_email,omitempty"`
	ProductName       string  `json:"product_name,omitempty"`
	ProductID         string  `json:"product_id,omitempty"`
	IsOrderBump       bool    `json:"is_order_bump,omitempty"`
	IsUpsell          bool    `json:"is_upsell,omitempty"`
	OfferType         string  `json:"offer_type,omitempty"`
}
