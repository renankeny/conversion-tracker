package domain

import (
	"strings"
	"time"
)

const (
	APIScopeAll       = "*"
	APIScopeEvents    = "events"
	APIScopeCRM       = "crm"
	APIScopeUsers     = "users"
	APIScopeWhatsApp  = "whatsapp"
	APIScopePixels    = "pixels"
	APIScopeAnalytics = "analytics"
	APIScopeSales     = "sales"
	APIScopeWebhooks  = "webhooks"
)

var AllAPIScopes = []string{
	APIScopeEvents,
	APIScopeCRM,
	APIScopeUsers,
	APIScopeWhatsApp,
	APIScopePixels,
	APIScopeAnalytics,
	APIScopeSales,
	APIScopeWebhooks,
}

var validScopeSet = map[string]bool{
	APIScopeEvents:    true,
	APIScopeCRM:       true,
	APIScopeUsers:     true,
	APIScopeWhatsApp:  true,
	APIScopePixels:    true,
	APIScopeAnalytics: true,
	APIScopeSales:     true,
	APIScopeWebhooks:  true,
}

// APIKey representa uma credencial unificada (API REST + Servidor MCP) vinculada a uma empresa.
type APIKey struct {
	ID           int64      `json:"id"`
	AccountID    int64      `json:"account_id"`
	Name         string     `json:"name"`
	KeyPrefix    string     `json:"key_prefix"`
	KeyHash      string     `json:"-"`
	KeyEncrypted string     `json:"-"`
	Plaintext    string     `json:"plaintext,omitempty"`
	Scopes       []string   `json:"scopes"`
	ReadOnly     bool       `json:"read_only"`
	LastUsedAt   *time.Time `json:"last_used_at,omitempty"`
	RevokedAt    *time.Time `json:"revoked_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// IsActive retorna true se a key não foi revogada.
func (k *APIKey) IsActive() bool {
	return k != nil && k.RevokedAt == nil
}

// HasScope verifica se a chave possui permissão para o módulo informado.
func (k *APIKey) HasScope(scope string) bool {
	if k == nil {
		return false
	}
	if scope == "" || len(k.Scopes) == 0 {
		return true
	}
	for _, s := range k.Scopes {
		if s == APIScopeAll || s == scope {
			return true
		}
	}
	return false
}

// NormalizeAPIKeyScopes sanitiza a lista de escopos informada pelo painel.
// Se `allowEmpty` for false e nenhum escopo válido for enviado, retorna erro.
func NormalizeAPIKeyScopes(raw []string, defaultAllIfNil bool) ([]string, bool) {
	if raw == nil && defaultAllIfNil {
		return []string{APIScopeAll}, true
	}
	seen := make(map[string]bool, len(AllAPIScopes))
	out := make([]string, 0, len(AllAPIScopes))
	for _, item := range raw {
		s := strings.ToLower(strings.TrimSpace(item))
		if s == APIScopeAll {
			return []string{APIScopeAll}, true
		}
		if validScopeSet[s] && !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		return nil, false
	}
	if len(out) == len(AllAPIScopes) {
		return []string{APIScopeAll}, true
	}
	return out, true
}
