import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import type {
  ConversationAuditItem,
  ConversationAuditKPIs,
  ConversationAuditQuery,
  ConversationAuditResponse,
  ConversationOriginCategory,
  ConversationOriginCounts,
  CrmBoardResponse,
  CrmStage,
  CrmTag,
  PublicCrmBoardResponse,
  WhatsAppInstance,
} from '#shared/types'
import { computeSaoPauloRange, type PeriodValue } from '~/stores/useDashboardStore'
import { useLiveEventsStore } from '~/stores/useLiveEventsStore'
import { downloadConversationsCSV } from '~/utils/conversationFormatters'
import { errorMessage } from '~/utils/errorMessage'

export const useConversationsAudit = (publicToken?: string) => {
  const api = useAPI()
  const wa = useWhatsApp()
  const toast = useToast()
  const liveStore = useLiveEventsStore()

  const loading = ref(true)
  const exporting = ref(false)
  const loadError = ref<string | null>(null)

  const period = ref<PeriodValue>('7d')
  const customFrom = ref('')
  const customTo = ref('')
  const selectedPixel = ref('')
  const selectedInstanceId = ref('')
  const search = ref('')
  const slaStatus = ref('all')
  const stageId = ref('')
  const tagId = ref('')
  const origin = ref<ConversationOriginCategory>('all')
  const campaign = ref('')
  const utm = ref('')
  const sortBy = ref('last_message_time')
  const sortDir = ref<'asc' | 'desc'>('desc')
  const page = ref(1)
  const pageSize = 50

  const items = ref<ConversationAuditItem[]>([])
  const total = ref(0)
  const kpis = ref<ConversationAuditKPIs>({
    total_conversations: 0, waiting_reply_count: 0, unanswered_count: 0,
    avg_first_response_sec: null, won_count: 0, won_revenue: 0,
  })
  const originCounts = ref<ConversationOriginCounts>({
    all: 0, meta_ads: 0, google_ads: 0, tiktok_ads: 0, bio_link: 0, site: 0,
  })
  const campaigns = ref<string[]>([])
  const utms = ref<string[]>([])
  const stages = ref<CrmStage[]>([])
  const tags = ref<CrmTag[]>([])
  const instances = ref<WhatsAppInstance[]>([])

  const selectedItem = ref<ConversationAuditItem | null>(null)
  const drawerOpen = ref(false)

  const dateRange = computed(() => computeSaoPauloRange(period.value, customFrom.value, customTo.value))

  const hasActiveFilters = computed(() => Boolean(
    search.value.trim() || slaStatus.value !== 'all' || stageId.value || tagId.value
    || origin.value !== 'all' || campaign.value || utm.value || (!publicToken && (selectedPixel.value || selectedInstanceId.value)),
  ))

  const buildQuery = (overrideLimit?: number, overrideOffset?: number): ConversationAuditQuery => ({
    from: dateRange.value.from,
    to: dateRange.value.to,
    pixel: publicToken ? undefined : (selectedPixel.value || undefined),
    instance_id: publicToken ? undefined : (selectedInstanceId.value || undefined),
    search: search.value.trim() || undefined,
    sla_status: slaStatus.value !== 'all' ? slaStatus.value : undefined,
    stage_id: stageId.value || undefined,
    tag_id: tagId.value || undefined,
    origin: origin.value !== 'all' ? origin.value : undefined,
    campaign: campaign.value || undefined,
    utm: utm.value || undefined,
    sort_by: sortBy.value,
    sort_dir: sortDir.value,
    limit: overrideLimit ?? pageSize,
    offset: overrideOffset ?? (page.value - 1) * pageSize,
  })

  const fetchEndpoint = (query: ConversationAuditQuery, signal?: AbortSignal): Promise<ConversationAuditResponse> => {
    if (!publicToken) return wa.listAuditConversations(query, signal)
    const q = new URLSearchParams()
    for (const [k, v] of Object.entries(query)) {
      if (v !== undefined && v !== null && v !== '') q.set(k, String(v))
    }
    const qs = q.toString() ? `?${q.toString()}` : ''
    return api.publicGet<ConversationAuditResponse>(`/public/dashboard/${encodeURIComponent(publicToken)}/conversations${qs}`, { signal })
  }

  let reqSeq = 0
  let abortCtrl: AbortController | null = null

  const fetchConversations = async (silent = false) => {
    abortCtrl?.abort()
    const ac = new AbortController()
    abortCtrl = ac
    const seq = ++reqSeq
    if (!silent) loading.value = true
    try {
      const res = await fetchEndpoint(buildQuery(), ac.signal)
      if (seq !== reqSeq) return
      items.value = res.items || []
      total.value = res.total || 0
      kpis.value = res.kpis || kpis.value
      originCounts.value = res.origin_counts || originCounts.value
      campaigns.value = res.campaigns || []
      utms.value = res.utms || []
      loadError.value = null
    } catch (err: unknown) {
      if (seq !== reqSeq || (err as { name?: string })?.name === 'AbortError') return
      loadError.value = errorMessage(err, 'Não foi possível carregar as conversas.')
    } finally {
      if (seq === reqSeq) loading.value = false
    }
  }

  const loadMetaFilters = async () => {
    try {
      if (publicToken) {
        const res = await api.publicGet<PublicCrmBoardResponse>(`/public/dashboard/${encodeURIComponent(publicToken)}/crm/board`).catch(() => null)
        if (res) {
          stages.value = res.stages || []
          tags.value = res.tags || []
        }
        return
      }
      const [boardRes, tagsRes, instRes] = await Promise.all([
        api.get<CrmBoardResponse>('/admin/crm/board').catch(() => null),
        api.get<CrmTag[]>('/admin/crm/tags').catch(() => []),
        api.get<{ data: WhatsAppInstance[] }>('/admin/whatsapp/instances').catch(() => ({ data: [] })),
      ])
      stages.value = boardRes?.stages || []
      tags.value = tagsRes || []
      instances.value = instRes?.data || []
    } catch {}
  }

  const toggleSort = (col: string) => {
    if (sortBy.value === col) sortDir.value = sortDir.value === 'desc' ? 'asc' : 'desc'
    else { sortBy.value = col; sortDir.value = 'desc' }
    page.value = 1
    void fetchConversations(false)
  }

  const clearFilters = () => {
    search.value = ''; slaStatus.value = 'all'; stageId.value = ''; tagId.value = ''
    origin.value = 'all'; campaign.value = ''; utm.value = ''
    if (!publicToken) { selectedPixel.value = ''; selectedInstanceId.value = '' }
    page.value = 1
    void fetchConversations(false)
  }

  const exportCSV = async () => {
    exporting.value = true
    try {
      const res = await fetchEndpoint(buildQuery(5000, 0))
      downloadConversationsCSV(res.items || [])
      toast.success('Exportado', `${(res.items || []).length} conversas exportadas em CSV.`)
    } catch (err: unknown) {
      toast.error('Erro', errorMessage(err, 'Não foi possível exportar o CSV.'))
    } finally {
      exporting.value = false
    }
  }

  const inspectConversation = (it: ConversationAuditItem) => {
    selectedItem.value = it
    drawerOpen.value = true
  }

  let searchTimer: ReturnType<typeof setTimeout> | null = null
  watch(search, () => {
    if (searchTimer) clearTimeout(searchTimer)
    searchTimer = setTimeout(() => { page.value = 1; void fetchConversations(false) }, 280)
  })

  watch([period, customFrom, customTo, selectedPixel, selectedInstanceId, slaStatus, stageId, tagId, origin, campaign, utm], () => {
    page.value = 1
    void fetchConversations(false)
  })

  watch(page, () => { void fetchConversations(false) })

  let liveTimer: ReturnType<typeof setTimeout> | null = null
  let offMsg: (() => void) | null = null
  let offCrm: (() => void) | null = null
  let pollTimer: ReturnType<typeof setInterval> | null = null

  const scheduleSilentRefresh = () => {
    if (liveTimer) clearTimeout(liveTimer)
    liveTimer = setTimeout(() => { void fetchConversations(true) }, 600)
  }

  onMounted(() => {
    void loadMetaFilters()
    void fetchConversations(false)
    if (publicToken) {
      pollTimer = setInterval(() => { void fetchConversations(true) }, 30_000)
    } else {
      offMsg = liveStore.onCustomEvent('whatsapp_message', scheduleSilentRefresh)
      offCrm = liveStore.onCustomEvent('crm_updated', scheduleSilentRefresh)
    }
  })

  onUnmounted(() => {
    abortCtrl?.abort()
    offMsg?.(); offCrm?.()
    if (searchTimer) clearTimeout(searchTimer)
    if (liveTimer) clearTimeout(liveTimer)
    if (pollTimer) clearInterval(pollTimer)
  })

  return {
    loading, exporting, loadError,
    period, customFrom, customTo, selectedPixel, selectedInstanceId,
    search, slaStatus, stageId, tagId, origin, campaign, utm,
    sortBy, sortDir, page, pageSize, hasActiveFilters,
    items, total, kpis, originCounts, campaigns, utms, stages, tags, instances,
    selectedItem, drawerOpen,
    fetchConversations, loadMetaFilters, toggleSort, clearFilters, exportCSV, inspectConversation,
  }
}
