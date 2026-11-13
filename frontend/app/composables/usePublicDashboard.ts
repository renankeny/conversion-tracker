import type { GeoStat } from '~/utils/geoCoordinates'
import type {
  AdPlatformLiveInsights,
  CounterSeriesResponse,
  CounterSummaryResponse,
  DashboardConfig,
  DashboardHeatmap,
  GroupedEventsResponse,
  PanelAudit,
  PreviousPeriodMetrics,
  SourceBreakdown,
  TrafficQualityStats,
} from '#shared/types'

interface PublicDashboardFilters {
  from: string
  to: string
}

// usePublicDashboard usa exclusivamente o cliente público ($publicApi),
// sem Authorization, sem X-Account-ID e sem enviar cookies administrativos.
export const usePublicDashboard = (token: string) => {
  const api = useAPI()
  const base = `/public/dashboard/${encodeURIComponent(token)}`

  const meta = (signal?: AbortSignal) =>
    api.publicGet<{ pixel_label: string; currency?: string; dashboard_config: DashboardConfig }>(`${base}/meta`, { signal })

  const adsSummary = (f: PublicDashboardFilters, signal?: AbortSignal) =>
    api.publicGet<{ meta?: AdPlatformLiveInsights; google?: AdPlatformLiveInsights }>(
      `${base}/ads-summary?from=${encodeURIComponent(f.from)}&to=${encodeURIComponent(f.to)}`,
      { signal },
    )

  const summary = (f: PublicDashboardFilters, signal?: AbortSignal) =>
    api.publicGet<CounterSummaryResponse>(`${base}/summary?from=${encodeURIComponent(f.from)}&to=${encodeURIComponent(f.to)}`, { signal })

  const series = (f: PublicDashboardFilters & { granularity: 'hour' | 'day' }, signal?: AbortSignal) =>
    api.publicGet<CounterSeriesResponse>(
      `${base}/series?from=${encodeURIComponent(f.from)}&to=${encodeURIComponent(f.to)}&granularity=${encodeURIComponent(f.granularity)}`,
      { signal },
    )

  const geo = (f: PublicDashboardFilters, signal?: AbortSignal) =>
    api.publicGet<GeoStat[]>(`${base}/geo?from=${encodeURIComponent(f.from)}&to=${encodeURIComponent(f.to)}`, { signal })

  const geoByEvent = (f: PublicDashboardFilters & { event: string }, signal?: AbortSignal) =>
    api.publicGet<GeoStat[]>(
      `${base}/geo-by-event?from=${encodeURIComponent(f.from)}&to=${encodeURIComponent(f.to)}&event=${encodeURIComponent(f.event)}`,
      { signal },
    )

  const purchases = (f: PublicDashboardFilters, signal?: AbortSignal) =>
    api.publicGet<GroupedEventsResponse>(`${base}/purchases?from=${encodeURIComponent(f.from)}&to=${encodeURIComponent(f.to)}`, { signal })

  const revenue = (f: PublicDashboardFilters, signal?: AbortSignal) =>
    api.publicGet<{
      total_revenue: number
      traffic_quality?: TrafficQualityStats
      source_breakdown?: SourceBreakdown
      panel_audit?: PanelAudit
      previous_period?: PreviousPeriodMetrics
      heatmap?: DashboardHeatmap
    }>(`${base}/revenue?from=${encodeURIComponent(f.from)}&to=${encodeURIComponent(f.to)}`, { signal })

  return { meta, adsSummary, summary, series, geo, geoByEvent, purchases, revenue }
}
