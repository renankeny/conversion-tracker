import type { Component } from 'vue'
import { useDashboardStore } from '~/stores/useDashboardStore'
import { useDashboardFetch } from '~/composables/useDashboardFetch'
import {
  EVENT_CATALOG, FULL_CARD_CATALOG, ALL_CARD_IDS, DEFAULT_ENABLED_CARDS, CATALOG_IDS,
  DEFAULT_FUNNEL_EVENTS, catalogById, computeCardMetricValue, resolveCardIcon, resolvePanelLeads,
  type CardFormat, type DashboardMetricsContext, type EventCatalogEntry,
} from '~/utils/dashboardEventCatalog'
import { computeCardRichDetails, type CardRichDetails } from '~/utils/dashboardCardDetails'
import type { CounterSummaryRow, DashboardConfig, GroupedEvent } from '#shared/types'

export interface DashboardVisibleCard extends CardRichDetails {
  id: string
  label: string
  icon: Component
  variant: EventCatalogEntry['variant']
  value: number
  format?: CardFormat
}

const PERIOD_LABELS: Record<string, string> = {
  today: 'Hoje', yesterday: 'Ontem', '7d': 'Últimos 7 dias',
  '30d': 'Últimos 30 dias', '60d': 'Últimos 60 dias', '90d': 'Últimos 90 dias', custom: 'Personalizado',
}

export const useDashboardView = () => {
  const toast = useToast()
  const pixelsApi = usePixels()
  const pixelStore = usePixelStore()
  const dashboardStore = useDashboardStore()

  const selectedPixel = computed({
    get: () => dashboardStore.selectedPixel,
    set: (v: string) => { dashboardStore.setSelectedPixel(v) },
  })

  const {
    loading, isRefreshing, adsLoading, summaryData, seriesData, trafficQuality, sourceBreakdown,
    panelAudit, previousPeriod, heatmap, metaInsights, googleInsights, metaCreatives,
    googleKeywords, recentPurchases, geoData, totalRevenue, refresh,
  } = useDashboardFetch(selectedPixel)

  const purchaseModalOpen = ref(false)
  const purchaseSelected = ref<GroupedEvent | null>(null)
  const onPurchaseSelect = (ev: GroupedEvent) => { purchaseSelected.value = ev; purchaseModalOpen.value = true }

  const filteredSeriesPoints = computed(() => seriesData.value?.points || [])

  const summaryByEvent = computed(() => new Map((summaryData.value?.events || []).map((r) => [r.event_name, r])))
  const sumOf = (event: string, field: 'count_total' | 'count_sent' | 'count_failed' = 'count_total'): number =>
    summaryByEvent.value.get(event)?.[field] ?? 0
  const sentOf = (event: string): number => sumOf(event, 'count_sent')
  const failedTotal = computed(() => (summaryData.value?.events || []).reduce((s: number, r: CounterSummaryRow) => s + r.count_failed, 0))

  const headerSubtitle = computed(() => `Visão executiva e auditoria de conversões — ${PERIOD_LABELS[dashboardStore.period]}`)
  const chartSubtitle = computed(() => `Comparativo de eventos — ${PERIOD_LABELS[dashboardStore.period]}`)
  const funnelSubtitle = computed(() => `Eventos do funil no período — ${PERIOD_LABELS[dashboardStore.period]}`)
  const purchasesSubtitle = computed(() => `${recentPurchases.value.length} compras recentes`)
  const selectedPixelObj = computed(() => pixelStore.pixels.find((p) => p.name === selectedPixel.value) || null)
  const activeCurrency = computed(() => selectedPixelObj.value?.tracker_config?.product_currency || 'BRL')

  const cardsEditorOpen = ref(false)
  const savingCardsPrefs = ref(false)
  const enabledCardIds = ref<string[]>([...DEFAULT_ENABLED_CARDS])
  const cardOrder = ref<string[]>([...ALL_CARD_IDS])
  const funnelEditorOpen = ref(false)
  const savingFunnelPrefs = ref(false)
  const enabledFunnelEvents = ref<string[]>([...DEFAULT_FUNNEL_EVENTS])
  const funnelOrder = ref<string[]>([...CATALOG_IDS])
  const eventLabelOverrides = ref<Record<string, string>>({})

  const eventLabel = (id: string, fallback: string): string => eventLabelOverrides.value[id] || fallback

  const hasCustomCardPrefs = computed(() => {
    const cfg = selectedPixelObj.value?.dashboard_config
    if (!cfg) return false
    if (cfg.event_labels && Object.values(cfg.event_labels).some(Boolean)) return true
    if (cfg.enabled_cards?.length) {
      if (cfg.enabled_cards.length !== DEFAULT_ENABLED_CARDS.length) return true
      if (cfg.enabled_cards.some((id) => !DEFAULT_ENABLED_CARDS.includes(id))) return true
    }
    if (cfg.card_order?.length) {
      const activeSaved = cfg.card_order.filter((id) => DEFAULT_ENABLED_CARDS.includes(id))
      if (activeSaved.some((id, idx) => id !== DEFAULT_ENABLED_CARDS[idx])) return true
    }
    return false
  })

  watch(selectedPixelObj, (p) => {
    const cfg = p?.dashboard_config
    enabledCardIds.value = cfg?.enabled_cards?.length ? cfg.enabled_cards : [...DEFAULT_ENABLED_CARDS]
    const savedOrder = cfg?.card_order?.length ? cfg.card_order : [...DEFAULT_ENABLED_CARDS, ...ALL_CARD_IDS]
    cardOrder.value = Array.from(new Set([...savedOrder, ...ALL_CARD_IDS]))
    enabledFunnelEvents.value = cfg?.enabled_funnel_events?.length ? cfg.enabled_funnel_events : [...DEFAULT_FUNNEL_EVENTS]
    funnelOrder.value = cfg?.funnel_order?.length ? cfg.funnel_order : [...CATALOG_IDS]
    eventLabelOverrides.value = cfg?.event_labels || {}
  }, { immediate: true })

  const cardPickerOptions = computed(() =>
    FULL_CARD_CATALOG.map((e) => ({ id: e.id, label: eventLabel(e.id, e.label), category: e.category })),
  )

  const metricsCtx = computed<DashboardMetricsContext>(() => ({
    sumOf,
    sentOf,
    failedTotal: failedTotal.value,
    totalRevenue: totalRevenue.value,
    currency: activeCurrency.value,
    trafficSource: dashboardStore.trafficSource,
    privacyMode: dashboardStore.privacyMode,
    metaInsights: metaInsights.value,
    googleInsights: googleInsights.value,
    sourceBreakdown: sourceBreakdown.value,
    panelAudit: panelAudit.value,
    previousPeriod: previousPeriod.value,
  }))

  const panelLeadsTotal = computed(() => resolvePanelLeads(metricsCtx.value))

  const visibleCards = computed<DashboardVisibleCard[]>(() => {
    const cards: DashboardVisibleCard[] = []
    const ctx = metricsCtx.value
    for (const id of cardOrder.value) {
      if (!enabledCardIds.value.includes(id)) continue
      const e = catalogById.get(id)
      if (!e) continue
      const value = computeCardMetricValue(id, ctx)
      cards.push({ id, label: eventLabel(id, e.label), icon: resolveCardIcon(e, ctx.trafficSource), variant: e.variant, value, format: e.format, ...computeCardRichDetails(id, value, ctx) })
    }
    return cards
  })

  const funnelPickerOptions = computed(() => EVENT_CATALOG.map((e) => ({ id: e.id, label: eventLabel(e.id, e.label) })))
  const funnelSteps = computed(() =>
    funnelOrder.value
      .filter((id) => enabledFunnelEvents.value.includes(id) && catalogById.has(id))
      .map((id) => {
        const e = catalogById.get(id)!
        return { event: id, label: eventLabel(id, e.label), color: e.color }
      }),
  )

  const saveDashboardConfig = async (patch: Partial<DashboardConfig>) => {
    if (!selectedPixelObj.value) return null
    const res = await pixelsApi.updateDashboardConfig(selectedPixelObj.value.id, {
      enabled_cards: enabledCardIds.value, card_order: cardOrder.value,
      enabled_funnel_events: enabledFunnelEvents.value, funnel_order: funnelOrder.value,
      event_labels: eventLabelOverrides.value, ...patch,
    })
    pixelStore.upsertPixel({ ...selectedPixelObj.value, dashboard_config: res.dashboard_config })
    return res.dashboard_config
  }

  const onSaveCards = async (payload: { ids: string[]; order: string[]; labels: Record<string, string> }) => {
    savingCardsPrefs.value = true
    try {
      await saveDashboardConfig({ enabled_cards: payload.ids, card_order: payload.order, event_labels: payload.labels })
      enabledCardIds.value = payload.ids; cardOrder.value = payload.order; eventLabelOverrides.value = payload.labels
      cardsEditorOpen.value = false; toast.success('Sucesso', 'Cards atualizados.')
    } catch { toast.error('Erro', 'Não foi possível salvar a configuração deste pixel.') }
    finally { savingCardsPrefs.value = false }
  }

  const onResetCardsDefault = () =>
    onSaveCards({ ids: [...DEFAULT_ENABLED_CARDS], order: Array.from(new Set([...DEFAULT_ENABLED_CARDS, ...ALL_CARD_IDS])), labels: {} })

  const onSaveFunnelEvents = async (payload: { ids: string[]; order: string[] }) => {
    savingFunnelPrefs.value = true
    try {
      await saveDashboardConfig({ enabled_funnel_events: payload.ids, funnel_order: payload.order })
      enabledFunnelEvents.value = payload.ids; funnelOrder.value = payload.order
      funnelEditorOpen.value = false; toast.success('Sucesso', 'Funil atualizado.')
    } catch { toast.error('Erro', 'Não foi possível salvar a configuração deste pixel.') }
    finally { savingFunnelPrefs.value = false }
  }

  return {
    dashboardStore, loading, isRefreshing, adsLoading, summaryData, seriesData, trafficQuality, heatmap, filteredSeriesPoints,
    metaCreatives, googleKeywords, recentPurchases, geoData, purchaseModalOpen, purchaseSelected, onPurchaseSelect,
    headerSubtitle, chartSubtitle, funnelSubtitle, purchasesSubtitle, selectedPixelObj, activeCurrency, panelLeadsTotal,
    hasCustomCardPrefs, cardsEditorOpen, savingCardsPrefs, enabledCardIds, cardOrder, cardPickerOptions, visibleCards,
    onSaveCards, onResetCardsDefault, funnelEditorOpen, savingFunnelPrefs, enabledFunnelEvents, funnelOrder,
    funnelPickerOptions, funnelSteps, onSaveFunnelEvents, eventLabelOverrides, refresh,
  }
}
