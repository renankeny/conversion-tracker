import type { Component } from 'vue'
import { useDashboardStore } from '~/stores/useDashboardStore'
import { useTenantStore } from '~/stores/useTenantStore'
import {
  EVENT_CATALOG, FULL_CARD_CATALOG, ALL_CARD_IDS, DEFAULT_ENABLED_CARDS, CATALOG_IDS,
  DEFAULT_FUNNEL_EVENTS, catalogById, computeCardMetricValue, type EventCatalogEntry,
} from '~/utils/dashboardEventCatalog'
import type {
  CounterSummaryResponse, CounterSeriesResponse, CounterSummaryRow,
  DashboardConfig, GroupedEvent, TrafficQualityStats, SourceBreakdown,
  AdPlatformLiveInsights, MetaCreativeInsight, GoogleKeywordInsight,
} from '#shared/types'

export const useDashboardView = () => {
  const dashboard = useDashboard()
  const toast = useToast()
  const liveStore = useLiveEventsStore()
  const pixelsApi = usePixels()
  const pixelStore = usePixelStore()
  const accountsApi = useAccounts()
  const dashboardStore = useDashboardStore()
  const tenantStore = useTenantStore()

  const selectedPixel = computed({
    get: () => dashboardStore.selectedPixel,
    set: (v: string) => { dashboardStore.setSelectedPixel(v) },
  })
  const loading = ref(false)
  const initialized = ref(false)
  const summaryData = ref<CounterSummaryResponse | null>(null)
  const seriesData = ref<CounterSeriesResponse | null>(null)
  const trafficQuality = ref<TrafficQualityStats | undefined>()
  const sourceBreakdown = ref<SourceBreakdown | null>(null)
  const metaInsights = ref<AdPlatformLiveInsights | null>(null)
  const googleInsights = ref<AdPlatformLiveInsights | null>(null)
  const metaCreatives = ref<MetaCreativeInsight[]>([])
  const googleKeywords = ref<GoogleKeywordInsight[]>([])
  const adsLoading = ref(false)
  const eventsFilterMap = ref<Record<string, string[]>>({})
  const recentPurchases = ref<GroupedEvent[]>([])
  const geoData = ref<any[]>([])
  const totalRevenue = ref(0)

  const purchaseModalOpen = ref(false)
  const purchaseSelected = ref<GroupedEvent | null>(null)
  const onPurchaseSelect = (ev: GroupedEvent) => { purchaseSelected.value = ev; purchaseModalOpen.value = true }

  const selectedEventsForChart = computed(() => eventsFilterMap.value[selectedPixel.value] || [])
  const filteredSeriesPoints = computed(() => {
    if (!selectedEventsForChart.value.length) return seriesData.value?.points || []
    const allowed = new Set(selectedEventsForChart.value)
    return (seriesData.value?.points || []).filter((p) => allowed.has(p.event_name))
  })

  const periodLabels: Record<string, string> = {
    today: 'Hoje', yesterday: 'Ontem', '7d': 'Últimos 7 dias',
    '30d': 'Últimos 30 dias', '60d': 'Últimos 60 dias', '90d': 'Últimos 90 dias', custom: 'Personalizado',
  }

  const summaryByEvent = computed(() => new Map((summaryData.value?.events || []).map((r) => [r.event_name, r])))
  const sumOf = (event: string, field: 'count_total' | 'count_sent' | 'count_failed' = 'count_total'): number =>
    summaryByEvent.value.get(event)?.[field] ?? 0

  const allEvents = computed<CounterSummaryRow[]>(() => summaryData.value?.events || [])
  const kpis = computed(() => ({
    total: allEvents.value.reduce((s, r) => s + r.count_total, 0),
    sent: allEvents.value.reduce((s, r) => s + r.count_sent, 0),
    failed: allEvents.value.reduce((s, r) => s + r.count_failed, 0),
  }))

  const headerSubtitle = computed(() => `Métricas e conversões — ${periodLabels[dashboardStore.period]}`)
  const chartSubtitle = computed(() => `Comparativo de eventos — ${periodLabels[dashboardStore.period]}`)
  const funnelSubtitle = computed(() => `Eventos do funil no período — ${periodLabels[dashboardStore.period]}`)
  const purchasesSubtitle = computed(() => `${recentPurchases.value.length} compras recentes`)
  const selectedPixelObj = computed(() => pixelStore.pixels.find((p) => p.name === selectedPixel.value) || null)

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

  const applyDashboardConfig = (cfg?: DashboardConfig) => {
    enabledCardIds.value = cfg?.enabled_cards?.length ? cfg.enabled_cards : [...DEFAULT_ENABLED_CARDS]
    const savedOrder = cfg?.card_order?.length ? cfg.card_order : [...ALL_CARD_IDS]
    const missing = ALL_CARD_IDS.filter((id) => !savedOrder.includes(id))
    cardOrder.value = [...savedOrder, ...missing]
    enabledFunnelEvents.value = cfg?.enabled_funnel_events?.length ? cfg.enabled_funnel_events : [...DEFAULT_FUNNEL_EVENTS]
    funnelOrder.value = cfg?.funnel_order?.length ? cfg.funnel_order : [...CATALOG_IDS]
    eventLabelOverrides.value = cfg?.event_labels || {}
  }

  watch(selectedPixelObj, (p) => applyDashboardConfig(p?.dashboard_config), { immediate: true })

  const cardPickerOptions = computed(() =>
    FULL_CARD_CATALOG.map((e) => ({ id: e.id, label: eventLabel(e.id, e.label), category: e.category })),
  )

  const visibleCards = computed(() => {
    const cards: Array<{ id: string; label: string; icon: Component; variant: EventCatalogEntry['variant']; value: number; format?: 'number' | 'currency' | 'roas' }> = []
    const ctx = {
      sumOf, failedTotal: kpis.value.failed, totalRevenue: totalRevenue.value,
      metaInsights: metaInsights.value, googleInsights: googleInsights.value, sourceBreakdown: sourceBreakdown.value,
    }
    for (const id of cardOrder.value) {
      if (!enabledCardIds.value.includes(id)) continue
      const e = catalogById.get(id)
      if (e) cards.push({ id, label: eventLabel(id, e.label), icon: e.icon, variant: e.variant, value: computeCardMetricValue(id, ctx), format: e.format })
    }
    return cards
  })

  const funnelPickerOptions = computed(() => EVENT_CATALOG.map((e) => ({ id: e.id, label: eventLabel(e.id, e.label) })))
  const funnelSteps = computed(() => {
    const steps: Array<{ event: string; label: string; color: string }> = []
    for (const id of funnelOrder.value) {
      if (!enabledFunnelEvents.value.includes(id)) continue
      const e = catalogById.get(id)
      if (e) steps.push({ event: id, label: eventLabel(id, e.label), color: e.color })
    }
    return steps
  })

  const saveDashboardConfig = async (patch: Partial<DashboardConfig>) => {
    if (!selectedPixelObj.value) return null
    const config: DashboardConfig = {
      enabled_cards: enabledCardIds.value, card_order: cardOrder.value,
      enabled_funnel_events: enabledFunnelEvents.value, funnel_order: funnelOrder.value,
      event_labels: eventLabelOverrides.value, ...patch,
    }
    const res = await pixelsApi.updateDashboardConfig(selectedPixelObj.value.id, config)
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

  const onSaveFunnelEvents = async (payload: { ids: string[]; order: string[] }) => {
    savingFunnelPrefs.value = true
    try {
      await saveDashboardConfig({ enabled_funnel_events: payload.ids, funnel_order: payload.order })
      enabledFunnelEvents.value = payload.ids; funnelOrder.value = payload.order
      funnelEditorOpen.value = false; toast.success('Sucesso', 'Funil atualizado.')
    } catch { toast.error('Erro', 'Não foi possível salvar a configuração deste pixel.') }
    finally { savingFunnelPrefs.value = false }
  }

  const ensurePixelSelected = () => {
    if (!pixelStore.pixels.length) return
    if (!selectedPixel.value || !pixelStore.pixels.some((p) => p.name === selectedPixel.value)) {
      selectedPixel.value = pixelStore.pixels[0]?.name || ''
    }
  }

  const fetchAll = async () => {
    ensurePixelSelected()
    loading.value = true
    adsLoading.value = true
    try {
      const { from, to } = dashboardStore.getDateRange()
      const granularity = dashboardStore.granularityFor
      const px = selectedPixel.value
      const ts = dashboardStore.trafficSource
      const [summary, series, purchases, geo, geoPurchases, revenue] = await Promise.all([
        dashboard.summary({ from, to, pixel: px, traffic_source: ts }),
        dashboard.series({ from, to, granularity, pixel: px, traffic_source: ts }),
        dashboard.recentPurchases({ from, to, limit: 50, pixel: px, traffic_source: ts }),
        dashboard.geoStats({ from, to, pixel: px, traffic_source: ts }),
        dashboard.geoStatsByEvent({ from, to, event: 'Purchase', pixel: px, traffic_source: ts }),
        dashboard.revenue({ from, to, pixel: px, traffic_source: ts }),
      ])
      summaryData.value = summary; seriesData.value = series; recentPurchases.value = purchases.data || []
      totalRevenue.value = revenue.total_revenue ?? 0; trafficQuality.value = revenue.traffic_quality
      sourceBreakdown.value = revenue.source_breakdown || null
      geoData.value = [...(geo || []).map((g: any) => ({ ...g, event_name: 'PageView' })), ...(geoPurchases || [])]

      if (px) {
        const [mLive, mRank, gLive, gRank] = await Promise.all([
          dashboard.metaLiveInsights({ pixel_name: px, from, to }).catch(() => null),
          dashboard.metaCreativeRanking({ pixel_name: px, from, to }).catch(() => null),
          dashboard.googleLiveInsights({ pixel_name: px, from, to }).catch(() => null),
          dashboard.googleKeywordRanking({ pixel_name: px, from, to }).catch(() => null),
        ])
        metaInsights.value = mLive; metaCreatives.value = (Array.isArray(mRank) ? mRank : []).slice(0, 3)
        googleInsights.value = gLive; googleKeywords.value = (Array.isArray(gRank) ? gRank : []).slice(0, 5)
      }
    } catch (error: any) {
      toast.error('Erro', error.message || 'Não foi possível carregar a dashboard')
    } finally { loading.value = false; adsLoading.value = false }
  }

  const loadEventsFilter = async () => {
    try { eventsFilterMap.value = (await accountsApi.getEventsFilter()).events_filter || {} }
    catch { eventsFilterMap.value = {} }
  }

  const refresh = () => fetchAll()
  watch([() => dashboardStore.period, () => dashboardStore.customFrom, () => dashboardStore.customTo, () => dashboardStore.trafficSource, selectedPixel], () => {
    if (!initialized.value) return
    if (dashboardStore.period !== 'custom' || (dashboardStore.customFrom && dashboardStore.customTo)) refresh()
  })

  watch(() => tenantStore.switchVersion, async () => {
    initialized.value = false
    await Promise.all([pixelStore.fetch(), loadEventsFilter()])
    ensurePixelSelected()
    initialized.value = true
    await fetchAll()
  })

  onMounted(async () => {
    await Promise.all([pixelStore.fetch().catch(() => {}), loadEventsFilter()])
    ensurePixelSelected()
    initialized.value = true
    await fetchAll()
  })

  return {
    dashboardStore, loading, adsLoading, summaryData, seriesData, trafficQuality, filteredSeriesPoints,
    metaCreatives, googleKeywords, recentPurchases, geoData, purchaseModalOpen, purchaseSelected, onPurchaseSelect,
    headerSubtitle, chartSubtitle, funnelSubtitle, purchasesSubtitle, selectedPixelObj,
    cardsEditorOpen, savingCardsPrefs, enabledCardIds, cardOrder, cardPickerOptions, visibleCards, onSaveCards,
    funnelEditorOpen, savingFunnelPrefs, enabledFunnelEvents, funnelOrder, funnelPickerOptions, funnelSteps, onSaveFunnelEvents,
    eventLabelOverrides, refresh,
  }
}
