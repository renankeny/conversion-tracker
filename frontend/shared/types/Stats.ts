export interface DashboardStats {
  status_totals: {
    sent: number
    queued: number
    pending?: number
    failed: number
  }
  event_totals: Record<string, number>
  per_day: Array<{
    date: string
    sent: number
    failed: number
    queued: number
    count?: number
  }>
  total_events: number
  tracked_users: number
  total_revenue?: number
  traffic_quality?: TrafficQualityStats
  source_breakdown?: SourceBreakdown
  panel_audit?: PanelAudit
  previous_period?: PreviousPeriodMetrics
  heatmap?: DashboardHeatmap
}

export interface SourceMetrics {
  visitors: number
  leads: number
  purchases: number
  revenue: number
}

export interface SourceBreakdown {
  meta: SourceMetrics
  google: SourceMetrics
}

export interface AuditCounter {
  total: number
  sent: number
  not_sent: number
  meta_total: number
  meta_sent: number
  meta_not_sent: number
  google_total: number
  google_sent: number
  google_not_sent: number
}

export interface PanelAudit {
  ad_messages: AuditCounter
  leads: AuditCounter
  purchases: AuditCounter
}

export interface PreviousPeriodMetrics {
  page_views: number
  ad_messages: number
  leads: number
  purchases: number
  revenue: number
}

export interface HeatmapBucket {
  weekday: number
  hour: number
  count: number
}

export interface DashboardHeatmap {
  messages: HeatmapBucket[]
  leads: HeatmapBucket[]
  sales: HeatmapBucket[]
}

export interface AdPlatformLiveInsights {
  spend: number
  clicks: number
  impressions: number
  messages?: number
  leads?: number
  purchases?: number
  revenue?: number
  conversions?: number
  prev_spend?: number
  prev_clicks?: number
  connected: boolean
}

export interface GoogleKeywordInsight {
  keyword: string
  match_type: string
  campaign_name: string
  clicks: number
  impressions: number
  spend: number
  cpc: number
  leads?: number
  purchases?: number
  revenue?: number
}

export interface TrafficQualityStats {
  total_visitors: number
  human_count: number
  human_percent: number
  bot_count: number
  bot_percent: number
  inconclusive_count: number
  inconclusive_percent: number
  bot_categories?: Record<string, number>
}

export interface StatsFilters {
  from?: string
  to?: string
  pixel?: string
}
