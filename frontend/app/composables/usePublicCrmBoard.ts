import { ref, computed, onMounted, onUnmounted } from 'vue'
import type { CrmLead, CrmStage, CrmTag, PublicCrmBoardResponse } from '#shared/types'
import { useCrmHelpers } from '~/composables/useCrmHelpers'
import { resolveStageEffectiveEvent } from '~/utils/crmEffectiveEvent'
import { errorMessage } from '~/utils/errorMessage'

interface PendingPortalLeadMove {
  lead: CrmLead
  from: string
  to: string
  fromIndex: number
  toIndex: number
}

export const usePublicCrmBoard = (token: string) => {
  const api = useAPI()
  const toast = useToast()
  const { getOriginCategory, isWonStage, isLostStage } = useCrmHelpers()

  const isLoading = ref(true)
  const loadError = ref<string | null>(null)
  const allowEdit = ref(false)
  const stages = ref<CrmStage[]>([])
  const leadsByStage = ref<Record<string, CrmLead[]>>({})
  const collapsedStages = ref<Record<string, boolean>>({})
  const allTags = ref<CrmTag[]>([])

  const searchQuery = ref('')
  const selectedTagFilter = ref('')
  const selectedOriginFilter = ref('')

  const isFilterActive = computed(() => Boolean(searchQuery.value || selectedTagFilter.value || selectedOriginFilter.value))
  const clearFilters = () => {
    searchQuery.value = ''
    selectedTagFilter.value = ''
    selectedOriginFilter.value = ''
  }

  const matchesFilter = (lead: CrmLead): boolean => {
    if (searchQuery.value) {
      const q = searchQuery.value.toLowerCase().trim()
      const name = `${lead.tracked_user?.first_name || ''} ${lead.tracked_user?.last_name || ''}`.toLowerCase()
      const email = (lead.tracked_user?.email || '').toLowerCase()
      const phone = (lead.tracked_user?.phone || '').replace(/\D/g, '')
      const qPhone = q.replace(/\D/g, '')
      if (!name.includes(q) && !email.includes(q) && !(qPhone && phone.includes(qPhone))) return false
    }
    if (selectedTagFilter.value) {
      const tagId = Number(selectedTagFilter.value)
      if (!(lead.tags || []).some((t) => t.id === tagId)) return false
    }
    if (selectedOriginFilter.value && getOriginCategory(lead.tracked_user) !== selectedOriginFilter.value) {
      return false
    }
    return true
  }

  const stageById = computed(() => new Map<number, CrmStage>(stages.value.map((s) => [Number(s.id), s])))
  const allBoardLeads = computed(() => Object.values(leadsByStage.value).flat())
  const visibleBoardLeads = computed(() => allBoardLeads.value.filter(matchesFilter))

  const totalPipelineValue = computed(() => visibleBoardLeads.value.reduce((acc, l) => {
    const st = l.stage_id ? stageById.value.get(Number(l.stage_id)) : undefined
    if (isWonStage(st) || isLostStage(st)) return acc
    return acc + (Number(l.value) || 0)
  }, 0))

  const totalLeadsCount = computed(() => allBoardLeads.value.length)
  const filteredTotalCount = computed(() => visibleBoardLeads.value.length)

  const wonLeads = computed(() => visibleBoardLeads.value.filter((l) => {
    const st = l.stage_id ? stageById.value.get(Number(l.stage_id)) : undefined
    return isWonStage(st)
  }))
  const totalWonValue = computed(() => wonLeads.value.reduce((acc, l) => acc + (Number(l.value) || 0), 0))
  const averageTicket = computed(() => {
    const wonWithVal = wonLeads.value.filter((l) => Number(l.value) > 0)
    if (!wonWithVal.length) return 0
    return wonWithVal.reduce((acc, l) => acc + Number(l.value), 0) / wonWithVal.length
  })

  const toggleStageCollapse = (stageId: string | number) => {
    collapsedStages.value[stageId] = !collapsedStages.value[stageId]
  }

  let boardSeq = 0
  const baseUrl = `/public/dashboard/${encodeURIComponent(token)}/crm`

  const fetchBoard = async (silent = false) => {
    const seq = ++boardSeq
    if (!silent) isLoading.value = true
    try {
      const data = await api.publicGet<PublicCrmBoardResponse>(`${baseUrl}/board`)
      if (seq !== boardSeq) return
      allowEdit.value = Boolean(data.allow_edit)
      stages.value = data.stages || []
      allTags.value = data.tags || []

      const grouped: Record<string, CrmLead[]> = {}
      const fallbackId = stages.value[0]?.id ?? null
      for (const s of stages.value) grouped[String(s.id)] = []
      for (const l of data.leads || []) {
        if (!l.stage_id && fallbackId) l.stage_id = fallbackId
        const sid = String(l.stage_id || fallbackId || '')
        if (!sid) continue
        if (!grouped[sid]) grouped[sid] = []
        grouped[sid]!.push(l)
      }
      leadsByStage.value = grouped
      loadError.value = null
    } catch (err: unknown) {
      if (seq !== boardSeq) return
      loadError.value = errorMessage(err, 'Não foi possível carregar o funil de vendas.')
    } finally {
      if (seq === boardSeq && !silent) isLoading.value = false
    }
  }

  const showPurchaseModal = ref(false)
  const purchaseValue = ref<number | null>(null)
  const isConfirmingPurchase = ref(false)
  let pendingMove: PendingPortalLeadMove | null = null

  const executeMove = async (lead: CrmLead, stageId: string, value: number): Promise<boolean> => {
    const parsedStage = parseInt(stageId, 10)
    lead.stage_id = parsedStage
    try {
      await api.publicPut(`${baseUrl}/leads/${lead.id}/move`, { stage_id: parsedStage, value })
      return true
    } catch (err: unknown) {
      toast.error('Erro ao mover lead', errorMessage(err, 'Não foi possível mover o lead.'))
      await fetchBoard(true)
      return false
    }
  }

  const onDragEnd = (evt: { from?: { dataset?: { stageId?: string } }; to?: { dataset?: { stageId?: string } }; oldIndex?: number; newIndex?: number }) => {
    if (!allowEdit.value) return
    const fromStageId = evt.from?.dataset?.stageId
    const toStageId = evt.to?.dataset?.stageId
    if (!fromStageId || !toStageId || fromStageId === toStageId) return
    const newIndex = evt.newIndex ?? 0
    const movedLead = leadsByStage.value[toStageId]?.[newIndex]
    if (!movedLead) return
    const targetStage = stages.value.find((s) => String(s.id) === String(toStageId))
    if (resolveStageEffectiveEvent(targetStage, movedLead.tracked_user) === 'Purchase' && (!movedLead.value || movedLead.value <= 0)) {
      pendingMove = { lead: movedLead, from: String(fromStageId), to: String(toStageId), fromIndex: evt.oldIndex ?? 0, toIndex: newIndex }
      purchaseValue.value = null
      showPurchaseModal.value = true
      return
    }
    void executeMove(movedLead, String(toStageId), movedLead.value || 0)
  }

  const cancelPurchase = () => {
    showPurchaseModal.value = false
    pendingMove = null
    void fetchBoard(true)
  }

  const confirmPurchase = async () => {
    if (isConfirmingPurchase.value || !pendingMove) return
    if (!purchaseValue.value || purchaseValue.value <= 0) {
      toast.error('Valor obrigatório', 'Informe um valor maior que R$ 0,00 para confirmar a venda.')
      return
    }
    isConfirmingPurchase.value = true
    const { lead, to } = pendingMove
    try {
      const ok = await executeMove(lead, to, purchaseValue.value)
      if (ok) {
        lead.value = purchaseValue.value
        pendingMove = null
        showPurchaseModal.value = false
      }
    } finally {
      isConfirmingPurchase.value = false
    }
  }

  let pollTimer: ReturnType<typeof setInterval> | null = null
  onMounted(() => {
    void fetchBoard(false)
    pollTimer = setInterval(() => { void fetchBoard(true) }, 30_000)
  })
  onUnmounted(() => { if (pollTimer) clearInterval(pollTimer) })

  return {
    isLoading, loadError, allowEdit, stages, leadsByStage, collapsedStages, allTags,
    searchQuery, selectedTagFilter, selectedOriginFilter, isFilterActive, clearFilters, matchesFilter,
    totalPipelineValue, totalLeadsCount, totalWonValue, averageTicket, filteredTotalCount,
    toggleStageCollapse, fetchBoard,
    showPurchaseModal, purchaseValue, isConfirmingPurchase, onDragEnd, cancelPurchase, confirmPurchase,
  }
}
