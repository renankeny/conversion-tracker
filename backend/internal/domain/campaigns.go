package domain

// PixelFinancialConfig armazena as taxas e custos operacionais por Pixel para
// cálculo de Lucro Líquido Real e ROI na página Campanhas.
type PixelFinancialConfig struct {
	GatewayFeePct   float64 `json:"gateway_fee_pct"`
	GatewayFixedFee float64 `json:"gateway_fixed_fee"`
	TaxPct          float64 `json:"tax_pct"`
	ProductCost     float64 `json:"product_cost"`
}

// CampaignRow representa uma linha normalizada (Campanha, Conjunto/Grupo ou Anúncio)
// unificando dados da Meta Graph API v26.0, Google Ads API v26 e atribuição TrackFind.
type CampaignRow struct {
	ID              string `json:"id"`
	Platform        string `json:"platform"`                   // "meta" | "google" | "unattributed"
	Level           string `json:"level"`                      // "campaign" | "adset" | "ad"
	EntitySubtype   string `json:"entity_subtype,omitempty"`   // "adset" | "ad_group" | "asset_group" | "ad"
	AccountRefID    int64  `json:"account_ref_id,omitempty"`   // meta_account_id ou google_ads_account_id
	AdAccountID     string `json:"ad_account_id,omitempty"`    // act_xxx ou customer_id
	LoginCustomerID string `json:"login_customer_id,omitempty"`
	Currency        string `json:"currency,omitempty"`

	CampaignID   string `json:"campaign_id,omitempty"`
	CampaignName string `json:"campaign_name,omitempty"`
	AdSetID      string `json:"adset_id,omitempty"`
	AdSetName    string `json:"adset_name,omitempty"`
	Name         string `json:"name"`
	ThumbnailURL string `json:"thumbnail_url,omitempty"`

	Status          string `json:"status"`                     // "ACTIVE" | "PAUSED" | "ARCHIVED" | "UNKNOWN"
	EffectiveStatus string `json:"effective_status,omitempty"` // status detalhado da plataforma
	Objective       string `json:"objective,omitempty"`
	ChannelType     string `json:"channel_type,omitempty"`
	BudgetID        string `json:"budget_id,omitempty"` // Google campaign_budget.id
	BudgetShared    bool   `json:"budget_shared,omitempty"`
	BudgetLevel     string `json:"budget_level,omitempty"` // "campaign" | "adset" | ""
	DailyBudget     float64 `json:"daily_budget"`
	LifetimeBudget  float64 `json:"lifetime_budget"`
	CanEditStatus   bool    `json:"can_edit_status"`
	CanEditBudget   bool    `json:"can_edit_budget"`

	// Métricas de Tráfego (Plataforma)
	Spend       float64 `json:"spend"`
	Impressions int64   `json:"impressions"`
	Clicks      int64   `json:"clicks"`
	CTR         float64 `json:"ctr"`
	CPC         float64 `json:"cpc"`
	CPM         float64 `json:"cpm"`

	// Métricas Reais Rastreadas (TrackFind)
	Leads            int64   `json:"leads"`
	Conversations    int64   `json:"conversations"`
	AIQualified      int64   `json:"ai_qualified"`
	GrossSales       int64   `json:"gross_sales"`
	RefundCount      int64   `json:"refund_count"`
	ChargebackCount  int64   `json:"chargeback_count"`
	NetSales         int64   `json:"net_sales"`
	GrossRevenue     float64 `json:"gross_revenue"`
	RefundAmount     float64 `json:"refund_amount"`
	NetRevenue       float64 `json:"net_revenue"`
	OperationalCosts float64 `json:"operational_costs"`
	NetProfit        float64 `json:"net_profit"`

	// Indicadores derivados
	RefundRate  float64 `json:"refund_rate"`
	RealCPA     float64 `json:"real_cpa"`
	CostPerLead float64 `json:"cost_per_lead"`
	GrossROAS   float64 `json:"gross_roas"`
	NetROAS     float64 `json:"net_roas"`
	ROI         float64 `json:"roi"`
	MatchSource string  `json:"match_source,omitempty"` // "id" | "utm" | "unattributed"
}

// CampaignsKpiSummary consolida os totais da barra executiva (KPI Strip).
type CampaignsKpiSummary struct {
	Spend            float64 `json:"spend"`
	Impressions      int64   `json:"impressions"`
	Clicks           int64   `json:"clicks"`
	Leads            int64   `json:"leads"`
	Conversations    int64   `json:"conversations"`
	AIQualified      int64   `json:"ai_qualified"`
	GrossSales       int64   `json:"gross_sales"`
	RefundCount      int64   `json:"refund_count"`
	ChargebackCount  int64   `json:"chargeback_count"`
	NetSales         int64   `json:"net_sales"`
	GrossRevenue     float64 `json:"gross_revenue"`
	RefundAmount     float64 `json:"refund_amount"`
	NetRevenue       float64 `json:"net_revenue"`
	OperationalCosts float64 `json:"operational_costs"`
	NetProfit        float64 `json:"net_profit"`
	RefundRate       float64 `json:"refund_rate"`
	RealCPA          float64 `json:"real_cpa"`
	GrossROAS        float64 `json:"gross_roas"`
	NetROAS          float64 `json:"net_roas"`
	ROI              float64 `json:"roi"`
}

// CampaignsOverviewResponse é o contrato da API GET /admin/campaigns/overview.
type CampaignsOverviewResponse struct {
	Platform        string               `json:"platform"`
	AttributionMode string               `json:"attribution_mode"` // "last_click" | "first_click"
	Since           string               `json:"since"`
	Until           string               `json:"until"`
	Currency        string               `json:"currency"`
	PixelFinancials PixelFinancialConfig `json:"pixel_financials"`
	Summary         CampaignsKpiSummary  `json:"summary"`
	Campaigns       []CampaignRow        `json:"campaigns"`
	AdSets          []CampaignRow        `json:"adsets"`
	Ads             []CampaignRow        `json:"ads"`
	Warnings        []string             `json:"warnings,omitempty"`
	Cached          bool                 `json:"cached"`
}

// CampaignStatusMutateInput é o payload para pausar/ativar Campanha, Conjunto/Grupo ou Anúncio.
type CampaignStatusMutateInput struct {
	Platform        string `json:"platform"`         // "meta" | "google"
	Level           string `json:"level"`            // "campaign" | "adset" | "ad"
	EntitySubtype   string `json:"entity_subtype"`   // "adset" | "ad_group" | "asset_group" | "ad"
	ObjectID        string `json:"object_id"`        // ID do objeto
	ParentID        string `json:"parent_id"`        // ad_group_id (obrigatório para Google adGroupAds:mutate)
	Status          string `json:"status"`           // "ACTIVE" | "PAUSED"
	PixelName       string `json:"pixel_name"`       // opcional (para resolver conta vinculada)
	AccountRefID    int64  `json:"account_ref_id"`   // meta_account_id ou google_ads_account_id
	AdAccountID     string `json:"ad_account_id"`    // act_xxx ou customer_id
	LoginCustomerID string `json:"login_customer_id"`
}

// CampaignBudgetMutateInput é o payload para alterar o orçamento diário.
type CampaignBudgetMutateInput struct {
	Platform        string  `json:"platform"`          // "meta" | "google"
	Level           string  `json:"level"`             // "campaign" | "adset"
	ObjectID        string  `json:"object_id"`         // Meta campaign/adset ID ou Google campaign ID
	BudgetID        string  `json:"budget_id"`         // Google campaign_budget.id
	DailyBudget     float64 `json:"daily_budget"`      // valor em moeda real (ex: 150.00)
	Currency        string  `json:"currency"`          // BRL, USD, etc.
	PixelName       string  `json:"pixel_name"`
	AccountRefID    int64   `json:"account_ref_id"`
	AdAccountID     string  `json:"ad_account_id"`
	LoginCustomerID string  `json:"login_customer_id"`
}
