export type CampaignRuleKind = 'metric' | 'schedule' | 'reactivate_late_sale'

export type CampaignRuleActionType =
  | 'pause'
  | 'activate'
  | 'increase_budget_pct'
  | 'increase_budget_fixed'
  | 'decrease_budget_pct'
  | 'decrease_budget_fixed'
  | 'set_budget'
  | 'notify_only'

export type CampaignRuleOperator = '>' | '>=' | '<' | '<=' | '==' | '!='

export type CampaignRulePeriod = 'today' | 'yesterday' | '3d' | '7d' | '14d' | '30d'

export type CampaignRuleLevel = 'campaign' | 'adset' | 'ad'

export type CampaignRuleTargetStatus = 'ACTIVE' | 'PAUSED' | 'ALL'

export type CampaignRuleLogStatus = 'executed' | 'simulated' | 'failed' | 'undone'

export interface CampaignRuleCondition {
  metric: string
  operator: CampaignRuleOperator
  value: number
}

export interface CampaignRule {
  id: number
  account_id: number
  name: string
  description: string
  template_key?: string
  enabled: boolean
  dry_run: boolean
  platform: 'all' | 'meta' | 'google'
  pixel_name: string
  level: CampaignRuleLevel
  attribution_mode: 'last_click' | 'first_click'
  rule_kind: CampaignRuleKind
  period_preset: CampaignRulePeriod
  conditions: CampaignRuleCondition[]
  action_type: CampaignRuleActionType
  action_value: number
  min_budget: number
  max_budget: number
  name_contains: string
  name_excludes: string
  target_status: CampaignRuleTargetStatus
  frequency_minutes: number
  min_hour_sp: number
  schedule_time: string
  schedule_days: number[]
  cooldown_minutes: number
  notify_push: boolean
  notify_whatsapp: boolean
  whatsapp_phone: string
  last_run_at?: string | null
  last_triggered_at?: string | null
  last_error?: string
  trigger_count: number
  created_at: string
  updated_at: string
}

export type CampaignRulePayload = Omit<
  CampaignRule,
  'id' | 'account_id' | 'last_run_at' | 'last_triggered_at' | 'last_error' | 'trigger_count' | 'created_at' | 'updated_at'
>

export interface CampaignRuleStats {
  total: number
  active: number
  dry_run: number
  executed_today: number
  simulated_today: number
}

export interface CampaignRuleMetricsSnapshot {
  spend: number
  impressions: number
  clicks: number
  ctr: number
  cpc: number
  leads: number
  conversations: number
  ai_qualified: number
  gross_sales: number
  refund_count: number
  net_sales: number
  net_revenue: number
  net_profit: number
  refund_rate: number
  real_cpa: number
  cost_per_lead: number
  cost_per_qualified: number
  net_roas: number
  roi: number
  daily_budget: number
}

export interface CampaignRuleLog {
  id: number
  account_id: number
  rule_id?: number | null
  rule_name: string
  platform: string
  level: CampaignRuleLevel
  entity_subtype?: string
  object_id: string
  parent_id?: string
  object_name: string
  pixel_name?: string
  currency: string
  action_type: CampaignRuleActionType
  dry_run: boolean
  status: CampaignRuleLogStatus
  previous_status?: string
  new_status?: string
  previous_budget: number
  new_budget: number
  metrics_snapshot: CampaignRuleMetricsSnapshot
  reason: string
  error_message?: string
  undone_at?: string | null
  created_at: string
}

export interface CampaignRuleRunSummary {
  rule_id: number
  rule_name: string
  dry_run: boolean
  evaluated_count: number
  matched_count: number
  executed_count: number
  skipped_count: number
  failed_count: number
  warnings?: string[]
  logs?: CampaignRuleLog[]
}
