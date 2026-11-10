import type { Component } from 'vue'
import { AlertTriangle } from 'lucide-vue-next'
import { useDashboardStore } from '~/stores/useDashboardStore'
import { useTenantStore } from '~/stores/useTenantStore'
import {
  EVENT_CATALOG, FAILED_CARD_ID, FAILED_CARD_LABEL, ALL_CARD_IDS, CATALOG_IDS,
  DEFAULT_FUNNEL_EVENTS, catalogById, REVENUE_CARD_ID, REVENUE_CARD_ICON, REVENUE_CARD_LABEL,
} from '~/utils/dashboardEventCatalog'
import type { EventCatalogEntry } from '~/utils/dashboardEventCatalog'
import type {
  CounterSummaryResponse, CounterSeriesResponse, CounterSummaryRow,
  DashboardConfig, GroupedEvent, TrafficQualityStats,
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
  const enabledCardIds = ref<string[]>([...ALL_CARD_IDS])
  const cardOrder = ref<string[]>([...ALL_CARD_IDS])
  const funnelEditorOpen = ref(false)
  const savingFunnelPrefs = ref(false)
  const enabledFunnelEvents = ref<string[]>([...DEFAULT_FUNNEL_EVENTS])
  const funnelOrder = ref<string[]>([...CATALOG_IDS])
  const eventLabelOverrides = ref<Record<string, string>>({})

  const eventLabel = (id: string, fallback: string): string => eventLabelOverrides.value[id] || fallback

  const applyDashboardConfig = (cfg?: DashboardConfig) => {
    enabledCardIds.value = cfg?.enabled_cards?.length ? cfg.enabled_cards : [...ALL_CARD_IDS]
    cardOrder.value = cfg?.card_order?.length ? cfg.card_order : [...ALL_CARD_IDS]
    enabledFunnelEvents.value = cfg?.enabled_funnel_events?.length ? cfg.enabled_funnel_events : [...DEFAULT_FUNNEL_EVENTS]
    funnelOrder.value = cfg?.funnel_order?.length ? cfg.funnel_order : [...CATALOG_IDS]
    eventLabelOverrides.value = cfg?.event_labels || {}
  }

  watch(selectedPixelObj, (p) => applyDashboardConfig(p?.dashboard_config), { immediate: true })

  const cardPickerOptions = computed(() => [
    ...EVENT_CATALOG.map((e) => ({ id: e.id, label: eventLabel(e.id, e.label) })),
    { id: FAILED_CARD_ID, label: eventLabel(FAILED_CARD_ID, FAILED_CARD_LABEL) },
    { id: REVENUE_CARD_ID, label: eventLabel(REVENUE_CARD_ID, REVENUE_CARD_LABEL) },
  ])

  const visibleCards = computed(() => {
    const cards: Array<{ id: string; label: string; icon: Component; variant: EventCatalogEntry['variant']; value: number; format?: 'number' | 'currency' }> = []
    for (const id of cardOrder.value) {
      if (!enabledCardIds.value.includes(id)) continue
      if (id === FAILED_CARD_ID) {
        cards.push({ id, label: eventLabel(id, FAILED_CARD_LABEL), icon: AlertTriangle, variant: 'danger', value: kpis.value.failed })
        continue
      }
      if (id === REVENUE_CARD_ID) {
        cards.push({ id, label: eventLabel(id, REVENUE_CARD_LABEL), icon: REVENUE_CARD_ICON, variant: 'primary', value: totalRevenue.value, format: 'currency' })
        continue
      }
      const e = catalogById.get(id)
      if (e) cards.push({ id, label: eventLabel(id, e.label), icon: e.icon, variant: e.variant, value: sumOf(id) })
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

  const fetchAll = async () => {
    loading.value = true
    try {
      const { from, to } = dashboardStore.getDateRange()
      const granularity = dashboardStore.granularityFor
      const [summary, series, purchases, geo, geoPurchases, revenue] = await Promise.all([
        dashboard.summary({ from, to, pixel: selectedPixel.value }),
        dashboard.series({ from, to, granularity, pixel: selectedPixel.value }),
        dashboard.recentPurchases({ from, to, limit: 50, pixel: selectedPixel.value }),
        dashboard.geoStats({ from, to, pixel: selectedPixel.value }),
        dashboard.geoStatsByEvent({ from, to, event: 'Purchase', pixel: selectedPixel.value }),
        dashboard.revenue({ from, to, pixel: selectedPixel.value }),
      ])
      summaryData.value = summary; seriesData.value = series; recentPurchases.value = purchases.data || []
      totalRevenue.value = revenue.total_revenue ?? 0; trafficQuality.value = revenue.traffic_quality
      const mappedGeo = (geo || []).map((g: any) => ({ ...g, event_name: 'PageView' }))
      geoData.value = [...mappedGeo, ...(geoPurchases || [])]
    } catch (error: any) {
      toast.error('Erro', error.message || 'Não foi possível carregar a dashboard')
    } finally { loading.value = false }
  }

  const loadEventsFilter = async () => {
    try {
      const res = await accountsApi.getEventsFilter()
      eventsFilterMap.value = res.events_filter || {}
    } catch { eventsFilterMap.value = {} }
  }

  const refresh = () => fetchAll()
  watch([() => dashboardStore.period, () => dashboardStore.customFrom, () => dashboardStore.customTo, selectedPixel], () => {
    if (!initialized.value) return
    if (dashboardStore.period !== 'custom' || (dashboardStore.customFrom && dashboardStore.customTo)) refresh()
  })

  watch(() => tenantStore.switchVersion, async () => {
    initialized.value = false
    await Promise.all([pixelStore.fetch(), loadEventsFilter()])
    initialized.value = true
    await fetchAll()
  })

  watch(() => liveStore.lastEventSeq, () => {
    const ev = liveStore.events[0]
    if (!ev || !dashboardStore.periodIncludesToday) return
    if (selectedPixel.value && ev.pixel_name !== selectedPixel.value) return
    if (summaryData.value) {
      let row = summaryData.value.events.find((e) => e.event_name === ev.event_name)
      if (!row) { row = { event_name: ev.event_name, count_total: 0, count_sent: 0, count_failed: 0 }; summaryData.value.events.push(row) }
      row.count_total++; if (ev.status === 'sent') row.count_sent++; else row.count_failed++
    }
    if (ev.event_name === 'Purchase') {
      if (ev.value) totalRevenue.value += ev.value
      recentPurchases.value.unshift({
        event_id: ev.event_id, event_name: ev.event_name, pixel_name: ev.pixel_name,
        external_id: ev.external_id, purchase_value: ev.value || 0, created_at: ev.ts,
        channels: (ev.channels || 'meta').split(',').map((c) => ({
          id: 0, channel: c as any, status: ev.status, response_code: ev.status === 'sent' ? 200 : 400, has_error: ev.status === 'failed',
        })),
      } as GroupedEvent)
      if (recentPurchases.value.length > 50) recentPurchases.value.pop()
    }
    if (geoData.value) {
      geoData.value.push({ country: ev.country || 'BR', state: ev.state || '', city: ev.city || '', count: 1, event_name: ev.event_name, ts: ev.ts })
    }
  })

  onMounted(async () => {
    await Promise.all([pixelStore.fetch().catch(() => {}), loadEventsFilter()])
    if (selectedPixel.value && pixelStore.pixels.length && !pixelStore.pixels.some((p) => p.name === selectedPixel.value)) {
      selectedPixel.value = ''
    }
    initialized.value = true
    await fetchAll()
  })

  return {
    dashboardStore, loading, summaryData, seriesData, trafficQuality, filteredSeriesPoints,
    recentPurchases, geoData, purchaseModalOpen, purchaseSelected, onPurchaseSelect,
    headerSubtitle, chartSubtitle, funnelSubtitle, purchasesSubtitle, selectedPixelObj,
    cardsEditorOpen, savingCardsPrefs, enabledCardIds, cardOrder, cardPickerOptions, visibleCards, onSaveCards,
    funnelEditorOpen, savingFunnelPrefs, enabledFunnelEvents, funnelOrder, funnelPickerOptions, funnelSteps, onSaveFunnelEvents,
    eventLabelOverrides, refresh,
  }
}
