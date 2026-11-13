import { defineStore } from 'pinia'
import { ref } from 'vue'
import type {
  AdPlatformLiveInsights,
  CounterSeriesResponse,
  CounterSummaryResponse,
  DashboardHeatmap,
  GoogleKeywordInsight,
  GroupedEvent,
  PanelAudit,
  PreviousPeriodMetrics,
  SourceBreakdown,
  TrafficQualityStats,
} from '#shared/types'
import type { GeoStat } from '~/utils/geoCoordinates'
import type { MetaCreativeInsight } from '~/composables/useDashboard'

export const useDashboardDataStore = defineStore('dashboardData', () => {
  const coreKey = ref('')
  const coreFetchedAt = ref(0)
  const adsKey = ref('')
  const adsFetchedAt = ref(0)
  const metaRankLoaded = ref(false)
  const googleRankLoaded = ref(false)

  const summaryData = ref<CounterSummaryResponse | null>(null)
  const seriesData = ref<CounterSeriesResponse | null>(null)
  const trafficQuality = ref<TrafficQualityStats | undefined>(undefined)
  const sourceBreakdown = ref<SourceBreakdown | null>(null)
  const panelAudit = ref<PanelAudit | null>(null)
  const previousPeriod = ref<PreviousPeriodMetrics | null>(null)
  const heatmap = ref<DashboardHeatmap | null>(null)
  const recentPurchases = ref<GroupedEvent[]>([])
  const geoData = ref<GeoStat[]>([])
  const totalRevenue = ref(0)

  const metaInsights = ref<AdPlatformLiveInsights | null>(null)
  const googleInsights = ref<AdPlatformLiveInsights | null>(null)
  const metaCreatives = ref<MetaCreativeInsight[]>([])
  const googleKeywords = ref<GoogleKeywordInsight[]>([])

  function clearAds(): void {
    adsKey.value = ''
    adsFetchedAt.value = 0
    metaRankLoaded.value = false
    googleRankLoaded.value = false
    metaInsights.value = null
    googleInsights.value = null
    metaCreatives.value = []
    googleKeywords.value = []
  }

  function clearAll(): void {
    coreKey.value = ''
    coreFetchedAt.value = 0
    summaryData.value = null
    seriesData.value = null
    trafficQuality.value = undefined
    sourceBreakdown.value = null
    panelAudit.value = null
    previousPeriod.value = null
    heatmap.value = null
    recentPurchases.value = []
    geoData.value = []
    totalRevenue.value = 0
    clearAds()
  }

  return {
    coreKey,
    coreFetchedAt,
    adsKey,
    adsFetchedAt,
    metaRankLoaded,
    googleRankLoaded,
    summaryData,
    seriesData,
    trafficQuality,
    sourceBreakdown,
    panelAudit,
    previousPeriod,
    heatmap,
    recentPurchases,
    geoData,
    totalRevenue,
    metaInsights,
    googleInsights,
    metaCreatives,
    googleKeywords,
    clearAds,
    clearAll,
  }
})
