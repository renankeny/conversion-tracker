import { ref, watch, onMounted, type ComputedRef } from 'vue'
import { useDashboardStore } from '~/stores/useDashboardStore'
import { useTenantStore } from '~/stores/useTenantStore'
import { errorMessage } from '~/utils/errorMessage'
import type {
  CounterSummaryResponse, CounterSeriesResponse,
  DashboardHeatmap, GeoStat, GroupedEvent, PanelAudit, PreviousPeriodMetrics,
  TrafficQualityStats, SourceBreakdown, AdPlatformLiveInsights, GoogleKeywordInsight,
} from '#shared/types'
import type { MetaCreativeInsight } from '~/composables/useDashboard'

export const useDashboardFetch = (selectedPixel: ComputedRef<string>) => {
  const dashboard = useDashboard()
  const toast = useToast()
  const pixelStore = usePixelStore()
  const accountsApi = useAccounts()
  const dashboardStore = useDashboardStore()
  const tenantStore = useTenantStore()

  const loading = ref(false)
  const adsLoading = ref(false)
  const initialized = ref(false)
  const summaryData = ref<CounterSummaryResponse | null>(null)
  const seriesData = ref<CounterSeriesResponse | null>(null)
  const trafficQuality = ref<TrafficQualityStats | undefined>()
  const sourceBreakdown = ref<SourceBreakdown | null>(null)
  const panelAudit = ref<PanelAudit | null>(null)
  const previousPeriod = ref<PreviousPeriodMetrics | null>(null)
  const heatmap = ref<DashboardHeatmap | null>(null)
  const metaInsights = ref<AdPlatformLiveInsights | null>(null)
  const googleInsights = ref<AdPlatformLiveInsights | null>(null)
  const metaCreatives = ref<MetaCreativeInsight[]>([])
  const googleKeywords = ref<GoogleKeywordInsight[]>([])
  const eventsFilterMap = ref<Record<string, string[]>>({})
  const recentPurchases = ref<GroupedEvent[]>([])
  const geoData = ref<GeoStat[]>([])
  const totalRevenue = ref(0)

  let fetchSeq = 0

  const clearAdsInsights = () => {
    metaInsights.value = null
    googleInsights.value = null
    metaCreatives.value = []
    googleKeywords.value = []
  }

  const ensurePixelSelected = () => {
    if (pixelStore.pixels.length && (!selectedPixel.value || !pixelStore.pixels.some((p) => p.name === selectedPixel.value))) {
      dashboardStore.setSelectedPixel(pixelStore.pixels[0]?.name || '')
    }
  }

  const fetchAll = async () => {
    ensurePixelSelected()
    const seq = ++fetchSeq
    const px = selectedPixel.value
    loading.value = true
    adsLoading.value = Boolean(px)
    clearAdsInsights()

    try {
      const { from, to } = dashboardStore.getDateRange()
      const ts = dashboardStore.trafficSource
      const [summary, series, purchases, geo, geoPurchases, revenue] = await Promise.all([
        dashboard.summary({ from, to, pixel: px, traffic_source: ts }),
        dashboard.series({ from, to, granularity: dashboardStore.granularityFor, pixel: px, traffic_source: ts }),
        dashboard.recentPurchases({ from, to, limit: 50, pixel: px, traffic_source: ts }),
        dashboard.geoStats({ from, to, pixel: px, traffic_source: ts }),
        dashboard.geoStatsByEvent({ from, to, event: 'Purchase', pixel: px, traffic_source: ts }),
        dashboard.revenue({ from, to, pixel: px, traffic_source: ts }),
      ])
      if (seq !== fetchSeq) return

      summaryData.value = summary
      seriesData.value = series
      recentPurchases.value = purchases.data || []
      totalRevenue.value = revenue.total_revenue ?? 0
      trafficQuality.value = revenue.traffic_quality
      sourceBreakdown.value = revenue.source_breakdown || null
      panelAudit.value = revenue.panel_audit || null
      previousPeriod.value = revenue.previous_period || null
      heatmap.value = revenue.heatmap || null
      geoData.value = [
        ...(geo || []).map((g) => ({ ...g, event_name: 'PageView' })),
        ...(geoPurchases || []),
      ]

      if (px) {
        const [mLive, mRank, gLive, gRank] = await Promise.all([
          dashboard.metaLiveInsights({ pixel_name: px, from, to }).catch(() => null),
          dashboard.metaCreativeRanking({ pixel_name: px, from, to }).catch(() => null),
          dashboard.googleLiveInsights({ pixel_name: px, from, to }).catch(() => null),
          dashboard.googleKeywordRanking({ pixel_name: px, from, to }).catch(() => null),
        ])
        if (seq !== fetchSeq) return
        metaInsights.value = mLive
        metaCreatives.value = (Array.isArray(mRank) ? mRank : []).slice(0, 3)
        googleInsights.value = gLive
        googleKeywords.value = (Array.isArray(gRank) ? gRank : []).slice(0, 5)
      }
    } catch (err: unknown) {
      if (seq !== fetchSeq) return
      toast.error('Erro', errorMessage(err, 'Não foi possível carregar a dashboard'))
    } finally {
      if (seq === fetchSeq) {
        loading.value = false
        adsLoading.value = false
      }
    }
  }

  const loadEventsFilter = async () => {
    try { eventsFilterMap.value = (await accountsApi.getEventsFilter()).events_filter || {} }
    catch { eventsFilterMap.value = {} }
  }

  const reloadForTenant = async (forcePixels: boolean) => {
    initialized.value = false
    clearAdsInsights()
    await Promise.all([pixelStore.fetch(forcePixels).catch(() => {}), loadEventsFilter()])
    ensurePixelSelected()
    initialized.value = true
    await fetchAll()
  }

  watch(
    [() => dashboardStore.period, () => dashboardStore.customFrom, () => dashboardStore.customTo, () => dashboardStore.trafficSource, selectedPixel],
    () => {
      if (!initialized.value) return
      if (dashboardStore.period === 'custom' && (!dashboardStore.customFrom || !dashboardStore.customTo)) return
      void fetchAll()
    },
  )

  watch(() => tenantStore.switchVersion, () => {
    void reloadForTenant(true)
  })

  onMounted(() => {
    void reloadForTenant(false)
  })

  return {
    loading, adsLoading, summaryData, seriesData, trafficQuality, sourceBreakdown,
    panelAudit, previousPeriod, heatmap, metaInsights, googleInsights, metaCreatives,
    googleKeywords, eventsFilterMap, recentPurchases, geoData, totalRevenue,
    refresh: () => fetchAll(),
  }
}

