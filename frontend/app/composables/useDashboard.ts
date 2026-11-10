import type {
  CounterFilters,
  CounterSeriesResponse,
  CounterSummaryResponse,
  EventFilters,
  GroupedEventsResponse,
  TrafficQualityStats,
} from '#shared/types'

export const useDashboard = () => {
  const api = useAPI()

  const summary = (filters: CounterFilters = {}) => {
    return api.get<CounterSummaryResponse>(
      '/admin/counters/summary' + buildQuery(filters)
    )
  }

  const series = (filters: CounterFilters = {}) => {
    return api.get<CounterSeriesResponse>(
      '/admin/counters/series' + buildQuery(filters)
    )
  }

  const recentPurchases = (filters: EventFilters = {}) => {
    const merged = { ...filters, event: 'Purchase', limit: filters.limit ?? 20, include_ltv: true }
    return api.get<GroupedEventsResponse>('/admin/sent-events/grouped' + buildQuery(merged))
  }

  const geoStats = (filters: CounterFilters = {}) => {
    return api.get<any[]>('/admin/stats/geo' + buildQuery(filters))
  }

  const revenue = (filters: CounterFilters = {}) => {
    return api.get<{ total_revenue: number; total_events: number; tracked_users: number; traffic_quality?: TrafficQualityStats }>(
      '/admin/stats' + buildQuery(filters)
    )
  }

  const geoStatsByEvent = (filters: CounterFilters & { event?: string } = {}) => {
    return api.get<any[]>('/admin/stats/geo-by-event' + buildQuery(filters))
  }

  const metaLiveInsights = (filters: { pixel_name: string; from: string; to: string }) => {
    return api.get<{ spend: number }>('/admin/dashboard/meta/live-insights' + buildQuery(filters))
  }

  const metaCreativeRanking = (filters: { pixel_name: string; from: string; to: string }) => {
    return api.get<MetaCreativeInsight[]>('/admin/dashboard/meta/creative-ranking' + buildQuery(filters))
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
  }
}

export interface MetaCreativeInsight {
  ad_id: string
  ad_name: string
  spend: string
  clicks: string
  image_url?: string
  thumbnail_url?: string
}

const buildQuery = (params: Record<string, any>): string => {
  const entries = Object.entries(params).filter(
    ([, v]) => v !== undefined && v !== '' && v !== null
  )
  if (entries.length === 0) return ''
  const qs = entries
    .map(([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(String(v))}`)
    .join('&')
  return '?' + qs
}
