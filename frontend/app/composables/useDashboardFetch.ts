import { ref, watch, nextTick, onMounted, onUnmounted, type ComputedRef } from 'vue'
import { storeToRefs } from 'pinia'
import { useDashboardStore } from '~/stores/useDashboardStore'
import { useDashboardDataStore } from '~/stores/useDashboardDataStore'
import { useTenantStore } from '~/stores/useTenantStore'
import { useLiveEventsStore } from '~/stores/useLiveEventsStore'
import { errorMessage } from '~/utils/errorMessage'

const ADS_CACHE_TTL_MS = 120_000
const LIVE_DEBOUNCE_MS = 3_500

function isAbortError(err: unknown): boolean {
  if (!err || typeof err !== 'object') return false
  const e = err as { name?: string; message?: string; cause?: { name?: string } }
  return e.name === 'AbortError' || e.cause?.name === 'AbortError' || Boolean(e.message?.includes('aborted'))
}

export const useDashboardFetch = (selectedPixel: ComputedRef<string>) => {
  const dashboard = useDashboard()
  const toast = useToast()
  const pixelStore = usePixelStore()
  const dashboardStore = useDashboardStore()
  const dataStore = useDashboardDataStore()
  const tenantStore = useTenantStore()
  const liveStore = useLiveEventsStore()

  const {
    summaryData, seriesData, trafficQuality, sourceBreakdown, panelAudit,
    previousPeriod, heatmap, recentPurchases, geoData, totalRevenue,
    metaInsights, googleInsights, metaCreatives, googleKeywords,
  } = storeToRefs(dataStore)

  const loading = ref(false)
  const isRefreshing = ref(false)
  const adsLoading = ref(false)
  const initialized = ref(false)

  let coreSeq = 0
  let adsSeq = 0
  let inFlightCoreKey = ''
  let coreAbort: AbortController | null = null
  let adsAbort: AbortController | null = null
  let liveTimer: ReturnType<typeof setTimeout> | null = null
  let offCrmUpdated: (() => void) | null = null

  const ensurePixelSelected = () => {
    if (pixelStore.pixels.length && (!selectedPixel.value || !pixelStore.pixels.some((p) => p.name === selectedPixel.value))) {
      dashboardStore.setSelectedPixel(pixelStore.pixels[0]?.name || '')
    }
  }

  const fetchCore = async (opts: { force?: boolean; silent?: boolean } = {}) => {
    const px = selectedPixel.value
    const { from, to } = dashboardStore.getDateRange()
    const ts = dashboardStore.trafficSource
    const acc = tenantStore.activeAccountId || ''
    const nextCoreKey = `${acc}|${px}|${from}|${to}|${ts}`

    if (!opts.force && inFlightCoreKey === nextCoreKey) return
    const hasWarm = dataStore.coreKey === nextCoreKey && summaryData.value !== null
    if (!opts.force && hasWarm && Date.now() - dataStore.coreFetchedAt < 20_000) return

    coreAbort?.abort()
    const ac = new AbortController()
    coreAbort = ac
    inFlightCoreKey = nextCoreKey
    const seq = ++coreSeq

    if (!hasWarm && !opts.silent) loading.value = true
    else isRefreshing.value = true

    try {
      const [summary, series, purchases, geo, geoPurchases, revenue] = await Promise.all([
        dashboard.summary({ from, to, pixel: px, traffic_source: ts }, ac.signal),
        dashboard.series({ from, to, granularity: dashboardStore.granularityFor, pixel: px, traffic_source: ts }, ac.signal),
        dashboard.recentPurchases({ from, to, limit: 50, pixel: px, traffic_source: ts }, ac.signal),
        dashboard.geoStats({ from, to, pixel: px, traffic_source: ts }, ac.signal),
        dashboard.geoStatsByEvent({ from, to, event: 'Purchase', pixel: px, traffic_source: ts }, ac.signal),
        dashboard.revenue({ from, to, pixel: px, traffic_source: ts }, ac.signal),
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
      geoData.value = [
        ...(geo || []).map((g) => ({ ...g, event_name: 'PageView' })),
        ...(geoPurchases || []),
      ]
      dataStore.coreKey = nextCoreKey
      dataStore.coreFetchedAt = Date.now()
    } catch (err: unknown) {
      if (seq !== coreSeq || isAbortError(err)) return
      if (!opts.silent) toast.error('Erro', errorMessage(err, 'Não foi possível carregar a dashboard'))
    } finally {
      if (seq === coreSeq) {
        inFlightCoreKey = ''
        loading.value = false
        isRefreshing.value = false
      }
    }
  }

  const fetchAds = async (opts: { force?: boolean } = {}) => {
    const px = selectedPixel.value
    if (!px) {
      dataStore.clearAds()
      adsLoading.value = false
      return
    }
    const { from, to } = dashboardStore.getDateRange()
    const ts = dashboardStore.trafficSource
    const nextAdsKey = `${tenantStore.activeAccountId || ''}|${px}|${from}|${to}`
    const sameWindow = !opts.force && dataStore.adsKey === nextAdsKey && Date.now() - dataStore.adsFetchedAt < ADS_CACHE_TTL_MS
    const wantMetaRank = ts !== 'google' && (!sameWindow || !dataStore.metaRankLoaded)
    const wantGoogleRank = ts !== 'meta' && (!sameWindow || !dataStore.googleRankLoaded)

    if (sameWindow && !wantMetaRank && !wantGoogleRank) return

    adsAbort?.abort()
    const ac = new AbortController()
    adsAbort = ac
    const seq = ++adsSeq
    if (!sameWindow) {
      dataStore.clearAds()
    }
    adsLoading.value = true

    try {
      const [mLive, mRank, gLive, gRank] = await Promise.all([
        sameWindow ? Promise.resolve(metaInsights.value) : dashboard.metaLiveInsights({ pixel_name: px, from, to }, ac.signal).catch(() => null),
        wantMetaRank ? dashboard.metaCreativeRanking({ pixel_name: px, from, to }, ac.signal).catch(() => null) : Promise.resolve(null),
        sameWindow ? Promise.resolve(googleInsights.value) : dashboard.googleLiveInsights({ pixel_name: px, from, to }, ac.signal).catch(() => null),
        wantGoogleRank ? dashboard.googleKeywordRanking({ pixel_name: px, from, to }, ac.signal).catch(() => null) : Promise.resolve(null),
      ])
      if (seq !== adsSeq || ac.signal.aborted) return

      if (!sameWindow) {
        metaInsights.value = mLive
        googleInsights.value = gLive
        dataStore.adsKey = nextAdsKey
        dataStore.adsFetchedAt = Date.now()
      }
      if (wantMetaRank) {
        metaCreatives.value = (Array.isArray(mRank) ? mRank : []).slice(0, 3)
        dataStore.metaRankLoaded = true
      }
      if (wantGoogleRank) {
        googleKeywords.value = (Array.isArray(gRank) ? gRank : []).slice(0, 5)
        dataStore.googleRankLoaded = true
      }
    } finally {
      if (seq === adsSeq) adsLoading.value = false
    }
  }

  const fetchAll = async (opts: { force?: boolean; silent?: boolean; skipAds?: boolean } = {}) => {
    ensurePixelSelected()
    await Promise.all([
      fetchCore({ force: opts.force, silent: opts.silent }),
      opts.skipAds ? Promise.resolve() : fetchAds({ force: opts.force }),
    ])
  }

  const scheduleLiveRefresh = () => {
    if (!initialized.value || !dashboardStore.periodIncludesToday || liveTimer !== null) return
    liveTimer = setTimeout(() => {
      liveTimer = null
      if (initialized.value && dashboardStore.periodIncludesToday) {
        void fetchAll({ force: true, silent: true, skipAds: true })
      }
    }, LIVE_DEBOUNCE_MS)
  }

  const reloadForTenant = async (forcePixels: boolean) => {
    initialized.value = false
    if (forcePixels) dataStore.clearAll()
    await pixelStore.fetch(forcePixels).catch(() => {})
    ensurePixelSelected()
    await nextTick()
    initialized.value = true
    await fetchAll({ force: forcePixels })
  }

  watch(
    [() => dashboardStore.period, () => dashboardStore.customFrom, () => dashboardStore.customTo, () => dashboardStore.trafficSource, selectedPixel],
    () => {
      if (!initialized.value) return
      if (dashboardStore.period === 'custom' && (!dashboardStore.customFrom || !dashboardStore.customTo)) return
      void fetchAll()
    },
  )

  watch(() => liveStore.lastEventSeq, () => {
    const latest = liveStore.events[0]
    if (latest && selectedPixel.value && latest.pixel_name && latest.pixel_name !== selectedPixel.value) return
    scheduleLiveRefresh()
  })

  watch(() => tenantStore.switchVersion, () => { void reloadForTenant(true) })

  onMounted(() => {
    if (!liveStore.connected && !liveStore.connecting) void liveStore.connect()
    offCrmUpdated = liveStore.onCustomEvent('crm_updated', () => scheduleLiveRefresh())
    void reloadForTenant(false)
  })

  onUnmounted(() => {
    coreAbort?.abort()
    adsAbort?.abort()
    if (liveTimer !== null) clearTimeout(liveTimer)
    offCrmUpdated?.()
  })

  return {
    loading, isRefreshing, adsLoading, summaryData, seriesData, trafficQuality, sourceBreakdown,
    panelAudit, previousPeriod, heatmap, metaInsights, googleInsights, metaCreatives,
    googleKeywords, recentPurchases, geoData, totalRevenue,
    refresh: () => fetchAll({ force: true }),
  }
}
