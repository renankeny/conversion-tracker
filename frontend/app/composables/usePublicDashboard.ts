import type { GeoStat } from '~/utils/geoCoordinates'
import type { DashboardTrafficSource } from '~/stores/useDashboardStore'
import type { MetaCreativeInsight } from '~/composables/useDashboard'
import type {
  AdPlatformLiveInsights,
  CounterSeriesResponse,
  CounterSummaryResponse,
  DashboardConfig,
  DashboardHeatmap,
  GoogleKeywordInsight,
  GroupedEventsResponse,
  PanelAudit,
  PreviousPeriodMetrics,
  SourceBreakdown,
  TrafficQualityStats,
} from '#shared/types'

export interface PublicDashboardFilters {
  from: string
  to: string
  traffic_source?: DashboardTrafficSource
}

const buildPublicQuery = (params: Record<string, string | undefined>): string => {
  const entries = Object.entries(params).filter(([, v]) => v !== undefined && v !== '')
  if (!entries.length) return ''
  return '?' + entries.map(([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(v!)}`).join('&')
}

// usePublicDashboard usa exclusivamente o cliente público ($publicApi),
// sem Authorization, sem X-Account-ID e sem enviar cookies administrativos.
export const usePublicDashboard = (token: string) => {
  const api = useAPI()
  const base = `/public/dashboard/${encodeURIComponent(token)}`

  const meta = (signal?: AbortSignal) =>
    api.publicGet<{ pixel_label: string; currency?: string; dashboard_config: DashboardConfig }>(`${base}/meta`, { signal })

  const adsSummary = (f: { from: string; to: string }, signal?: AbortSignal) =>
    api.publicGet<{ meta?: AdPlatformLiveInsights; google?: AdPlatformLiveInsights }>(
      `${base}/ads-summary${buildPublicQuery({ from: f.from, to: f.to })}`,
      { signal },
    )

  const metaCreatives = (f: { from: string; to: string }, signal?: AbortSignal) =>
    api.publicGet<MetaCreativeInsight[]>(
      `${base}/meta-creatives${buildPublicQuery({ from: f.from, to: f.to })}`,
      { signal },
    )

  const googleKeywords = (f: { from: string; to: string }, signal?: AbortSignal) =>
    api.publicGet<GoogleKeywordInsight[]>(
      `${base}/google-keywords${buildPublicQuery({ from: f.from, to: f.to })}`,
      { signal },
    )

  const summary = (f: PublicDashboardFilters, signal?: AbortSignal) =>
    api.publicGet<CounterSummaryResponse>(
      `${base}/summary${buildPublicQuery({ from: f.from, to: f.to, traffic_source: f.traffic_source })}`,
      { signal },
    )

  const series = (f: PublicDashboardFilters & { granularity: 'hour' | 'day' }, signal?: AbortSignal) =>
    api.publicGet<CounterSeriesResponse>(
      `${base}/series${buildPublicQuery({ from: f.from, to: f.to, granularity: f.granularity, traffic_source: f.traffic_source })}`,
      { signal },
    )

  const geo = (f: PublicDashboardFilters, signal?: AbortSignal) =>
    api.publicGet<GeoStat[]>(
      `${base}/geo${buildPublicQuery({ from: f.from, to: f.to, traffic_source: f.traffic_source })}`,
      { signal },
    )

  const geoByEvent = (f: PublicDashboardFilters & { event: string }, signal?: AbortSignal) =>
    api.publicGet<GeoStat[]>(
      `${base}/geo-by-event${buildPublicQuery({ from: f.from, to: f.to, event: f.event, traffic_source: f.traffic_source })}`,
      { signal },
    )

  const purchases = (f: PublicDashboardFilters, signal?: AbortSignal) =>
    api.publicGet<GroupedEventsResponse>(
      `${base}/purchases${buildPublicQuery({ from: f.from, to: f.to, traffic_source: f.traffic_source })}`,
      { signal },
    )

  const revenue = (f: PublicDashboardFilters, signal?: AbortSignal) =>
    api.publicGet<{
      total_revenue: number
      traffic_quality?: TrafficQualityStats
      source_breakdown?: SourceBreakdown
      panel_audit?: PanelAudit
      previous_period?: PreviousPeriodMetrics
      heatmap?: DashboardHeatmap
    }>(
      `${base}/revenue${buildPublicQuery({ from: f.from, to: f.to, traffic_source: f.traffic_source })}`,
      { signal },
    )

  return { meta, adsSummary, metaCreatives, googleKeywords, summary, series, geo, geoByEvent, purchases, revenue }
}
