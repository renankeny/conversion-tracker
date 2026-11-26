export interface PixelFinancialConfig {
  gateway_fee_pct: number
  gateway_fixed_fee: number
  tax_pct: number
  product_cost: number
}

export type CampaignPlatform = 'all' | 'meta' | 'google'
export type CampaignAttributionMode = 'last_click' | 'first_click'
export type CampaignLevel = 'campaign' | 'adset' | 'ad'

export interface CampaignRow {
  id: string
  platform: 'meta' | 'google' | 'unattributed'
  level: CampaignLevel
  entity_subtype?: string
  account_ref_id?: number
  ad_account_id?: string
  login_customer_id?: string
  currency?: string

  campaign_id?: string
  campaign_name?: string
  adset_id?: string
  adset_name?: string
  name: string
  thumbnail_url?: string

  status: 'ACTIVE' | 'PAUSED' | 'ARCHIVED' | 'UNKNOWN'
  effective_status?: string
  objective?: string
  channel_type?: string
  budget_id?: string
  budget_shared?: boolean
  budget_level?: string
  daily_budget: number
  lifetime_budget: number
  can_edit_status: boolean
  can_edit_budget: boolean

  spend: number
  impressions: number
  clicks: number
  ctr: number
  cpc: number
  cpm: number

  leads: number
  conversations: number
  ai_qualified: number
  gross_sales: number
  refund_count: number
  chargeback_count: number
  net_sales: number
  gross_revenue: number
  refund_amount: number
  net_revenue: number
  operational_costs: number
  net_profit: number

  refund_rate: number
  real_cpa: number
  cost_per_lead: number
  gross_roas: number
  net_roas: number
  roi: number
  match_source?: string
}

export interface CampaignsKpiSummary {
  spend: number
  impressions: number
  clicks: number
  leads: number
  conversations: number
  ai_qualified: number
  gross_sales: number
  refund_count: number
  chargeback_count: number
  net_sales: number
  gross_revenue: number
  refund_amount: number
  net_revenue: number
  operational_costs: number
  net_profit: number
  refund_rate: number
  real_cpa: number
  gross_roas: number
  net_roas: number
  roi: number
}

export interface CampaignsOverviewResponse {
  platform: CampaignPlatform
  attribution_mode: CampaignAttributionMode
  since: string
  until: string
  currency: string
  pixel_financials: PixelFinancialConfig
  summary: CampaignsKpiSummary
  campaigns: CampaignRow[]
  adsets: CampaignRow[]
  ads: CampaignRow[]
  warnings?: string[]
  cached: boolean
}
