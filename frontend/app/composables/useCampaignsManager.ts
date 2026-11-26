import type {
  CampaignAttributionMode,
  CampaignPlatform,
  CampaignRow,
  CampaignsKpiSummary,
  CampaignsOverviewResponse,
  GoogleAdsAccount,
  MetaAccount,
} from '#shared/types'
import type { AdAccountSelectOption } from '~/components/BaseAdAccountSelect.vue'
import type { HierarchyTabKey } from '~/components/campaigns/CampaignsHierarchyTabs.vue'
import { formatSPDateISO, resolvePeriodDates } from '~/utils/campaignFormatters'
import { extractCampaignErrMsg, useCampaignMutations } from './useCampaignMutations'

const emptySummary = (): CampaignsKpiSummary => ({
  spend: 0, impressions: 0, clicks: 0, leads: 0, conversations: 0, ai_qualified: 0,
  gross_sales: 0, refund_count: 0, chargeback_count: 0, net_sales: 0,
  gross_revenue: 0, refund_amount: 0, net_revenue: 0, operational_costs: 0,
  net_profit: 0, refund_rate: 0, real_cpa: 0, gross_roas: 0, net_roas: 0, roi: 0,
})

export function useCampaignsManager() {
  const api = useAPI()
  const pixelStore = usePixelStore()

  const platform = ref<CampaignPlatform>('all')
  const pixelName = ref('')
  const selectedAdAccount = ref('')
  const periodPreset = ref('7d')
  const customSince = ref(formatSPDateISO(new Date(Date.now() - 6 * 86400000)))
  const customUntil = ref(formatSPDateISO(new Date()))
  const attributionMode = ref<CampaignAttributionMode>('last_click')

  const activeTab = ref<HierarchyTabKey>('campaigns')
  const statusFilter = ref('all')
  const searchQuery = ref('')
  const selectedCampaignIds = ref<string[]>([])
  const selectedAdSetIds = ref<string[]>([])

  const metaAccounts = ref<MetaAccount[]>([])
  const googleAccounts = ref<GoogleAdsAccount[]>([])
  const loadingAccounts = ref(false)
  const overview = ref<CampaignsOverviewResponse | null>(null)
  const loading = ref(false)
  const error = ref('')

  const adAccountOptions = computed<AdAccountSelectOption[]>(() => {
    const opts: AdAccountSelectOption[] = [
      { value: '', label: 'Automático (conta do Pixel / ativa)', channel: 'none' },
    ]
    if (platform.value !== 'google') {
      for (const acc of metaAccounts.value) {
        if (!acc.enabled || !acc.ad_account_id) continue
        opts.push({
          value: `meta:${acc.id}`,
          label: acc.ad_account_name || acc.name,
          sublabel: acc.ad_account_id,
          avatarUrl: acc.ad_account_picture_url || acc.oauth_picture_url || acc.picture_url,
          channel: 'meta',
          badge: 'Meta',
        })
      }
    }
    if (platform.value !== 'meta') {
      for (const acc of googleAccounts.value) {
        if (!acc.enabled || !acc.customer_id) continue
        opts.push({
          value: `google:${acc.id}`,
          label: acc.customer_name || acc.name,
          sublabel: acc.customer_id,
          avatarUrl: acc.oauth_picture_url || acc.picture_url,
          channel: 'google_ads',
          badge: 'Google',
        })
      }
    }
    return opts
  })

  const loadAccounts = async () => {
    loadingAccounts.value = true
    try {
      const [mRes, gRes] = await Promise.allSettled([
        api.get<{ data?: MetaAccount[] }>('/admin/meta-accounts'),
        api.get<{ data?: GoogleAdsAccount[] }>('/admin/google-ads-accounts'),
      ])
      if (mRes.status === 'fulfilled') metaAccounts.value = mRes.value.data || []
      if (gRes.status === 'fulfilled') googleAccounts.value = gRes.value.data || []
    } finally {
      loadingAccounts.value = false
    }
  }

  const fetchOverview = async (forceRefresh = false) => {
    loading.value = true
    error.value = ''
    try {
      const { since, until } = resolvePeriodDates(periodPreset.value, customSince.value, customUntil.value)
      const query: Record<string, string> = {
        platform: platform.value,
        attribution: attributionMode.value,
        since,
        until,
      }
      if (pixelName.value) query.pixel = pixelName.value
      if (selectedAdAccount.value.startsWith('meta:')) {
        query.meta_account_id = selectedAdAccount.value.slice(5)
      } else if (selectedAdAccount.value.startsWith('google:')) {
        query.google_ads_account_id = selectedAdAccount.value.slice(7)
      }
      if (forceRefresh) query.refresh = '1'
      overview.value = await api.get<CampaignsOverviewResponse>('/admin/campaigns/overview', { query })
    } catch (err) {
      error.value = extractCampaignErrMsg(err, 'Não foi possível carregar as campanhas.')
    } finally {
      loading.value = false
    }
  }

  const applyRowFilters = (rows: CampaignRow[]): CampaignRow[] => {
    const q = searchQuery.value.trim().toLowerCase()
    return rows.filter((r) => {
      if (statusFilter.value !== 'all' && r.platform !== 'unattributed' && r.status !== statusFilter.value) {
        return false
      }
      if (!q) return true
      return (
        r.name.toLowerCase().includes(q) ||
        r.id.toLowerCase().includes(q) ||
        (r.campaign_name || '').toLowerCase().includes(q) ||
        (r.adset_name || '').toLowerCase().includes(q)
      )
    })
  }

  const visibleCampaigns = computed(() => applyRowFilters(overview.value?.campaigns || []))
  const visibleAdSets = computed(() => {
    let rows = overview.value?.adsets || []
    if (selectedCampaignIds.value.length) {
      const set = new Set(selectedCampaignIds.value)
      rows = rows.filter((r) => r.campaign_id && set.has(r.campaign_id))
    }
    return applyRowFilters(rows)
  })
  const visibleAds = computed(() => {
    let rows = overview.value?.ads || []
    if (selectedCampaignIds.value.length) {
      const cSet = new Set(selectedCampaignIds.value)
      rows = rows.filter((r) => r.campaign_id && cSet.has(r.campaign_id))
    }
    if (selectedAdSetIds.value.length) {
      const aSet = new Set(selectedAdSetIds.value)
      rows = rows.filter((r) => r.adset_id && aSet.has(r.adset_id))
    }
    return applyRowFilters(rows)
  })

  const activeRows = computed(() => {
    if (activeTab.value === 'adsets') return visibleAdSets.value
    if (activeTab.value === 'ads') return visibleAds.value
    return visibleCampaigns.value
  })
  const summary = computed(() => overview.value?.summary || emptySummary())
  const currency = computed(() => overview.value?.currency || 'BRL')

  const toggleSelectRow = (id: string) => {
    const target = activeTab.value === 'campaigns' ? selectedCampaignIds : selectedAdSetIds
    target.value = target.value.includes(id)
      ? target.value.filter((x) => x !== id)
      : [...target.value, id]
  }

  const drillDownRow = (row: CampaignRow) => {
    if (row.platform === 'unattributed') return
    if (row.level === 'campaign') {
      selectedCampaignIds.value = [row.id]
      selectedAdSetIds.value = []
      activeTab.value = 'adsets'
    } else if (row.level === 'adset') {
      if (row.campaign_id) selectedCampaignIds.value = [row.campaign_id]
      selectedAdSetIds.value = [row.id]
      activeTab.value = 'ads'
    }
  }

  const mutations = useCampaignMutations(pixelName, currency, fetchOverview)

  watch([platform, pixelName, selectedAdAccount, periodPreset, attributionMode], () => {
    selectedCampaignIds.value = []
    selectedAdSetIds.value = []
    fetchOverview(false)
  })
  watch([customSince, customUntil], () => {
    if (periodPreset.value === 'custom') fetchOverview(false)
  })

  return {
    pixelStore, platform, pixelName, selectedAdAccount, periodPreset, customSince, customUntil,
    attributionMode, activeTab, statusFilter, searchQuery, selectedCampaignIds, selectedAdSetIds,
    adAccountOptions, loadingAccounts, overview, loading, error,
    visibleCampaigns, visibleAdSets, visibleAds, activeRows, summary, currency,
    loadAccounts, fetchOverview, toggleSelectRow, drillDownRow, ...mutations,
  }
}
