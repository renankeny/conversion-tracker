<template>
  <div class="dashboard-page">
    <DashboardFilterBar
      :selected-pixel-obj="selectedPixelObj"
      :header-subtitle="headerSubtitle"
      :loading="loading"
      @refresh="refresh"
    />

    <HealthAlertsBanner />

    <!-- Cards Executivos & Auditoria (Meta / Google × Painel + Enviados / Não enviados) -->
    <div class="section-header">
      <h3 class="section-title">Visão Executiva & Auditoria de Envios</h3>
      <div class="section-actions">
        <BaseButton
          v-if="hasCustomCardPrefs"
          variant="ghost"
          size="sm"
          :loading="savingCardsPrefs"
          @click="onResetCardsDefault"
        >
          Padrão executivo
        </BaseButton>
        <BaseButton variant="ghost" size="sm" :disabled="!selectedPixelObj" @click="cardsEditorOpen = true">
          Editar cards
        </BaseButton>
      </div>
    </div>
    <p v-if="!selectedPixelObj" class="section-hint">Selecione um pixel específico para personalizar cards e funil.</p>

    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
      <KpiCard
        v-for="card in visibleCards"
        :key="card.id"
        :label="card.label"
        :value="card.value"
        :format="card.format"
        :variant="card.variant"
        :icon="card.icon"
        :change-percent="card.changePercent"
        :change-inverse="card.changeInverse"
        :previous-label="card.previousLabel"
        :blurred="card.blurred"
        :comparisons="card.comparisons"
        :delivery="card.delivery"
      />
    </div>

    <!-- Top 3 Criativos Meta Ads & Top 5 Palavras-Chave Google Ads -->
    <div class="section-header">
      <h3 class="section-title">Top Criativos & Palavras-Chave</h3>
    </div>
    <div
      class="grid grid-cols-1 gap-6"
      :class="dashboardStore.trafficSource === 'all' ? 'xl:grid-cols-2' : ''"
    >
      <MetaCreativeRankingCard
        v-if="dashboardStore.trafficSource !== 'google'"
        :creatives="metaCreatives"
        :loading="adsLoading"
        :blurred="dashboardStore.privacyMode"
      />
      <GoogleKeywordRankingCard
        v-if="dashboardStore.trafficSource !== 'meta'"
        :keywords="googleKeywords"
        :loading="adsLoading"
        :blurred="dashboardStore.privacyMode"
      />
    </div>

    <!-- Mapa de Calor Dia × Hora (Mensagens, Leads e Vendas) -->
    <DashboardHeatmapCard :heatmap="heatmap" />

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

    <!-- Mapa de Localização + Vendas Recentes -->
    <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
      <GeoGlobe :stats="geoData" :from="dashboardStore.getDateRange().from" :to="dashboardStore.getDateRange().to" />
      <RecentPurchases :purchases="recentPurchases" :subtitle="purchasesSubtitle" @select="onPurchaseSelect" />
    </div>

    <!-- Modais -->
    <EventDetailsModal v-model="purchaseModalOpen" :source="purchaseSelected" />

    <EventPickerModal
      v-model="cardsEditorOpen"
      title="Escolher cards da Dashboard"
      description="Selecione, ordene (setas) e renomeie os cards de Eventos, Meta Ads, Google Ads e Auditoria."
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
import KpiCard from '~/components/dashboard/KpiCard.vue'
import MetaCreativeRankingCard from '~/components/dashboard/MetaCreativeRankingCard.vue'
import GoogleKeywordRankingCard from '~/components/dashboard/GoogleKeywordRankingCard.vue'
import DashboardHeatmapCard from '~/components/dashboard/DashboardHeatmapCard.vue'
import EventsChart from '~/components/dashboard/EventsChart.vue'
import ConversionFunnel from '~/components/dashboard/ConversionFunnel.vue'
import GeoGlobe from '~/components/dashboard/GeoGlobe.vue'
import RecentPurchases from '~/components/dashboard/RecentPurchases.vue'
import HealthAlertsBanner from '~/components/dashboard/HealthAlertsBanner.vue'
import TrafficQualityCard from '~/components/dashboard/TrafficQualityCard.vue'
import EventDetailsModal from '~/components/events/EventDetailsModal.vue'
import EventPickerModal from '~/components/dashboard/EventPickerModal.vue'
import { useDashboardView } from '~/composables/useDashboardView'

definePageMeta({
  layout: 'app',
  middleware: 'auth',
})

const { formatTitle } = useBranding()
useHead({ title: computed(() => formatTitle('Dashboard')) })

const {
  dashboardStore, loading, adsLoading, summaryData, seriesData, trafficQuality, heatmap, filteredSeriesPoints,
  metaCreatives, googleKeywords, recentPurchases, geoData, purchaseModalOpen, purchaseSelected, onPurchaseSelect,
  headerSubtitle, chartSubtitle, funnelSubtitle, purchasesSubtitle, selectedPixelObj, hasCustomCardPrefs,
  cardsEditorOpen, savingCardsPrefs, enabledCardIds, cardOrder, cardPickerOptions, visibleCards, onSaveCards, onResetCardsDefault,
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
  gap: 0.75rem;
  margin-bottom: -0.75rem;
}

.section-actions {
  display: flex;
  align-items: center;
  gap: 0.35rem;
}

.section-title {
  font-size: var(--fs-md);
  font-weight: var(--fw-semibold);
  color: var(--text-2);
  text-transform: uppercase;
  letter-spacing: 0.04em;
  margin: 0;
}

.section-hint {
  font-size: var(--fs-sm);
  color: var(--text-3);
  margin: -0.75rem 0 0;
}

.funnel-wrapper {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.funnel-wrapper > .section-header {
  margin-bottom: -0.5rem;
}
</style>
