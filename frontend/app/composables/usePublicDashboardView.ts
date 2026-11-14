import { ref, computed } from 'vue'
import { usePublicDashboardFetch } from '~/composables/usePublicDashboardFetch'
import {
  ALL_CARD_IDS, DEFAULT_ENABLED_CARDS, CATALOG_IDS,
  DEFAULT_FUNNEL_EVENTS, catalogById, computeCardMetricValue, resolveCardIcon, resolvePanelLeads,
  type DashboardMetricsContext,
} from '~/utils/dashboardEventCatalog'
import { computeCardRichDetails } from '~/utils/dashboardCardDetails'
import type { DashboardVisibleCard } from '~/composables/useDashboardView'
import type { CounterSummaryRow, GroupedEvent } from '#shared/types'

export const PUBLIC_PERIOD_LABELS: Record<string, string> = {
  today: 'Hoje', yesterday: 'Ontem', '7d': 'Últimos 7 dias',
  '30d': 'Últimos 30 dias', '60d': 'Últimos 60 dias', '90d': 'Últimos 90 dias', custom: 'Personalizado',
}

export const usePublicDashboardView = (token: string) => {
  const {
    period, customFrom, customTo, trafficSource, privacyMode, dateRange, granularity,
    loading, isRefreshing, adsLoading, notFound, metaLoaded, loadError, pixelLabel, currency, dashboardConfig,
    summaryData, seriesData, recentPurchases, geoData, totalRevenue, trafficQuality,
    sourceBreakdown, panelAudit, previousPeriod, heatmap, metaInsights, googleInsights,
    metaCreatives, googleKeywords, refresh, retryLoad,
  } = usePublicDashboardFetch(token)

  const purchaseModalOpen = ref(false)
  const purchaseSelected = ref<GroupedEvent | null>(null)
  const onPurchaseSelect = (ev: GroupedEvent) => {
    purchaseSelected.value = ev
    purchaseModalOpen.value = true
  }

  const summaryByEvent = computed(() => new Map<string, CounterSummaryRow>((summaryData.value?.events || []).map((r) => [r.event_name, r])))
  const sumOf = (event: string) => summaryByEvent.value.get(event)?.count_total ?? 0
  const sentOf = (event: string) => summaryByEvent.value.get(event)?.count_sent ?? 0
  const failedTotal = computed(() => (summaryData.value?.events || []).reduce((s, r) => s + r.count_failed, 0))
  const eventLabel = (id: string, fallback: string): string => dashboardConfig.value?.event_labels?.[id] || fallback

  const periodLabel = computed(() => PUBLIC_PERIOD_LABELS[period.value] || 'Hoje')
  const headerSubtitle = computed(() => `Visão executiva e auditoria de conversões • ${periodLabel.value}`)
  const chartSubtitle = computed(() => `Comparativo de eventos • ${periodLabel.value}`)
  const funnelSubtitle = computed(() => `Eventos do funil no período • ${periodLabel.value}`)
  const purchasesSubtitle = computed(() => `${recentPurchases.value.length} compras recentes`)

  const metricsCtx = computed<DashboardMetricsContext>(() => ({
    sumOf,
    sentOf,
    failedTotal: failedTotal.value,
    totalRevenue: totalRevenue.value,
    currency: currency.value,
    trafficSource: trafficSource.value,
    privacyMode: privacyMode.value,
    metaInsights: metaInsights.value,
    googleInsights: googleInsights.value,
    sourceBreakdown: sourceBreakdown.value,
    panelAudit: panelAudit.value,
    previousPeriod: previousPeriod.value,
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
      cards.push({
        id,
        label: eventLabel(id, e.label),
        icon: resolveCardIcon(e, ctx.trafficSource),
        variant: e.variant,
        value,
        format: e.format,
        ...computeCardRichDetails(id, value, ctx),
      })
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

  return {
    period, customFrom, customTo, trafficSource, privacyMode, dateRange, granularity,
    loading, isRefreshing, adsLoading, notFound, metaLoaded, loadError, pixelLabel, currency,
    headerSubtitle, chartSubtitle, funnelSubtitle, purchasesSubtitle,
    summaryData, seriesData, recentPurchases, geoData, trafficQuality, heatmap,
    metaCreatives, googleKeywords, panelLeadsTotal, visibleCards, funnelSteps,
    purchaseModalOpen, purchaseSelected, onPurchaseSelect, refresh, retryLoad,
  }
}
