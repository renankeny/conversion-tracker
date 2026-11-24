<template>
  <div class="public-dashboard-view">
    <div class="page-header">
      <div>
        <div class="title-wrap">
          <h1 class="page-title">{{ pixelLabel || 'Dashboard' }}</h1>
          <span class="live-pill" title="Atualização automática ativa">
            <span class="pulse-dot" />
            <span>LIVE</span>
          </span>
        </div>
        <p class="page-subtitle">{{ headerSubtitle }}</p>
      </div>

      <div class="header-controls">
        <BaseDropdown v-model="trafficSource" :options="sourceOptions" />
        <PeriodSelector v-model="period" v-model:customFrom="customFrom" v-model:customTo="customTo" />
        <BaseButton
          variant="ghost"
          size="sm"
          :title="privacyMode ? 'Mostrar valores financeiros' : 'Ocultar valores financeiros'"
          @click="privacyMode = !privacyMode"
        >
          <EyeOff v-if="privacyMode" :size="15" />
          <Eye v-else :size="15" />
        </BaseButton>
        <BaseButton variant="outline" size="sm" :loading="loading || isRefreshing" @click="refresh">Atualizar</BaseButton>
      </div>
    </div>

    <div v-if="loadError" class="error-banner">
      <span>{{ loadError }}</span>
      <BaseButton variant="primary" size="sm" :loading="loading" @click="retryLoad">Tentar novamente</BaseButton>
    </div>

    <SalesGoalProgressCard
      v-if="salesGoal"
      :goal-type="salesGoal.type"
      :target="salesGoal.target"
      :current="salesGoal.current"
      :currency="currency"
      :period-label="salesGoal.periodLabel"
      :blurred="privacyMode"
    />

    <div class="section-header">
      <h3 class="section-title">Visão Executiva & Auditoria de Envios</h3>
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
        :blurred="c.blurred"
        :comparisons="c.comparisons"
        :delivery="c.delivery"
      />
    </div>

    <div class="section-header">
      <h3 class="section-title">Top Criativos & Palavras-Chave</h3>
    </div>
    <div class="grid grid-cols-1 gap-6" :class="trafficSource === 'all' ? 'xl:grid-cols-2' : ''">
      <MetaCreativeRankingCard
        v-if="trafficSource !== 'google'"
        :creatives="metaCreatives"
        :loading="adsLoading"
        :blurred="privacyMode"
        :currency="currency"
      />
      <GoogleKeywordRankingCard
        v-if="trafficSource !== 'meta'"
        :keywords="googleKeywords"
        :loading="adsLoading"
        :blurred="privacyMode"
        :currency="currency"
      />
    </div>

    <TrackableLinksCard :overview="trackableLinks" :currency="currency" :blurred="privacyMode" public-mode />
    <DashboardHeatmapCard :heatmap="heatmap" />
    <TrafficQualityCard :stats="trafficQuality" />

    <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
      <EventsChart :points="seriesData?.points || []" :granularity="seriesData?.granularity || granularity" :subtitle="chartSubtitle" />

      <div class="funnel-wrapper">
        <div class="section-header">
          <h3 class="section-title">Funil de conversão</h3>
        </div>
        <ConversionFunnel :events="summaryData?.events || []" :steps="funnelSteps" :lead-total="panelLeadsTotal" :subtitle="funnelSubtitle" />
      </div>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
      <GeoGlobe :stats="geoData" :from="dateRange.from" :to="dateRange.to" />
      <RecentPurchases
        :purchases="recentPurchases"
        :subtitle="purchasesSubtitle"
        :currency="currency"
        :blurred="privacyMode"
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
  </div>
</template>

<script setup lang="ts">
import { Eye, EyeOff } from 'lucide-vue-next'
import KpiCard from '~/components/dashboard/KpiCard.vue'
import SalesGoalProgressCard from '~/components/dashboard/SalesGoalProgressCard.vue'
import MetaCreativeRankingCard from '~/components/dashboard/MetaCreativeRankingCard.vue'
import GoogleKeywordRankingCard from '~/components/dashboard/GoogleKeywordRankingCard.vue'
import DashboardHeatmapCard from '~/components/dashboard/DashboardHeatmapCard.vue'
import TrackableLinksCard from '~/components/dashboard/TrackableLinksCard.vue'
import TrafficQualityCard from '~/components/dashboard/TrafficQualityCard.vue'
import EventsChart from '~/components/dashboard/EventsChart.vue'
import ConversionFunnel from '~/components/dashboard/ConversionFunnel.vue'
import GeoGlobe from '~/components/dashboard/GeoGlobe.vue'
import RecentPurchases from '~/components/dashboard/RecentPurchases.vue'
import PeriodSelector from '~/components/dashboard/PeriodSelector.vue'
import EventDetailsModal from '~/components/events/EventDetailsModal.vue'
import { usePublicDashboardView } from '~/composables/usePublicDashboardView'

const route = useRoute()
const token = route.params.token as string
const { formatTitle } = useBranding()

const sourceOptions = [
  { value: 'all', label: 'Todas as fontes' },
  { value: 'meta', label: 'Meta Ads' },
  { value: 'google', label: 'Google Ads' },
]

const {
  period, customFrom, customTo, trafficSource, privacyMode, dateRange, granularity,
  loading, isRefreshing, adsLoading, loadError, pixelLabel, currency,
  headerSubtitle, chartSubtitle, funnelSubtitle, purchasesSubtitle, periodLabel,
  summaryData, seriesData, recentPurchases, geoData, trafficQuality, heatmap, trackableLinks,
  metaCreatives, googleKeywords, panelLeadsTotal, salesGoal, visibleCards, funnelSteps,
  purchaseModalOpen, purchaseSelected, onPurchaseSelect, refresh, retryLoad,
} = usePublicDashboardView(token)

useHead({ title: computed(() => formatTitle(pixelLabel.value || 'Dashboard')) })
</script>

<style scoped>
.public-dashboard-view { display: flex; flex-direction: column; gap: 1.5rem; width: 100%; }
.error-banner { display: flex; align-items: center; justify-content: space-between; gap: 1rem; padding: 0.75rem 1rem; border-radius: 10px; background: var(--danger-soft); border: 1px solid var(--danger); color: var(--danger); font-size: var(--fs-md); flex-wrap: wrap; }
.page-header { display: flex; justify-content: space-between; align-items: flex-end; gap: 1rem; flex-wrap: wrap; }
.title-wrap { display: flex; align-items: center; gap: 0.5rem; }
.page-title { font-family: var(--font-sans); font-size: var(--fs-3xl); font-weight: var(--fw-bold); color: var(--text-strong); letter-spacing: -0.02em; margin: 0; }
.live-pill { display: inline-flex; align-items: center; gap: 0.35rem; padding: 0.15rem 0.45rem; background: var(--brand-soft); border: 1px solid var(--brand-border); border-radius: 9999px; font-size: var(--fs-2xs); font-weight: var(--fw-bold); color: var(--brand); letter-spacing: 0.06em; }
.pulse-dot { width: 6px; height: 6px; border-radius: 50%; background: var(--brand); box-shadow: 0 0 6px var(--brand); animation: livePulse 1.8s infinite; }
@keyframes livePulse { 0%, 100% { transform: scale(0.95); opacity: 0.8; } 50% { transform: scale(1.3); opacity: 1; box-shadow: 0 0 10px var(--brand); } }
.page-subtitle { color: var(--text-3); font-size: var(--fs-base); margin-top: 0.25rem; }
.header-controls { display: flex; gap: 0.625rem; align-items: center; flex-wrap: wrap; }
.section-header { display: flex; align-items: center; justify-content: space-between; gap: 0.75rem; margin-bottom: -0.75rem; }
.section-title { font-size: var(--fs-md); font-weight: var(--fw-semibold); color: var(--text-2); text-transform: uppercase; letter-spacing: 0.04em; margin: 0; }
.funnel-wrapper { display: flex; flex-direction: column; gap: 1rem; }
.funnel-wrapper > .section-header { margin-bottom: -0.5rem; }
</style>
