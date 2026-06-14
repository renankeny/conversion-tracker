import type {
  CounterFilters,
  CounterSeriesResponse,
  CounterSummaryResponse,
  EventFilters,
  EventsResponse,
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
    const merged = { ...filters, event: 'Purchase', limit: filters.limit ?? 20 }
    return api.get<EventsResponse>('/admin/sent-events' + buildQuery(merged))
  }

  const geoStats = (filters: CounterFilters = {}) => {
    return api.get<any[]>('/admin/stats/geo' + buildQuery(filters))
  }

  return { summary, series, recentPurchases, geoStats }
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
