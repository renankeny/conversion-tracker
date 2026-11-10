package handlers

import (
	"context"

	"github.com/renankeny/conversion-tracker/backend/internal/auth"
)

// TenantScope encapsula o filtro multi-tenant aplicado às queries.
type TenantScope struct {
	IsOwner     bool  // se true, vê tudo quando AccountID == 0
	CanOverride bool  // se true (owner/admin/manager humano), pode alternar entre empresas próprias
	AccountID   int64 // quando > 0, restringe queries a esta empresa (-1 bloqueia fallback)
}

// TenantFromContext extrai o escopo de acesso a partir das claims autenticadas.
// Fail-closed: ausência de claims nunca concede privilégio de owner.
func TenantFromContext(ctx context.Context) TenantScope {
	c, ok := auth.ClaimsFrom(ctx)
	if !ok || c == nil {
		return TenantScope{IsOwner: false, CanOverride: false, AccountID: -1}
	}
	if c.IsOwner() {
		return TenantScope{IsOwner: true, CanOverride: true, AccountID: c.AccountID}
	}
	accID := c.AccountID
	if accID <= 0 {
		accID = -1 // previne bypass (0) no banco para não-owners sem conta selecionada
	}
	canOverride := !c.IsAPIKey() && (c.Role == "admin" || c.Role == "manager")
	return TenantScope{IsOwner: false, CanOverride: canOverride, AccountID: accID}
}

// EffectiveAccountID retorna o account_id efetivo. Apenas owners globais podem
// aplicar override arbitrário sem validação prévia de posse no banco.
func (t TenantScope) EffectiveAccountID(override int64) int64 {
	if t.IsOwner && override > 0 {
		return override
	}
	if t.AccountID > 0 {
		return t.AccountID
	}
	if t.IsOwner {
		return 1
	}
	return -1
}
