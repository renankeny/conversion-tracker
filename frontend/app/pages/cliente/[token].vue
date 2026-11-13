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
          <p class="page-subtitle">Visualização pública — {{ PUBLIC_PERIOD_LABELS[period] }}</p>
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
          :currency="currency"
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
        <EventsChart :points="seriesData?.points || []" :granularity="seriesData?.granularity || granularity" subtitle="Comparativo de eventos" />
        <ConversionFunnel :events="summaryData?.events || []" :steps="funnelSteps" :lead-total="panelLeadsTotal" subtitle="Eventos do funil no período" />
      </div>

      <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
        <GeoGlobe :stats="geoData" :from="dateRange.from" :to="dateRange.to" />
        <RecentPurchases
          :purchases="recentPurchases"
          :subtitle="`${recentPurchases.length} compras recentes`"
          :currency="currency"
          public-mode
          @select="onPurchaseSelect"
        />
      </div>

      <EventDetailsModal
        v-model="purchaseModalOpen"
        :source="purchaseSelected"
        :allow-user-details="false"
        :currency="currency"
        public-mode
      />
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
import { PUBLIC_PERIOD_LABELS, usePublicDashboardView } from '~/composables/usePublicDashboardView'

definePageMeta({ layout: 'default' })

const route = useRoute()
const token = route.params.token as string
const { formatTitle } = useBranding()

const {
  period, customFrom, customTo, dateRange, granularity, loading, notFound, metaLoaded, loadError, pixelLabel, currency,
  summaryData, seriesData, recentPurchases, geoData, heatmap, panelLeadsTotal, visibleCards, funnelSteps,
  purchaseModalOpen, purchaseSelected, onPurchaseSelect, refresh, retryLoad,
} = usePublicDashboardView(token)

useHead({ title: computed(() => formatTitle(pixelLabel.value || 'Dashboard')) })
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
