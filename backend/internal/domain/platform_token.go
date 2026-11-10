package domain

import "time"

// PlatformToken guarda os tokens/secrets de webhook de integrações
// como Hotmart, Yampi, Kirvano para validar a autenticidade dos eventos.
type PlatformToken struct {
	ID           int64     `json:"id"`
	AccountID    int64     `json:"account_id"`
	PlatformName string    `json:"platform_name"`
	TokenSecret  string    `json:"token_secret"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
