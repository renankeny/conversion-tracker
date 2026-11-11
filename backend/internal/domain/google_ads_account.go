package domain

import "time"

// GoogleAdsConversionAction representa uma ConversionAction configurada
// no painel do Google Ads. Cada ID corresponde a um tipo de conversao
// (Vendas, Lead, Cadastro, etc.) e cada upload de conversao offline
// precisa apontar pra UMA action especifica via conversion_action.
type GoogleAdsConversionAction struct {
	// EventType e o nome do evento TrackFind (ex: "Purchase", "Lead").
	// Usado pelo worker pra escolher qual action disparar quando um
	// evento desse tipo entra na fila.
	EventType string `json:"event_type"`
	// ConversionActionID e o ID interno do Google Ads (numero) da action.
	// Vai no payload como "customers/{cid}/conversionActions/{ID}".
	ConversionActionID string `json:"conversion_action_id"`
	// Name e label apenas pra UI.
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

// GoogleAdsAccount representa uma conta de Google Ads conectada (via OAuth)
// que pode receber conversoes offline. Diferente de Meta/TikTok que sao
// pixel-based, Google Ads trabalha por GCLID — cada conversao carrega
// seu proprio click identifier — entao a conta vive separada do pixel
// e pode ser referenciada por varios pixels.
type GoogleAdsAccount struct {
	ID              int64  `json:"id"`
	AccountID       int64  `json:"account_id"`
	Name            string `json:"name"`
	CustomerID      string `json:"customer_id"`
	LoginCustomerID string `json:"login_customer_id"`
	// ConversionActionID legado (single-action) — mantido pra compat
	// reversa mas nao mais usado pelo worker. Use ConversionActions.
	ConversionActionID string `json:"-"`
	// ConversionActions e o array de mapeamentos event_type ->
	// conversion_action_id. O worker escolhe qual disparar com base
	// no nome do evento que entrou na fila.
	ConversionActions []GoogleAdsConversionAction `json:"conversion_actions"`
	// RefreshToken nunca aparece em JSON publico (campo nao exportado no PublicView).
	RefreshToken    string    `json:"-"`
	OAuthEmail      string    `json:"oauth_email"`
	OAuthPictureURL string    `json:"oauth_picture_url"`
	Enabled         bool      `json:"enabled"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// HasCredentials indica se a conta possui OAuth conectado e CustomerID configurado,
// permitindo disparos tanto pelas actions da conta quanto por override explícito do CRM.
func (a GoogleAdsAccount) HasCredentials() bool {
	return a.RefreshToken != "" && a.CustomerID != ""
}

// IsConnected indica se a conta tem credenciais validas pra enviar
// conversoes (refresh_token + customer_id e pelo menos uma action ativa).
func (a GoogleAdsAccount) IsConnected() bool {
	if !a.HasCredentials() {
		return false
	}
	return a.HasActionForAnyEvent()
}

// HasActionForAnyEvent retorna true se existe ao menos uma
// ConversionAction ativa configurada (algum evento sera enviado).
func (a GoogleAdsAccount) HasActionForAnyEvent() bool {
	for _, ca := range a.ConversionActions {
		if ca.Enabled && ca.ConversionActionID != "" && ca.EventType != "" {
			return true
		}
	}
	return false
}

// FindActionsForEvent retorna TODAS as ConversionActions ativas para
// um event_type especifico. Permite disparar varias conversoes Google
// pelo mesmo evento (ex: InitiateCheckout dispara uma Action "Begin
// Checkout" e outra "Lead Engajado" simultaneamente).
func (a GoogleAdsAccount) FindActionsForEvent(eventType string) []*GoogleAdsConversionAction {
	out := make([]*GoogleAdsConversionAction, 0)
	isLeadAlias := eventType == EventLead || eventType == EventLeadSubmitted
	for i := range a.ConversionActions {
		ca := &a.ConversionActions[i]
		if !ca.Enabled || ca.ConversionActionID == "" {
			continue
		}
		if ca.EventType == eventType || (isLeadAlias && (ca.EventType == EventLead || ca.EventType == EventLeadSubmitted)) {
			out = append(out, ca)
		}
	}
	return out
}

// FindActionForEvent retorna a PRIMEIRA ConversionAction ativa para um
// event_type. Util pro fluxo de teste (UI escolhe uma especifica). Em
// producao, prefira FindActionsForEvent que dispara todas.
func (a GoogleAdsAccount) FindActionForEvent(eventType string) *GoogleAdsConversionAction {
	actions := a.FindActionsForEvent(eventType)
	if len(actions) == 0 {
		return nil
	}
	return actions[0]
}

// PublicView mascara campos sensiveis pro painel.
func (a GoogleAdsAccount) PublicView() map[string]any {
	return map[string]any{
		"id":                 a.ID,
		"account_id":         a.AccountID,
		"name":               a.Name,
		"customer_id":        a.CustomerID,
		"login_customer_id":  a.LoginCustomerID,
		"conversion_actions": a.ConversionActions,
		"oauth_email":        a.OAuthEmail,
		"oauth_picture_url":  a.OAuthPictureURL,
		"picture_url":        a.OAuthPictureURL,
		"oauth_connected":    a.RefreshToken != "",
		"enabled":            a.Enabled,
		"connected":          a.IsConnected(),
		"created_at":         a.CreatedAt,
		"updated_at":         a.UpdatedAt,
	}
}
