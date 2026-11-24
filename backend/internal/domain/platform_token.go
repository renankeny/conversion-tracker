package domain

import (
	"strings"
	"time"
)

// PlatformToken guarda os tokens/secrets de webhook de integrações
// como Hotmart, Yampi e Kirvano para validar a autenticidade dos eventos.
type PlatformToken struct {
	ID           int64     `json:"id"`
	AccountID    int64     `json:"account_id"`
	PlatformName string    `json:"platform_name"`
	TokenSecret  string    `json:"token_secret"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// PlatformWebhookStats agrega contadores de webhook_logs por gateway.
type PlatformWebhookStats struct {
	TotalAccepted  int64      `json:"total_accepted"`
	TotalFailed    int64      `json:"total_failed"`
	LastReceivedAt *time.Time `json:"last_received_at,omitempty"`
}

// PlatformSummaryItem consolida estado de token e tráfego para exibição nos cards.
type PlatformSummaryItem struct {
	HasToken       bool       `json:"has_token"`
	MaskedToken    string     `json:"masked_token,omitempty"`
	TokenUpdatedAt *time.Time `json:"token_updated_at,omitempty"`
	TotalAccepted  int64      `json:"total_accepted"`
	TotalFailed    int64      `json:"total_failed"`
	LastReceivedAt *time.Time `json:"last_received_at,omitempty"`
}

var supportedCheckoutPlatforms = map[string]struct{}{
	"hotmart":    {},
	"kiwify":     {},
	"cakto":      {},
	"kirvano":    {},
	"eduzz":      {},
	"monetizze":  {},
	"braip":      {},
	"perfectpay": {},
	"ticto":      {},
	"greenn":     {},
	"lastlink":   {},
	"hubla":      {},
	"guru":       {},
	"pepper":     {},
	"cartpanda":  {},
	"appmax":     {},
	"doppus":     {},
	"vega":       {},
	"yampi":      {},
	"pagtrust":   {},
	"onixpay":    {},
}

// IsSupportedCheckoutPlatform valida se o slug corresponde a um checkout homologado.
func IsSupportedCheckoutPlatform(name string) bool {
	_, ok := supportedCheckoutPlatforms[strings.ToLower(strings.TrimSpace(name))]
	return ok
}

// MaskPlatformSecret mascara um token mantendo apenas os últimos caracteres visíveis.
func MaskPlatformSecret(secret string) string {
	s := strings.TrimSpace(secret)
	if s == "" {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= 4 {
		return "••••••••"
	}
	return "••••••••" + string(runes[len(runes)-4:])
}
