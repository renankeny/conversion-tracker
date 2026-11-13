import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { computeSaoPauloRange, type PeriodValue } from '~/stores/useDashboardStore'
import {
  ALL_CARD_IDS, DEFAULT_ENABLED_CARDS, CATALOG_IDS,
  DEFAULT_FUNNEL_EVENTS, catalogById, computeCardMetricValue, resolveCardIcon, resolvePanelLeads,
} from '~/utils/dashboardEventCatalog'
import { computeCardRichDetails } from '~/utils/dashboardCardDetails'
import { errorMessage } from '~/utils/errorMessage'
import type { GeoStat } from '~/utils/geoCoordinates'
import type { DashboardVisibleCard } from '~/composables/useDashboardView'
import type {
  AdPlatformLiveInsights, CounterSummaryResponse, CounterSeriesResponse, CounterSummaryRow,
  DashboardConfig, DashboardHeatmap, GroupedEvent, PanelAudit, PreviousPeriodMetrics, SourceBreakdown,
} from '#shared/types'

export const PUBLIC_PERIOD_LABELS: Record<string, string> = {
  today: 'Hoje', yesterday: 'Ontem', '7d': 'Últimos 7 dias',
  '30d': 'Últimos 30 dias', '60d': 'Últimos 60 dias', '90d': 'Últimos 90 dias', custom: 'Personalizado',
}

export const usePublicDashboardView = (token: string) => {
  const api = usePublicDashboard(token)
  const { fetchBranding } = useBranding()

  const period = ref<PeriodValue>('today')
  const customFrom = ref('')
  const customTo = ref('')
  const dateRange = computed(() => computeSaoPauloRange(period.value, customFrom.value, customTo.value))
  const granularity = computed<'hour' | 'day'>(() => (period.value === 'today' || period.value === 'yesterday' ? 'hour' : 'day'))

  const loading = ref(false)
  const notFound = ref(false)
  const metaLoaded = ref(false)
  const loadError = ref<string | null>(null)
  const pixelLabel = ref('')
  const currency = ref('BRL')
  const dashboardConfig = ref<DashboardConfig | null>(null)

  const summaryData = ref<CounterSummaryResponse | null>(null)
  const seriesData = ref<CounterSeriesResponse | null>(null)
  const recentPurchases = ref<GroupedEvent[]>([])
  const geoData = ref<GeoStat[]>([])
  const totalRevenue = ref(0)
  const sourceBreakdown = ref<SourceBreakdown | null>(null)
  const panelAudit = ref<PanelAudit | null>(null)
  const previousPeriod = ref<PreviousPeriodMetrics | null>(null)
  const heatmap = ref<DashboardHeatmap | null>(null)
  const metaInsights = ref<AdPlatformLiveInsights | null>(null)
  const googleInsights = ref<AdPlatformLiveInsights | null>(null)

  const purchaseModalOpen = ref(false)
  const purchaseSelected = ref<GroupedEvent | null>(null)
  const onPurchaseSelect = (ev: GroupedEvent) => { purchaseSelected.value = ev; purchaseModalOpen.value = true }

  const summaryByEvent = computed(() => new Map<string, CounterSummaryRow>((summaryData.value?.events || []).map((r) => [r.event_name, r])))
  const sumOf = (event: string) => summaryByEvent.value.get(event)?.count_total ?? 0
  const sentOf = (event: string) => summaryByEvent.value.get(event)?.count_sent ?? 0
  const failedTotal = computed(() => (summaryData.value?.events || []).reduce((s, r) => s + r.count_failed, 0))
  const eventLabel = (id: string, fallback: string): string => dashboardConfig.value?.event_labels?.[id] || fallback

  const metricsCtx = computed(() => ({
    sumOf, sentOf, failedTotal: failedTotal.value, totalRevenue: totalRevenue.value, currency: currency.value,
    metaInsights: metaInsights.value, googleInsights: googleInsights.value,
    sourceBreakdown: sourceBreakdown.value, panelAudit: panelAudit.value, previousPeriod: previousPeriod.value,
  }))

  const panelLeadsTotal = computed(() => resolvePanelLeads(metricsCtx.value))

  const visibleCards = computed<DashboardVisibleCard[]>(() => {
    const order = dashboardConfig.value?.card_order?.length
      ? Array.from(new Set([...dashboardConfig.value.card_order, ...ALL_CARD_IDS]))
      : Array.from(new Set([...DEFAULT_ENABLED_CARDS, ...ALL_CARD_IDS]))
    const enabled = dashboardConfig.value?.enabled_cards?.length ? dashboardConfig.value.enabled_cards : DEFAULT_ENABLED_CARDS
    const cards: DashboardVisibleCard[] = []
    const ctx = metricsCtx.value
    for (const id of order) {
      if (!enabled.includes(id)) continue
      const e = catalogById.get(id)
      if (!e) continue
      const value = computeCardMetricValue(id, ctx)
      cards.push({ id, label: eventLabel(id, e.label), icon: resolveCardIcon(e), variant: e.variant, value, format: e.format, ...computeCardRichDetails(id, value, ctx) })
    }
    return cards
  })

  const funnelSteps = computed(() => {
    const order = dashboardConfig.value?.funnel_order?.length ? dashboardConfig.value.funnel_order : CATALOG_IDS
    const enabled = dashboardConfig.value?.enabled_funnel_events?.length ? dashboardConfig.value.enabled_funnel_events : DEFAULT_FUNNEL_EVENTS
    return order.filter((id) => enabled.includes(id) && catalogById.has(id)).map((id) => {
      const e = catalogById.get(id)!
      return { event: id, label: eventLabel(id, e.label), color: e.color }
    })
  })

  let fetchSeq = 0
  let abortCtrl: AbortController | null = null
  const isAborted = (err: unknown) => Boolean(err && typeof err === 'object' && ((err as { name?: string }).name === 'AbortError' || String(err).includes('aborted')))

  const handleApiError = (err: unknown) => {
    if (isAborted(err)) return
    const code = err && typeof err === 'object' ? (err as { statusCode?: number }).statusCode : undefined
    if (code === 404 || code === 410) { notFound.value = true; loadError.value = null }
    else loadError.value = errorMessage(err, 'Não foi possível carregar os dados no momento.')
  }

  const fetchMeta = async () => {
    try {
      const res = await api.meta()
      pixelLabel.value = res.pixel_label; currency.value = res.currency || 'BRL'; dashboardConfig.value = res.dashboard_config
      metaLoaded.value = true; loadError.value = null
    } catch (err: unknown) { handleApiError(err) }
  }

  const fetchAll = async () => {
    if (notFound.value || !metaLoaded.value) return
    abortCtrl?.abort()
    const ac = new AbortController(); abortCtrl = ac
    const seq = ++fetchSeq; loading.value = true
    try {
      const { from, to } = dateRange.value
      const [summary, series, purchases, geo, geoPurchases, revenue, ads] = await Promise.all([
        api.summary({ from, to }, ac.signal), api.series({ from, to, granularity: granularity.value }, ac.signal),
        api.purchases({ from, to }, ac.signal), api.geo({ from, to }, ac.signal),
        api.geoByEvent({ from, to, event: 'Purchase' }, ac.signal), api.revenue({ from, to }, ac.signal),
        api.adsSummary({ from, to }, ac.signal).catch(() => null),
      ])
      if (seq !== fetchSeq || ac.signal.aborted) return
      summaryData.value = summary; seriesData.value = series; recentPurchases.value = purchases.data || []
      totalRevenue.value = revenue.total_revenue ?? 0; sourceBreakdown.value = revenue.source_breakdown || null
      panelAudit.value = revenue.panel_audit || null; previousPeriod.value = revenue.previous_period || null
      heatmap.value = revenue.heatmap || null; metaInsights.value = ads?.meta || null; googleInsights.value = ads?.google || null
      geoData.value = [...(geo || []).map((g) => ({ ...g, event_name: 'PageView' })), ...(geoPurchases || [])]
      loadError.value = null
    } catch (err: unknown) { if (seq === fetchSeq) handleApiError(err) }
    finally { if (seq === fetchSeq) loading.value = false }
  }

  const refresh = () => fetchAll()
  const retryLoad = async () => {
    loading.value = true
    if (!metaLoaded.value) await fetchMeta()
    if (!notFound.value && metaLoaded.value) await fetchAll()
    else loading.value = false
  }

  watch([period, customFrom, customTo], () => {
    if (period.value !== 'custom' || (customFrom.value && customTo.value)) void refresh()
  })

  onMounted(async () => { void fetchBranding(); await retryLoad() })
  onUnmounted(() => { abortCtrl?.abort() })

  return {
    period, customFrom, customTo, dateRange, granularity, loading, notFound, metaLoaded, loadError, pixelLabel, currency,
    summaryData, seriesData, recentPurchases, geoData, heatmap, panelLeadsTotal, visibleCards, funnelSteps,
    purchaseModalOpen, purchaseSelected, onPurchaseSelect, refresh, retryLoad,
  }
}

