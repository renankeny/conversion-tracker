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

      <!-- Cards Executivos & Auditoria de Envios -->
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

      <!-- Top 3 Criativos Meta Ads & Top 5 Palavras-Chave Google Ads -->
      <div class="section-header">
        <h3 class="section-title">Top Criativos & Palavras-Chave</h3>
      </div>
      <div
        class="grid grid-cols-1 gap-6"
        :class="trafficSource === 'all' ? 'xl:grid-cols-2' : ''"
      >
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

      <!-- Mapa de Calor Dia × Hora -->
      <DashboardHeatmapCard :heatmap="heatmap" />

      <!-- Auditoria de Tráfego & Bots -->
      <TrafficQualityCard :stats="trafficQuality" />

      <!-- Série Temporal e Funil de Conversão -->
      <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
        <EventsChart :points="seriesData?.points || []" :granularity="seriesData?.granularity || granularity" :subtitle="chartSubtitle" />

        <div class="funnel-wrapper">
          <div class="section-header">
            <h3 class="section-title">Funil de conversão</h3>
          </div>
          <ConversionFunnel :events="summaryData?.events || []" :steps="funnelSteps" :lead-total="panelLeadsTotal" :subtitle="funnelSubtitle" />
        </div>
      </div>

      <!-- Mapa de Localização + Vendas Recentes -->
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
    </template>
  </div>
</template>

<script setup lang="ts">
import { Eye, EyeOff } from 'lucide-vue-next'
import KpiCard from '~/components/dashboard/KpiCard.vue'
import MetaCreativeRankingCard from '~/components/dashboard/MetaCreativeRankingCard.vue'
import GoogleKeywordRankingCard from '~/components/dashboard/GoogleKeywordRankingCard.vue'
import DashboardHeatmapCard from '~/components/dashboard/DashboardHeatmapCard.vue'
import TrafficQualityCard from '~/components/dashboard/TrafficQualityCard.vue'
import EventsChart from '~/components/dashboard/EventsChart.vue'
import ConversionFunnel from '~/components/dashboard/ConversionFunnel.vue'
import GeoGlobe from '~/components/dashboard/GeoGlobe.vue'
import RecentPurchases from '~/components/dashboard/RecentPurchases.vue'
import PeriodSelector from '~/components/dashboard/PeriodSelector.vue'
import EventDetailsModal from '~/components/events/EventDetailsModal.vue'
import { usePublicDashboardView } from '~/composables/usePublicDashboardView'

definePageMeta({ layout: 'default' })

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
  loading, isRefreshing, adsLoading, notFound, metaLoaded, loadError, pixelLabel, currency,
  headerSubtitle, chartSubtitle, funnelSubtitle, purchasesSubtitle,
  summaryData, seriesData, recentPurchases, geoData, trafficQuality, heatmap,
  metaCreatives, googleKeywords, panelLeadsTotal, visibleCards, funnelSteps,
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
.title-wrap { display: flex; align-items: center; gap: 0.5rem; }
.page-title { font-family: var(--font-sans); font-size: var(--fs-3xl); font-weight: var(--fw-bold); color: var(--text-strong); letter-spacing: -0.02em; margin-bottom: 0; }
.live-pill { display: inline-flex; align-items: center; gap: 0.35rem; padding: 0.15rem 0.45rem; background: var(--brand-soft); border: 1px solid var(--brand-border); border-radius: 9999px; font-size: var(--fs-2xs); font-weight: var(--fw-bold); color: var(--brand); letter-spacing: 0.06em; }
.pulse-dot { width: 6px; height: 6px; border-radius: 50%; background: var(--brand); box-shadow: 0 0 6px var(--brand); animation: livePulse 1.8s infinite; }
@keyframes livePulse { 0%, 100% { transform: scale(0.95); opacity: 0.8; } 50% { transform: scale(1.3); opacity: 1; box-shadow: 0 0 10px var(--brand); } }
.page-subtitle { color: var(--text-3); font-size: var(--fs-base); margin-top: 0.25rem; }
.header-controls { display: flex; gap: 0.625rem; align-items: center; flex-wrap: wrap; }
.section-header { display: flex; align-items: center; justify-content: space-between; gap: 0.75rem; margin-bottom: -0.75rem; }
.section-title { font-size: var(--fs-md); font-weight: var(--fw-semibold); color: var(--text-2); text-transform: uppercase; letter-spacing: 0.04em; margin: 0; }
.funnel-wrapper { display: flex; flex-direction: column; gap: 1rem; }
.funnel-wrapper > .section-header { margin-bottom: -0.5rem; }
@media (max-width: 640px) { .public-dashboard-page { padding: 1rem 0.875rem 2rem; } .page-header { align-items: stretch; } }
</style>
