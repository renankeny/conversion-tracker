package domain

import "time"

// APIKey representa um token server-to-server vinculado a uma conta.
// O plaintext é armazenado apenas como SHA-256; uma única vez é exposto
// no momento da criação para o admin copiar.
type APIKey struct {
	ID         int64      `json:"id"`
	AccountID  int64      `json:"account_id"`
	Name       string     `json:"name"`
	KeyPrefix  string     `json:"key_prefix"`           // "trk_xxxx" (8 chars)
	KeyHash    string     `json:"-"`                    // nunca expõe
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// IsActive retorna true se a key não foi revogada.
func (k *APIKey) IsActive() bool {
	return k != nil && k.RevokedAt == nil
}
