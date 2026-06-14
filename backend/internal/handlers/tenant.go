package handlers

import (
	"context"

	"github.com/renankeny/conversion-tracker/backend/internal/auth"
)

// TenantScope encapsula o filtro multi-tenant aplicado às queries.
type TenantScope struct {
	IsOwner   bool  // se true, vê tudo (não aplica filtro)
	AccountID int64 // se IsOwner=false, restringe a esta conta
}

// TenantFromContext extrai o escopo de acesso a partir das claims do JWT.
// Owners (sem role ou role="owner") veem tudo. Members só sua conta.
func TenantFromContext(ctx context.Context) TenantScope {
	c, ok := auth.ClaimsFrom(ctx)
	if !ok {
		return TenantScope{IsOwner: true} // safe default; só chega aqui se middleware permitiu
	}
	if c.IsOwner() {
		return TenantScope{IsOwner: true}
	}
	return TenantScope{IsOwner: false, AccountID: c.AccountID}
}

// EffectiveAccountID retorna o account_id que deve ser usado em criações
// (CREATE pixel, CREATE url_mapping, etc).
//   - Owner pode passar um override via query/body. Se não passar, usa default 1.
//   - Member sempre usa o account_id dele (override é ignorado por segurança).
func (t TenantScope) EffectiveAccountID(override int64) int64 {
	if t.IsOwner {
		if override > 0 {
			return override
		}
		return 1 // default account
	}
	return t.AccountID
}
