package domain

// DashboardConfig é a configuração editável da dashboard web (cards, funil,
// nomes customizados de evento). Vive por PIXEL — cada pixel tem sua própria
// dashboard, já que os eventos mapeados variam de pixel pra pixel (uma
// empresa com 2 pixels pode querer cards/ordem/nomes diferentes em cada um).
type DashboardConfig struct {
	// EnabledCards filtra quais cards de evento aparecem no topo da dashboard.
	// CardOrder é a ordem de exibição (lista completa do catálogo, na ordem
	// escolhida) — EnabledCards é aplicado como filtro sobre essa ordem.
	EnabledCards []string `json:"enabled_cards"`
	CardOrder    []string `json:"card_order"`

	// EnabledFunnelEvents e FunnelOrder controlam o funil de conversão, de
	// forma independente dos cards.
	EnabledFunnelEvents []string `json:"enabled_funnel_events"`
	FunnelOrder         []string `json:"funnel_order"`

	// EventLabels é o nome customizado por evento (chave = event_name
	// original, ex: "InitiateCheckout" -> "Iniciou Compra"). Vale para
	// cards e funil.
	EventLabels map[string]string `json:"event_labels"`

	// Controles do Portal Público do Cliente (/cliente/{token}):
	// nil equivale a true por padrão (retrocompatível com pixels existentes).
	PublicShowCRM           *bool `json:"public_show_crm,omitempty"`
	PublicAllowCRMEdit      bool  `json:"public_allow_crm_edit,omitempty"`
	PublicShowConversations *bool `json:"public_show_conversations,omitempty"`
	PublicShowSalesGoal     *bool `json:"public_show_sales_goal,omitempty"`
}

func (c DashboardConfig) IsPublicCRMEnabled() bool {
	return c.PublicShowCRM == nil || *c.PublicShowCRM
}

func (c DashboardConfig) IsPublicCRMEditable() bool {
	return c.IsPublicCRMEnabled() && c.PublicAllowCRMEdit
}

func (c DashboardConfig) IsPublicConversationsEnabled() bool {
	return c.PublicShowConversations == nil || *c.PublicShowConversations
}

func (c DashboardConfig) IsPublicSalesGoalEnabled() bool {
	return c.PublicShowSalesGoal == nil || *c.PublicShowSalesGoal
}
