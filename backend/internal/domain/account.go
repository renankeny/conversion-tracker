package domain

import "time"

// Account representa um cliente do TrackFind. Cada conta é um silo isolado
// de pixels, tracked_users, sent_events e configurações.
type Account struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AdminRole controla o nível de acesso do admin no painel.
//
//	"owner"  → acesso global (vê e edita tudo de todas as contas)
//	"member" → acesso restrito a uma conta específica (account_id)
const (
	RoleOwner  = "owner"
	RoleMember = "member"
)

// IsValidRole valida se o role string é aceito.
func IsValidRole(role string) bool {
	return role == RoleOwner || role == RoleMember
}
