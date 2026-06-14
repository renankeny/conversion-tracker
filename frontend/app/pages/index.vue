<template>
  <div class="dashboard-page">
    <div class="page-header">
      <div>
        <h1 class="page-title">Dashboard</h1>
        <p class="page-subtitle">{{ headerSubtitle }}</p>
      </div>
      <div class="header-controls">
        <PeriodSelector v-model="period" />
        <BaseButton variant="outline" size="sm" :loading="loading" @click="refresh">
          Atualizar
        </BaseButton>
      </div>
    </div>

    <!-- KPIs -->
    <div class="kpis-grid">
      <KpiCard
        label="Visitas (PageView)"
        :value="kpis.pageViews"
        variant="primary"
        :icon="icons.eye"
      />
      <KpiCard
        label="Add to Cart"
        :value="kpis.addToCart"
        variant="warning"
        :icon="icons.cart"
      />
      <KpiCard
        label="Initiate Checkout"
        :value="kpis.initiateCheckout"
        variant="info"
        :icon="icons.checkout"
      />
      <KpiCard
        label="Leads"
        :value="kpis.leads"
        variant="primary"
        :icon="icons.lead"
      />
      <KpiCard
        label="Compras (Purchase)"
        :value="kpis.purchases"
        variant="success"
        :icon="icons.purchase"
      />
      <KpiCard
        label="Taxa de conversão"
        :value="kpis.conversionRate"
        format="percent"
        variant="success"
        :icon="icons.trending"
        hint="Purchases / Visitas"
      />
    </div>

    <!-- Health -->
    <div class="health-row">
      <KpiCard
        label="Eventos enviados"
        :value="kpis.sent"
        variant="success"
        :icon="icons.check"
        :hint="kpis.sentRate"
      />
      <KpiCard
        label="Eventos com falha"
        :value="kpis.failed"
        variant="danger"
        :icon="icons.alert"
      />
      <KpiCard
        label="Total de eventos"
        :value="kpis.total"
        variant="info"
        :icon="icons.layers"
        hint="Soma de todos os eventos no período"
      />
    </div>

    <!-- Gráfico + Funil -->
    <div class="grid-2">
      <EventsChart
        :points="seriesData?.points || []"
        :granularity="seriesData?.granularity || 'day'"
        :subtitle="chartSubtitle"
      />
      <ConversionFunnel
        :events="summaryData?.events || []"
        :subtitle="funnelSubtitle"
      />
    </div>

    <!-- Globo + Últimas compras -->
    <div class="grid-2-globe">
      <GeoGlobe :stats="geoData" />
      <RecentPurchases
        :purchases="recentPurchases"
        :subtitle="purchasesSubtitle"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import KpiCard from '~/components/dashboard/KpiCard.vue'
import EventsChart from '~/components/dashboard/EventsChart.vue'
import ConversionFunnel from '~/components/dashboard/ConversionFunnel.vue'
import RecentPurchases from '~/components/dashboard/RecentPurchases.vue'
import GeoGlobe from '~/components/dashboard/GeoGlobe.vue'
import PeriodSelector, { type PeriodValue } from '~/components/dashboard/PeriodSelector.vue'
import type {
  CounterSummaryResponse,
  CounterSeriesResponse,
  CounterSummaryRow,
  SentEvent,
} from '#shared/types'

definePageMeta({
  layout: 'app',
  middleware: 'auth',
})

useHead({ title: 'Dashboard - TrackFind' })

const dashboard = useDashboard()
const toast = useToast()

const period = ref<PeriodValue>('7d')
const loading = ref(false)

const summaryData = ref<CounterSummaryResponse | null>(null)
const seriesData = ref<CounterSeriesResponse | null>(null)
const recentPurchases = ref<SentEvent[]>([])
const geoData = ref<any[]>([])

// === Helpers de período ===
const periodLabels: Record<PeriodValue, string> = {
  '24h': 'Últimas 24 horas',
  '7d': 'Últimos 7 dias',
  '30d': 'Últimos 30 dias',
  '90d': 'Últimos 90 dias',
}

const periodToFromIso = (p: PeriodValue): string => {
  const now = new Date()
  const from = new Date(now)
  switch (p) {
    case '24h':
      from.setHours(now.getHours() - 24)
      break
    case '7d':
      from.setDate(now.getDate() - 7)
      break
    case '30d':
      from.setDate(now.getDate() - 30)
      break
    case '90d':
      from.setDate(now.getDate() - 90)
      break
  }
  return from.toISOString()
}

const granularityFor = (p: PeriodValue): 'hour' | 'day' => {
  return p === '24h' ? 'hour' : 'day'
}

// === KPIs derivados ===
const summaryByEvent = computed(() => {
  const map = new Map<string, CounterSummaryRow>()
  for (const row of summaryData.value?.events || []) {
    map.set(row.event_name, row)
  }
  return map
})

const sumOf = (event: string, field: 'count_total' | 'count_sent' | 'count_failed' = 'count_total'): number => {
  return summaryByEvent.value.get(event)?.[field] ?? 0
}

const allEvents = computed<CounterSummaryRow[]>(() => summaryData.value?.events || [])

const kpis = computed(() => {
  const pageViews = sumOf('PageView')
  const purchases = sumOf('Purchase')
  const total = allEvents.value.reduce((s, r) => s + r.count_total, 0)
  const sent = allEvents.value.reduce((s, r) => s + r.count_sent, 0)
  const failed = allEvents.value.reduce((s, r) => s + r.count_failed, 0)
  const conversionRate = pageViews > 0 ? purchases / pageViews : 0
  const sentRatePct = total > 0 ? Math.round((sent / total) * 100) : 0

  return {
    pageViews,
    addToCart: sumOf('AddToCart'),
    initiateCheckout: sumOf('InitiateCheckout'),
    leads: sumOf('Lead'),
    purchases,
    total,
    sent,
    failed,
    conversionRate,
    sentRate: total > 0 ? `${sentRatePct}% de sucesso` : '',
  }
})

// === Subtitles ===
const headerSubtitle = computed(() => `Métricas e conversões — ${periodLabels[period.value]}`)
const chartSubtitle = computed(() => `Comparativo de eventos — ${periodLabels[period.value]}`)
const funnelSubtitle = computed(() => `Eventos do funil no período — ${periodLabels[period.value]}`)
const purchasesSubtitle = computed(() => `${recentPurchases.value.length} compras recentes`)

// === Icons ===
const icons = {
  eye: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor"><path d="M10 12a2 2 0 100-4 2 2 0 000 4z"/><path fill-rule="evenodd" d="M.458 10C1.732 5.943 5.522 3 10 3s8.268 2.943 9.542 7c-1.274 4.057-5.064 7-9.542 7S1.732 14.057.458 10zM14 10a4 4 0 11-8 0 4 4 0 018 0z" clip-rule="evenodd"/></svg>',
  cart: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor"><path d="M3 1a1 1 0 000 2h1.22l.305 1.222a.997.997 0 00.01.042l1.358 5.43-.893.892C3.74 11.846 4.632 14 6.414 14H15a1 1 0 100-2H6.414l1-1H14a1 1 0 00.894-.553l3-6A1 1 0 0017 3H6.28l-.31-1.243A1 1 0 005 1H3z"/><circle cx="7" cy="17" r="2"/><circle cx="15" cy="17" r="2"/></svg>',
  checkout: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M5 9V7a5 5 0 0110 0v2a2 2 0 012 2v5a2 2 0 01-2 2H5a2 2 0 01-2-2v-5a2 2 0 012-2zm8-2v2H7V7a3 3 0 016 0z" clip-rule="evenodd"/></svg>',
  lead: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M10 9a3 3 0 100-6 3 3 0 000 6zm-7 9a7 7 0 1114 0H3z" clip-rule="evenodd"/></svg>',
  purchase: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor"><path d="M2 10a8 8 0 018-8v8h8a8 8 0 11-16 0z"/><path d="M12 2.252A8.014 8.014 0 0117.748 8H12V2.252z"/></svg>',
  trending: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M12 7a1 1 0 110-2h5a1 1 0 011 1v5a1 1 0 11-2 0V8.414l-4.293 4.293a1 1 0 01-1.414 0L8 10.414l-4.293 4.293a1 1 0 01-1.414-1.414l5-5a1 1 0 011.414 0L11 10.586 14.586 7H12z" clip-rule="evenodd"/></svg>',
  check: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd"/></svg>',
  alert: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clip-rule="evenodd"/></svg>',
  layers: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor"><path d="M10.394 2.08a1 1 0 00-.788 0l-7 3a1 1 0 000 1.84L5.25 7.99l5.046-2.205a.75.75 0 01.6 1.373L6.3 9.094l3.306 1.417a1 1 0 00.788 0l7-3a1 1 0 000-1.84l-7-3zM3.31 9.397L5 10.12v4.102a8.969 8.969 0 00-1.05-.174 1 1 0 01-.89-.89 11.115 11.115 0 01.25-3.762zM9.3 16.573A9.026 9.026 0 007 14.935v-3.957l1.818.78a3 3 0 002.364 0l5.508-2.361a11.026 11.026 0 01.25 3.762 1 1 0 01-.89.89 8.968 8.968 0 00-5.35 2.524 1 1 0 01-1.4 0zM6 18a1 1 0 001-1v-2.065a8.935 8.935 0 00-2-.712V17a1 1 0 001 1z"/></svg>',
}

// === Fetch ===
const fetchAll = async () => {
  loading.value = true
  try {
    const from = periodToFromIso(period.value)
    const to = new Date().toISOString()
    const granularity = granularityFor(period.value)

    const [summary, series, purchases, geo] = await Promise.all([
      dashboard.summary({ from, to }),
      dashboard.series({ from, to, granularity }),
      dashboard.recentPurchases({ from, to, limit: 10 }),
      dashboard.geoStats({ from, to }),
    ])

    summaryData.value = summary
    seriesData.value = series
    recentPurchases.value = purchases.data || []
    geoData.value = geo || []
  } catch (error: any) {
    toast.error('Erro', error.message || 'Não foi possível carregar a dashboard')
  } finally {
    loading.value = false
  }
}

const refresh = () => fetchAll()

watch(period, fetchAll)
onMounted(fetchAll)
</script>

<style scoped>
.dashboard-page {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
  gap: 1rem;
  flex-wrap: wrap;
}

.page-title {
  font-family: 'Instrument Sans', sans-serif;
  font-size: 1.875rem;
  font-weight: 700;
  color: var(--text-strong);
  margin-bottom: 0.25rem;
}

.page-subtitle {
  color: var(--text-2);
  font-size: 0.9375rem;
}

.header-controls {
  display: flex;
  gap: 0.625rem;
  align-items: center;
  flex-wrap: wrap;
}

.kpis-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 1rem;
}

.health-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 1rem;
}

.grid-2 {
  display: grid;
  grid-template-columns: 1.6fr 1fr;
  gap: 1rem;
}

.grid-2-globe {
  display: grid;
  grid-template-columns: 1.6fr 1fr;
  gap: 1rem;
}

@media (max-width: 1023px) {
  .grid-2,
  .grid-2-globe {
    grid-template-columns: 1fr;
  }
}
</style>
