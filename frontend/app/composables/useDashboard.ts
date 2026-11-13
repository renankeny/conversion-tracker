import type {
  AdPlatformLiveInsights,
  CounterFilters,
  CounterSeriesResponse,
  CounterSummaryResponse,
  DashboardHeatmap,
  EventFilters,
  GoogleKeywordInsight,
  GroupedEventsResponse,
  PanelAudit,
  PreviousPeriodMetrics,
  SourceBreakdown,
  TrafficQualityStats,
} from '#shared/types'

export const useDashboard = () => {
  const api = useAPI()

  const summary = (filters: CounterFilters = {}, signal?: AbortSignal) => {
    return api.get<CounterSummaryResponse>('/admin/counters/summary' + buildQuery(filters), { signal })
  }

  const series = (filters: CounterFilters = {}, signal?: AbortSignal) => {
    return api.get<CounterSeriesResponse>('/admin/counters/series' + buildQuery(filters), { signal })
  }

  const recentPurchases = (filters: EventFilters = {}, signal?: AbortSignal) => {
    const merged = { ...filters, event: 'Purchase', limit: filters.limit ?? 20, include_ltv: true, include_buyer: true }
    return api.get<GroupedEventsResponse>('/admin/sent-events/grouped' + buildQuery(merged), { signal })
  }

  const geoStats = (filters: CounterFilters = {}, signal?: AbortSignal) => {
    return api.get<GeoStatPoint[]>('/admin/stats/geo' + buildQuery(filters), { signal })
  }

  const revenue = (filters: CounterFilters = {}, signal?: AbortSignal) => {
    return api.get<{
      total_revenue: number
      total_events: number
      tracked_users: number
      traffic_quality?: TrafficQualityStats
      source_breakdown?: SourceBreakdown
      panel_audit?: PanelAudit
      previous_period?: PreviousPeriodMetrics
      heatmap?: DashboardHeatmap
    }>('/admin/stats' + buildQuery(filters), { signal })
  }

  const geoStatsByEvent = (filters: CounterFilters & { event?: string } = {}, signal?: AbortSignal) => {
    return api.get<GeoStatPoint[]>('/admin/stats/geo-by-event' + buildQuery(filters), { signal })
  }

  const metaLiveInsights = (filters: { pixel_name: string; from: string; to: string }, signal?: AbortSignal) => {
    return api.get<AdPlatformLiveInsights>('/admin/dashboard/meta/live-insights' + buildQuery(filters), { signal })
  }

  const metaCreativeRanking = (filters: { pixel_name: string; from: string; to: string }, signal?: AbortSignal) => {
    return api.get<MetaCreativeInsight[]>('/admin/dashboard/meta/creative-ranking' + buildQuery(filters), { signal })
  }

  const googleLiveInsights = (filters: { pixel_name: string; from: string; to: string }, signal?: AbortSignal) => {
    return api.get<AdPlatformLiveInsights>('/admin/dashboard/google/live-insights' + buildQuery(filters), { signal })
  }

  const googleKeywordRanking = (filters: { pixel_name: string; from: string; to: string }, signal?: AbortSignal) => {
    return api.get<GoogleKeywordInsight[]>('/admin/dashboard/google/keyword-ranking' + buildQuery(filters), { signal })
  }

  return {
    summary,
    series,
    recentPurchases,
    geoStats,
    geoStatsByEvent,
    revenue,
    metaLiveInsights,
    metaCreativeRanking,
    googleLiveInsights,
    googleKeywordRanking,
  }
}

export interface GeoStatPoint {
  country: string
  state?: string
  city?: string
  lat?: number
  lon?: number
  count: number
}

export interface MetaCreativeInsight {
  ad_id: string
  ad_name: string
  spend: string
  clicks: string
  leads?: number
  purchases?: number
  placement?: string
  image_url?: string
  thumbnail_url?: string
}

const buildQuery = (params: object): string => {
  const entries = Object.entries(params as Record<string, unknown>).filter(
    ([, v]) => v !== undefined && v !== '' && v !== null,
  )
  if (entries.length === 0) return ''
  const qs = entries
    .map(([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(String(v))}`)
    .join('&')
  return '?' + qs
}
