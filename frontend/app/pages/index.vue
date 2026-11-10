<template>
  <div class="dashboard-page">
    <DashboardFilterBar
      :selected-pixel-obj="selectedPixelObj"
      :header-subtitle="headerSubtitle"
      :loading="loading"
      @refresh="refresh"
    />

    <!-- Banner de Diagnósticos e Alertas de Saúde -->
    <HealthAlertsBanner />

    <!-- Cards de Eventos -->
    <div class="section-header">
      <h3 class="section-title">Eventos</h3>
      <BaseButton variant="ghost" size="sm" :disabled="!selectedPixelObj" @click="cardsEditorOpen = true">
        Editar cards
      </BaseButton>
    </div>
    <p v-if="!selectedPixelObj" class="section-hint">Selecione um pixel específico para personalizar cards e funil.</p>
    <div class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-4 gap-4">
      <KpiCard
        v-for="card in visibleCards"
        :key="card.id"
        :label="card.label"
        :value="card.value"
        :format="card.format"
        :variant="card.variant"
        :icon="card.icon"
      />
    </div>

    <!-- Auditoria de Tráfego & Bots -->
    <TrafficQualityCard :stats="trafficQuality" />

    <!-- Série Temporal e Funil de Conversão -->
    <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
      <EventsChart :points="filteredSeriesPoints" :granularity="seriesData?.granularity || 'day'" :subtitle="chartSubtitle" />

      <div class="funnel-wrapper">
        <div class="section-header">
          <h3 class="section-title">Funil de conversão</h3>
          <BaseButton variant="ghost" size="sm" :disabled="!selectedPixelObj" @click="funnelEditorOpen = true">
            Editar funil
          </BaseButton>
        </div>
        <ConversionFunnel :events="summaryData?.events || []" :steps="funnelSteps" :subtitle="funnelSubtitle" />
      </div>
    </div>

    <!-- Mapa de Localização + Vendas -->
    <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
      <GeoGlobe :stats="geoData" :from="dashboardStore.getDateRange().from" :to="dashboardStore.getDateRange().to" />
      <RecentPurchases :purchases="recentPurchases" :subtitle="purchasesSubtitle" @select="onPurchaseSelect" />
    </div>

    <!-- Modais -->
    <EventDetailsModal v-model="purchaseModalOpen" :source="purchaseSelected" />

    <EventPickerModal
      v-model="cardsEditorOpen"
      title="Escolher cards"
      description="Selecione, ordene (setas) e renomeie os cards deste pixel."
      :events="cardPickerOptions"
      :selected="enabledCardIds"
      :order="cardOrder"
      :saving="savingCardsPrefs"
      editable-labels
      :label-overrides="eventLabelOverrides"
      @save="onSaveCards"
    />

    <EventPickerModal
      v-model="funnelEditorOpen"
      title="Escolher eventos do funil"
      description="Selecione e ordene os eventos do funil deste pixel (independente dos cards)."
      :events="funnelPickerOptions"
      :selected="enabledFunnelEvents"
      :order="funnelOrder"
      :saving="savingFunnelPrefs"
      @save="onSaveFunnelEvents"
    />
  </div>
</template>

<script setup lang="ts">
import DashboardFilterBar from '~/components/dashboard/DashboardFilterBar.vue'
import HealthAlertsBanner from '~/components/dashboard/HealthAlertsBanner.vue'
import TrafficQualityCard from '~/components/dashboard/TrafficQualityCard.vue'
import KpiCard from '~/components/dashboard/KpiCard.vue'
import EventsChart from '~/components/dashboard/EventsChart.vue'
import ConversionFunnel from '~/components/dashboard/ConversionFunnel.vue'
import GeoGlobe from '~/components/dashboard/GeoGlobe.vue'
import RecentPurchases from '~/components/dashboard/RecentPurchases.vue'
import EventPickerModal from '~/components/dashboard/EventPickerModal.vue'
import EventDetailsModal from '~/components/events/EventDetailsModal.vue'

definePageMeta({
  layout: 'app',
  middleware: 'auth',
})

const { formatTitle } = useBranding()
useHead({ title: computed(() => formatTitle('Dashboard')) })

const {
  dashboardStore, loading, summaryData, seriesData, trafficQuality, filteredSeriesPoints,
  recentPurchases, geoData, purchaseModalOpen, purchaseSelected, onPurchaseSelect,
  headerSubtitle, chartSubtitle, funnelSubtitle, purchasesSubtitle, selectedPixelObj,
  cardsEditorOpen, savingCardsPrefs, enabledCardIds, cardOrder, cardPickerOptions, visibleCards, onSaveCards,
  funnelEditorOpen, savingFunnelPrefs, enabledFunnelEvents, funnelOrder, funnelPickerOptions, funnelSteps, onSaveFunnelEvents,
  eventLabelOverrides, refresh,
} = useDashboardView()
</script>

<style scoped>
.dashboard-page {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: -0.75rem;
}

.section-title {
  font-size: var(--fs-base);
  font-weight: var(--fw-semibold);
  text-transform: uppercase;
  letter-spacing: 0.03em;
  color: var(--text-2);
}

.section-hint {
  font-size: var(--fs-sm);
  color: var(--text-3);
  margin-top: -0.5rem;
}

.funnel-wrapper {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  height: 100%;
}

.funnel-wrapper .section-header {
  margin-bottom: 0;
}
</style>
