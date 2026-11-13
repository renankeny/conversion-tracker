import type { GeoStat } from '~/utils/geoCoordinates'
import type {
  CounterSeriesResponse,
  CounterSummaryResponse,
  DashboardConfig,
  DashboardHeatmap,
  GroupedEventsResponse,
  PanelAudit,
  PreviousPeriodMetrics,
  SourceBreakdown,
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

  const meta = () =>
    api.publicGet<{ pixel_label: string; dashboard_config: DashboardConfig }>(`${base}/meta`)

  const summary = (f: PublicDashboardFilters) =>
    api.publicGet<CounterSummaryResponse>(`${base}/summary?from=${encodeURIComponent(f.from)}&to=${encodeURIComponent(f.to)}`)

  const series = (f: PublicDashboardFilters & { granularity: 'hour' | 'day' }) =>
    api.publicGet<CounterSeriesResponse>(
      `${base}/series?from=${encodeURIComponent(f.from)}&to=${encodeURIComponent(f.to)}&granularity=${encodeURIComponent(f.granularity)}`,
    )

  const geo = (f: PublicDashboardFilters) =>
    api.publicGet<GeoStat[]>(`${base}/geo?from=${encodeURIComponent(f.from)}&to=${encodeURIComponent(f.to)}`)

  const geoByEvent = (f: PublicDashboardFilters & { event: string }) =>
    api.publicGet<GeoStat[]>(
      `${base}/geo-by-event?from=${encodeURIComponent(f.from)}&to=${encodeURIComponent(f.to)}&event=${encodeURIComponent(f.event)}`,
    )

  const purchases = (f: PublicDashboardFilters) =>
    api.publicGet<GroupedEventsResponse>(`${base}/purchases?from=${encodeURIComponent(f.from)}&to=${encodeURIComponent(f.to)}`)

  const revenue = (f: PublicDashboardFilters) =>
    api.publicGet<{
      total_revenue: number
      source_breakdown?: SourceBreakdown
      panel_audit?: PanelAudit
      previous_period?: PreviousPeriodMetrics
      heatmap?: DashboardHeatmap
    }>(`${base}/revenue?from=${encodeURIComponent(f.from)}&to=${encodeURIComponent(f.to)}`)

  return { meta, summary, series, geo, geoByEvent, purchases, revenue }
}
