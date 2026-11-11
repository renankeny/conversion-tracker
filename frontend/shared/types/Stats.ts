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

export interface AdPlatformLiveInsights {
  spend: number
  clicks: number
  impressions: number
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

