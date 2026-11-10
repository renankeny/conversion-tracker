package domain

import "time"

// Account representa um cliente do TrackFind. Cada conta é um silo isolado
// de pixels, tracked_users, sent_events e configurações.
type Account struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Enabled     bool   `json:"enabled"`
	OwnerID     *int64 `json:"owner_id,omitempty"`
	PixelsCount int    `json:"pixels_count"`
	UsersCount  int    `json:"users_count"`
	// Override de retenção. NULL => usa env global; 0 => sem retenção; N>0 => N dias.
	SentEventsRetentionDays    *int `json:"sent_events_retention_days,omitempty"`
	EventCountersRetentionDays *int `json:"event_counters_retention_days,omitempty"`
	// Nomes customizados dos eventos na dashboard (cards/funil). Chave é o
	// event_name original (ex: "InitiateCheckout"), valor é o label escolhido
	// pela empresa (ex: "Iniciou Compra").
	EventLabelOverrides map[string]string `json:"event_label_overrides"`
	// EventsFilter guarda os eventos selecionados na página de Eventos, por
	// contexto de pixel. Chave é o pixel_name (ou "" pra "Todos os Pixels"),
	// valor é a lista de event_name marcados pra exibir. Cada empresa tem o
	// seu próprio mapa.
	EventsFilter map[string][]string `json:"events_filter"`
	CreatedAt    time.Time           `json:"created_at"`
	UpdatedAt    time.Time           `json:"updated_at"`
}

// AdminRole controla o nível de acesso do admin no painel.
const (
	RoleSuperAdmin = "superadmin"
	RoleAdmin      = "admin"
	RoleManager    = "manager"
	RoleEmployee   = "employee"
)

// IsValidRole valida se o role string é aceito.
func IsValidRole(role string) bool {
	return role == RoleSuperAdmin || role == RoleAdmin || role == RoleManager || role == RoleEmployee
}
