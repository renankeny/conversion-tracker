<template>
  <div class="public-dashboard-page">
    <div v-if="notFound" class="state-box">
      <div class="mb-4 flex justify-center"><BrandLogo size="md" show-tagline /></div>
      <h1>Link indisponível</h1>
      <p>Esse link de visualização é inválido ou foi desativado. Peça um novo link a quem compartilhou.</p>
    </div>

    <div v-else-if="loadError && !metaLoaded" class="state-box">
      <div class="mb-4 flex justify-center"><BrandLogo size="md" show-tagline /></div>
      <h1>Falha temporária de conexão</h1>
      <p class="mb-4">{{ loadError }}</p>
      <div class="flex justify-center">
        <BaseButton variant="primary" size="sm" :loading="loading" @click="retryLoad">Tentar novamente</BaseButton>
      </div>
    </div>

    <template v-else>
      <div class="page-header">
        <div>
          <div class="brand-header-row"><BrandLogo size="sm" /></div>
          <h1 class="page-title">{{ pixelLabel || 'Dashboard' }}</h1>
          <p class="page-subtitle">Visualização pública — {{ periodLabels[period] }}</p>
        </div>
        <div class="header-controls">
          <PeriodSelector v-model="period" v-model:customFrom="customFrom" v-model:customTo="customTo" />
          <BaseButton variant="outline" size="sm" :loading="loading" @click="refresh">Atualizar</BaseButton>
        </div>
      </div>

      <div v-if="loadError" class="error-banner">
        <span>{{ loadError }}</span>
        <BaseButton variant="primary" size="sm" :loading="loading" @click="retryLoad">Tentar novamente</BaseButton>
      </div>

      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <KpiCard
          v-for="c in visibleCards"
          :key="c.id"
          :label="c.label"
          :value="c.value"
          :format="c.format"
          :variant="c.variant"
          :icon="c.icon"
          :change-percent="c.changePercent"
          :change-inverse="c.changeInverse"
          :previous-label="c.previousLabel"
          :comparisons="c.comparisons"
          :delivery="c.delivery"
        />
      </div>

      <DashboardHeatmapCard :heatmap="heatmap" />

      <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
        <EventsChart :points="seriesData?.points || []" :granularity="seriesData?.granularity || 'day'" subtitle="Comparativo de eventos" />
        <ConversionFunnel :events="summaryData?.events || []" :steps="funnelSteps" subtitle="Eventos do funil no período" />
      </div>

      <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
        <GeoGlobe :stats="geoData" :from="dateRange.from" :to="dateRange.to" />
        <RecentPurchases :purchases="recentPurchases" :subtitle="`${recentPurchases.length} compras recentes`" @select="onPurchaseSelect" />
      </div>

      <EventDetailsModal v-model="purchaseModalOpen" :source="purchaseSelected" :allow-user-details="false" />
    </template>
  </div>
</template>

<script setup lang="ts">
import KpiCard from '~/components/dashboard/KpiCard.vue'
import DashboardHeatmapCard from '~/components/dashboard/DashboardHeatmapCard.vue'
import EventsChart from '~/components/dashboard/EventsChart.vue'
import ConversionFunnel from '~/components/dashboard/ConversionFunnel.vue'
import GeoGlobe from '~/components/dashboard/GeoGlobe.vue'
import RecentPurchases from '~/components/dashboard/RecentPurchases.vue'
import PeriodSelector from '~/components/dashboard/PeriodSelector.vue'
import EventDetailsModal from '~/components/events/EventDetailsModal.vue'
import { computeSaoPauloRange, type PeriodValue } from '~/stores/useDashboardStore'
import {
  ALL_CARD_IDS, DEFAULT_ENABLED_CARDS, CATALOG_IDS,
  DEFAULT_FUNNEL_EVENTS, catalogById, computeCardMetricValue, resolveCardIcon,
} from '~/utils/dashboardEventCatalog'
import { computeCardRichDetails } from '~/utils/dashboardCardDetails'
import { errorMessage } from '~/utils/errorMessage'
import type { GeoStat } from '~/utils/geoCoordinates'
import type { DashboardVisibleCard } from '~/composables/useDashboardView'
import type {
  CounterSummaryResponse, CounterSeriesResponse, CounterSummaryRow,
  DashboardConfig, DashboardHeatmap, GroupedEvent, PanelAudit, PreviousPeriodMetrics, SourceBreakdown,
} from '#shared/types'

definePageMeta({ layout: 'default' })

const route = useRoute()
const token = route.params.token as string
const api = usePublicDashboard(token)
const { formatTitle, fetchBranding } = useBranding()

const period = ref<PeriodValue>('today')
const customFrom = ref('')
const customTo = ref('')
const dateRange = computed(() => computeSaoPauloRange(period.value, customFrom.value, customTo.value))
const granularity = computed<'hour' | 'day'>(() =>
  period.value === 'today' || period.value === 'yesterday' ? 'hour' : 'day',
)

const loading = ref(false)
const notFound = ref(false)
const metaLoaded = ref(false)
const loadError = ref<string | null>(null)
const pixelLabel = ref('')
const dashboardConfig = ref<DashboardConfig | null>(null)

useHead({ title: computed(() => formatTitle(pixelLabel.value || 'Dashboard')) })

const summaryData = ref<CounterSummaryResponse | null>(null)
const seriesData = ref<CounterSeriesResponse | null>(null)
const recentPurchases = ref<GroupedEvent[]>([])
const geoData = ref<GeoStat[]>([])
const totalRevenue = ref(0)
const sourceBreakdown = ref<SourceBreakdown | null>(null)
const panelAudit = ref<PanelAudit | null>(null)
const previousPeriod = ref<PreviousPeriodMetrics | null>(null)
const heatmap = ref<DashboardHeatmap | null>(null)

const purchaseModalOpen = ref(false)
const purchaseSelected = ref<GroupedEvent | null>(null)
const onPurchaseSelect = (ev: GroupedEvent) => { purchaseSelected.value = ev; purchaseModalOpen.value = true }

const periodLabels: Record<string, string> = {
  today: 'Hoje', yesterday: 'Ontem', '7d': 'Últimos 7 dias',
  '30d': 'Últimos 30 dias', '60d': 'Últimos 60 dias', '90d': 'Últimos 90 dias', custom: 'Personalizado',
}

const summaryByEvent = computed(() => {
  const map = new Map<string, CounterSummaryRow>()
  for (const row of summaryData.value?.events || []) map.set(row.event_name, row)
  return map
})
const sumOf = (event: string) => summaryByEvent.value.get(event)?.count_total ?? 0
const failedTotal = computed(() => (summaryData.value?.events || []).reduce((s, r) => s + r.count_failed, 0))
const eventLabel = (id: string, fallback: string): string => dashboardConfig.value?.event_labels?.[id] || fallback

const visibleCards = computed<DashboardVisibleCard[]>(() => {
  const order = dashboardConfig.value?.card_order?.length
    ? Array.from(new Set([...dashboardConfig.value.card_order, ...ALL_CARD_IDS]))
    : Array.from(new Set([...DEFAULT_ENABLED_CARDS, ...ALL_CARD_IDS]))
  const enabled = dashboardConfig.value?.enabled_cards?.length ? dashboardConfig.value.enabled_cards : DEFAULT_ENABLED_CARDS
  const cards: DashboardVisibleCard[] = []
  const ctx = {
    sumOf, failedTotal: failedTotal.value, totalRevenue: totalRevenue.value,
    sourceBreakdown: sourceBreakdown.value, panelAudit: panelAudit.value, previousPeriod: previousPeriod.value,
  }
  for (const id of order) {
    if (!enabled.includes(id)) continue
    const e = catalogById.get(id)
    if (!e) continue
    const value = computeCardMetricValue(id, ctx)
    const rich = computeCardRichDetails(id, value, ctx)
    cards.push({ id, label: eventLabel(id, e.label), icon: resolveCardIcon(e), variant: e.variant, value, format: e.format, ...rich })
  }
  return cards
})

const funnelSteps = computed(() => {
  const order = dashboardConfig.value?.funnel_order?.length ? dashboardConfig.value.funnel_order : CATALOG_IDS
  const enabled = dashboardConfig.value?.enabled_funnel_events?.length ? dashboardConfig.value.enabled_funnel_events : DEFAULT_FUNNEL_EVENTS
  const steps: Array<{ event: string; label: string; color: string }> = []
  for (const id of order) {
    if (!enabled.includes(id)) continue
    const e = catalogById.get(id)
    if (e) steps.push({ event: id, label: eventLabel(id, e.label), color: e.color })
  }
  return steps
})

const handleApiError = (err: unknown) => {
  const code = err && typeof err === 'object' ? (err as { statusCode?: number }).statusCode : undefined
  if (code === 404 || code === 410) { notFound.value = true; loadError.value = null }
  else loadError.value = errorMessage(err, 'Não foi possível carregar os dados no momento.')
}

const fetchMeta = async () => {
  try {
    const res = await api.meta()
    pixelLabel.value = res.pixel_label; dashboardConfig.value = res.dashboard_config
    metaLoaded.value = true; loadError.value = null
  } catch (err: unknown) { handleApiError(err) }
}

const fetchAll = async () => {
  if (notFound.value || !metaLoaded.value) return
  loading.value = true
  try {
    const { from, to } = dateRange.value
    const [summary, series, purchases, geo, geoPurchases, revenue] = await Promise.all([
      api.summary({ from, to }), api.series({ from, to, granularity: granularity.value }),
      api.purchases({ from, to }), api.geo({ from, to }),
      api.geoByEvent({ from, to, event: 'Purchase' }), api.revenue({ from, to }),
    ])
    summaryData.value = summary; seriesData.value = series; recentPurchases.value = purchases.data || []
    totalRevenue.value = revenue.total_revenue ?? 0
    sourceBreakdown.value = revenue.source_breakdown || null
    panelAudit.value = revenue.panel_audit || null
    previousPeriod.value = revenue.previous_period || null
    heatmap.value = revenue.heatmap || null
    geoData.value = [...(geo || []).map((g) => ({ ...g, event_name: 'PageView' })), ...(geoPurchases || [])]
    loadError.value = null
  } catch (err: unknown) { handleApiError(err) }
  finally { loading.value = false }
}

const refresh = () => fetchAll()
const retryLoad = async () => {
  loading.value = true
  if (!metaLoaded.value) await fetchMeta()
  if (!notFound.value && metaLoaded.value) await fetchAll()
  else loading.value = false
}

watch([period, customFrom, customTo], () => {
  if (period.value !== 'custom' || (customFrom.value && customTo.value)) refresh()
})

onMounted(async () => { void fetchBranding(); await retryLoad() })
</script>

<style scoped>
.public-dashboard-page { display: flex; flex-direction: column; gap: 1.5rem; max-width: 1400px; margin: 0 auto; padding: 1.5rem 1.25rem 3rem; }
.state-box { max-width: 480px; margin: 15vh auto 0; text-align: center; color: var(--text-1); }
.state-box h1 { font-family: var(--font-heading); font-size: var(--fs-2xl); font-weight: var(--fw-bold); margin-bottom: 0.5rem; }
.state-box p { color: var(--text-2); font-size: var(--fs-md); }
.error-banner { display: flex; align-items: center; justify-content: space-between; gap: 1rem; padding: 0.75rem 1rem; border-radius: 10px; background: var(--danger-soft); border: 1px solid var(--danger); color: var(--danger); font-size: var(--fs-md); flex-wrap: wrap; }
.brand-header-row { margin-bottom: 0.5rem; }
.page-header { display: flex; justify-content: space-between; align-items: flex-end; gap: 1rem; flex-wrap: wrap; }
.page-title { font-family: var(--font-heading); font-size: var(--fs-3xl); font-weight: var(--fw-bold); color: var(--text-strong); margin-bottom: 0.25rem; }
.page-subtitle { color: var(--text-2); font-size: var(--fs-lg); }
.header-controls { display: flex; gap: 0.625rem; align-items: center; flex-wrap: wrap; }
@media (max-width: 640px) { .public-dashboard-page { padding: 1rem 0.875rem 2rem; } .page-header { align-items: stretch; } }
</style>
