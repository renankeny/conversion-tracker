import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { computeSaoPauloRange, type DashboardTrafficSource, type PeriodValue } from '~/stores/useDashboardStore'
import { errorMessage } from '~/utils/errorMessage'
import type { GeoStat } from '~/utils/geoCoordinates'
import type { MetaCreativeInsight } from '~/composables/useDashboard'
import type {
  AdPlatformLiveInsights, CounterSummaryResponse, CounterSeriesResponse,
  DashboardConfig, DashboardHeatmap, GoogleKeywordInsight, GroupedEvent,
  PanelAudit, PreviousPeriodMetrics, SourceBreakdown, TrafficQualityStats,
} from '#shared/types'

const ADS_CACHE_TTL_MS = 120_000
const LIVE_POLL_MS = 30_000

const isAborted = (err: unknown): boolean => {
  if (!err || typeof err !== 'object') return false
  const e = err as { name?: string; message?: string; cause?: { name?: string } }
  return e.name === 'AbortError' || e.cause?.name === 'AbortError' || Boolean(e.message?.includes('aborted'))
}

export const usePublicDashboardFetch = (token: string) => {
  const api = usePublicDashboard(token)
  const { fetchBranding } = useBranding()

  const period = ref<PeriodValue>('today')
  const customFrom = ref('')
  const customTo = ref('')
  const trafficSource = ref<DashboardTrafficSource>('all')
  const privacyMode = ref(false)

  const dateRange = computed(() => computeSaoPauloRange(period.value, customFrom.value, customTo.value))
  const granularity = computed<'hour' | 'day'>(() => (period.value === 'today' || period.value === 'yesterday' ? 'hour' : 'day'))
  const periodIncludesToday = computed(() => period.value !== 'yesterday' && period.value !== 'custom')

  const loading = ref(false)
  const isRefreshing = ref(false)
  const adsLoading = ref(false)
  const notFound = ref(false)
  const metaLoaded = ref(false)
  const loadError = ref<string | null>(null)
  const pixelLabel = ref('')
  const currency = ref('BRL')
  const dashboardConfig = ref<DashboardConfig | null>(null)

  const summaryData = ref<CounterSummaryResponse | null>(null)
  const seriesData = ref<CounterSeriesResponse | null>(null)
  const recentPurchases = ref<GroupedEvent[]>([])
  const geoData = ref<GeoStat[]>([])
  const totalRevenue = ref(0)
  const trafficQuality = ref<TrafficQualityStats | undefined>(undefined)
  const sourceBreakdown = ref<SourceBreakdown | null>(null)
  const panelAudit = ref<PanelAudit | null>(null)
  const previousPeriod = ref<PreviousPeriodMetrics | null>(null)
  const heatmap = ref<DashboardHeatmap | null>(null)
  const metaInsights = ref<AdPlatformLiveInsights | null>(null)
  const googleInsights = ref<AdPlatformLiveInsights | null>(null)
  const metaCreatives = ref<MetaCreativeInsight[]>([])
  const googleKeywords = ref<GoogleKeywordInsight[]>([])

  let coreSeq = 0
  let adsSeq = 0
  let adsKey = ''
  let adsFetchedAt = 0
  let metaRankLoaded = false
  let googleRankLoaded = false
  let coreAbort: AbortController | null = null
  let adsAbort: AbortController | null = null
  let livePollTimer: ReturnType<typeof setInterval> | null = null

  const handleApiError = (err: unknown) => {
    if (isAborted(err)) return
    const code = err && typeof err === 'object' ? (err as { statusCode?: number }).statusCode : undefined
    if (code === 404 || code === 410) { notFound.value = true; loadError.value = null }
    else loadError.value = errorMessage(err, 'Não foi possível carregar os dados no momento.')
  }

  const fetchMeta = async () => {
    try {
      const res = await api.meta()
      pixelLabel.value = res.pixel_label
      currency.value = res.currency || 'BRL'
      dashboardConfig.value = res.dashboard_config
      metaLoaded.value = true
      loadError.value = null
    } catch (err: unknown) { handleApiError(err) }
  }

  const fetchCore = async (opts: { silent?: boolean } = {}) => {
    if (notFound.value || !metaLoaded.value) return
    coreAbort?.abort()
    const ac = new AbortController()
    coreAbort = ac
    const seq = ++coreSeq

    if (summaryData.value === null && !opts.silent) loading.value = true
    else if (!opts.silent) isRefreshing.value = true

    try {
      const { from, to } = dateRange.value
      const ts = trafficSource.value
      const [summary, series, purchases, geo, geoPurchases, revenue] = await Promise.all([
        api.summary({ from, to, traffic_source: ts }, ac.signal),
        api.series({ from, to, granularity: granularity.value, traffic_source: ts }, ac.signal),
        api.purchases({ from, to, traffic_source: ts }, ac.signal),
        api.geo({ from, to, traffic_source: ts }, ac.signal),
        api.geoByEvent({ from, to, event: 'Purchase', traffic_source: ts }, ac.signal),
        api.revenue({ from, to, traffic_source: ts }, ac.signal),
      ])
      if (seq !== coreSeq || ac.signal.aborted) return

      summaryData.value = summary
      seriesData.value = series
      recentPurchases.value = purchases.data || []
      totalRevenue.value = revenue.total_revenue ?? 0
      trafficQuality.value = revenue.traffic_quality
      sourceBreakdown.value = revenue.source_breakdown || null
      panelAudit.value = revenue.panel_audit || null
      previousPeriod.value = revenue.previous_period || null
      heatmap.value = revenue.heatmap || null
      geoData.value = [...(geo || []).map((g) => ({ ...g, event_name: 'PageView' })), ...(geoPurchases || [])]
      loadError.value = null
    } catch (err: unknown) {
      if (seq === coreSeq && !opts.silent) handleApiError(err)
    } finally {
      if (seq === coreSeq) { loading.value = false; isRefreshing.value = false }
    }
  }

  const fetchAds = async (opts: { force?: boolean } = {}) => {
    if (notFound.value || !metaLoaded.value) return
    const { from, to } = dateRange.value
    const ts = trafficSource.value
    const nextKey = `${from}|${to}`
    const sameWindow = !opts.force && adsKey === nextKey && Date.now() - adsFetchedAt < ADS_CACHE_TTL_MS
    const wantMetaRank = ts !== 'google' && (!sameWindow || !metaRankLoaded)
    const wantGoogleRank = ts !== 'meta' && (!sameWindow || !googleRankLoaded)

    if (sameWindow && !wantMetaRank && !wantGoogleRank) return

    adsAbort?.abort()
    const ac = new AbortController()
    adsAbort = ac
    const seq = ++adsSeq
    if (!sameWindow) {
      metaInsights.value = null; googleInsights.value = null
      metaCreatives.value = []; googleKeywords.value = []
      metaRankLoaded = false; googleRankLoaded = false
    }
    adsLoading.value = true

    try {
      const [ads, mRank, gRank] = await Promise.all([
        sameWindow ? Promise.resolve(null) : api.adsSummary({ from, to }, ac.signal).catch(() => null),
        wantMetaRank ? api.metaCreatives({ from, to }, ac.signal).catch(() => null) : Promise.resolve(null),
        wantGoogleRank ? api.googleKeywords({ from, to }, ac.signal).catch(() => null) : Promise.resolve(null),
      ])
      if (seq !== adsSeq || ac.signal.aborted) return

      if (!sameWindow) {
        metaInsights.value = ads?.meta || null
        googleInsights.value = ads?.google || null
        adsKey = nextKey
        adsFetchedAt = Date.now()
      }
      if (wantMetaRank) {
        metaCreatives.value = (Array.isArray(mRank) ? mRank : []).slice(0, 3)
        metaRankLoaded = true
      }
      if (wantGoogleRank) {
        googleKeywords.value = (Array.isArray(gRank) ? gRank : []).slice(0, 5)
        googleRankLoaded = true
      }
    } finally {
      if (seq === adsSeq) adsLoading.value = false
    }
  }

  const fetchAll = async (opts: { force?: boolean } = {}) => {
    await Promise.all([fetchCore(), fetchAds({ force: opts.force })])
  }

  const refresh = async () => {
    await fetchMeta()
    if (!notFound.value && metaLoaded.value) await fetchAll({ force: true })
  }

  const retryLoad = async () => {
    loading.value = true
    if (!metaLoaded.value) await fetchMeta()
    if (!notFound.value && metaLoaded.value) await fetchAll({ force: true })
    else loading.value = false
  }

  watch([period, customFrom, customTo, trafficSource], () => {
    if (period.value === 'custom' && (!customFrom.value || !customTo.value)) return
    void fetchAll()
  })

  onMounted(async () => {
    void fetchBranding()
    await retryLoad()
    livePollTimer = setInterval(() => {
      if (import.meta.client && document.visibilityState === 'visible' && metaLoaded.value && !notFound.value && periodIncludesToday.value) {
        void fetchCore({ silent: true })
      }
    }, LIVE_POLL_MS)
  })

  onUnmounted(() => {
    coreAbort?.abort()
    adsAbort?.abort()
    if (livePollTimer !== null) clearInterval(livePollTimer)
  })

  return {
    period, customFrom, customTo, trafficSource, privacyMode, dateRange, granularity,
    loading, isRefreshing, adsLoading, notFound, metaLoaded, loadError, pixelLabel, currency, dashboardConfig,
    summaryData, seriesData, recentPurchases, geoData, totalRevenue, trafficQuality,
    sourceBreakdown, panelAudit, previousPeriod, heatmap, metaInsights, googleInsights,
    metaCreatives, googleKeywords, refresh, retryLoad,
  }
}
