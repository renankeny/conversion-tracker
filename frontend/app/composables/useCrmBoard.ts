import { ref, computed, watch, nextTick, onMounted, onUnmounted } from 'vue'
import type { CrmBoardResponse, CrmLead, CrmStage, CrmTag, GoogleAdsAccount } from '#shared/types'
import { useAPI } from '~/composables/useAPI'
import { useGoogleAdsAccounts } from '~/composables/useGoogleAdsAccounts'
import { useCrmHelpers } from '~/composables/useCrmHelpers'
import { useCrmDragDrop } from '~/composables/useCrmDragDrop'
import { useAuthStore } from '~/stores/useAuthStore'
import { useLiveEventsStore } from '~/stores/useLiveEventsStore'
import { errorMessage } from '~/utils/errorMessage'

export const useCrmBoard = () => {
  const api = useAPI()
  const toast = useToast()
  const authStore = useAuthStore()
  const liveStore = useLiveEventsStore()
  const { getOriginCategory, isWonStage, isLostStage } = useCrmHelpers()

  const isLoading = ref(true)
  const stages = ref<CrmStage[]>([])
  const leads = ref<CrmLead[]>([])
  const leadsByStage = ref<Record<string, CrmLead[]>>({})
  const selectedPixel = ref('')
  const collapsedStages = ref<Record<string, boolean>>({})
  const allTags = ref<CrmTag[]>([])
  const gadsAccounts = ref<GoogleAdsAccount[]>([])

  const canManageStages = computed(() => authStore.user?.role !== 'employee')
  const canDeleteLeads = computed(() => authStore.user?.role !== 'employee')

  // Filtros
  const searchQuery = ref('')
  const selectedTagFilter = ref('')
  const selectedOriginFilter = ref('')

  const isFilterActive = computed(() => {
    return !!searchQuery.value || !!selectedTagFilter.value || !!selectedOriginFilter.value
  })

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

      const matchesName = name.includes(q)
      const matchesEmail = email.includes(q)
      const matchesPhone = qPhone ? phone.includes(qPhone) : false
      if (!matchesName && !matchesEmail && !matchesPhone) return false
    }

    if (selectedTagFilter.value) {
      const tagId = Number(selectedTagFilter.value)
      const hasTag = (lead.tags || []).some((t) => t.id === tagId)
      if (!hasTag) return false
    }

    if (selectedOriginFilter.value) {
      const category = getOriginCategory(lead.tracked_user)
      if (category !== selectedOriginFilter.value) return false
    }

    return true
  }

  const stageById = computed(() => {
    const map = new Map<number, CrmStage>()
    for (const s of stages.value) map.set(Number(s.id), s)
    return map
  })

  const allBoardLeads = computed(() => Object.values(leadsByStage.value).flat())
  const visibleBoardLeads = computed(() => allBoardLeads.value.filter(matchesFilter))

  // Valor em Negociação: soma apenas etapas abertas (exclui ganhas e perdidas)
  const totalPipelineValue = computed(() => {
    return visibleBoardLeads.value.reduce((acc, l) => {
      const st = l.stage_id ? stageById.value.get(Number(l.stage_id)) : undefined
      if (isWonStage(st) || isLostStage(st)) return acc
      return acc + (Number(l.value) || 0)
    }, 0)
  })

  const totalLeadsCount = computed(() => allBoardLeads.value.length)
  const filteredTotalCount = computed(() => visibleBoardLeads.value.length)

  const wonLeads = computed(() => {
    return visibleBoardLeads.value.filter((l) => {
      const st = l.stage_id ? stageById.value.get(Number(l.stage_id)) : undefined
      return isWonStage(st)
    })
  })

  const totalWonValue = computed(() => {
    return wonLeads.value.reduce((acc, l) => acc + (Number(l.value) || 0), 0)
  })

  const averageTicket = computed(() => {
    const wonWithVal = wonLeads.value.filter((l) => Number(l.value) > 0)
    if (!wonWithVal.length) return 0
    const sum = wonWithVal.reduce((acc, l) => acc + Number(l.value), 0)
    return sum / wonWithVal.length
  })

  const toggleStageCollapse = (stageId: string | number) => {
    collapsedStages.value[stageId] = !collapsedStages.value[stageId]
  }

  const getStageTotalValue = (stageId: string | number) => {
    const stageLeads = leadsByStage.value[stageId] || []
    return stageLeads.reduce((acc, l) => acc + (Number(l.value) || 0), 0)
  }

  const getLeadsForStage = (stageId: string | number) => {
    return leadsByStage.value[stageId] || []
  }

  let boardSeq = 0
  let suppressPixelWatch = false

  const fetchBoard = async (silent = false) => {
    const seq = ++boardSeq
    if (!silent) isLoading.value = true
    try {
      const data = await api.get<CrmBoardResponse>(`/admin/crm/board?pixel=${encodeURIComponent(selectedPixel.value)}`)
      if (seq !== boardSeq) return
      stages.value = data.stages || []
      leads.value = data.leads || []

      const grouped: Record<string, CrmLead[]> = {}
      const fallbackId = stages.value[0]?.id ?? null
      stages.value.forEach((s) => {
        grouped[String(s.id)] = []
      })
      leads.value.forEach((l) => {
        if (!l.stage_id && fallbackId) {
          l.stage_id = fallbackId
        }
        const sid = String(l.stage_id || fallbackId || '')
        if (!sid) return
        if (!grouped[sid]) grouped[sid] = []
        grouped[sid]!.push(l)
      })
      leadsByStage.value = grouped
    } catch (err: unknown) {
      if (seq !== boardSeq) return
      if (!silent) {
        toast.error('Erro', errorMessage(err, 'Falha ao carregar o funil de vendas.'))
      }
    } finally {
      if (seq === boardSeq && !silent) isLoading.value = false
    }
  }

  const resetPixelAndFetchBoard = async () => {
    suppressPixelWatch = true
    selectedPixel.value = ''
    await nextTick()
    suppressPixelWatch = false
    await fetchBoard(false)
  }

  const loadGadsAccounts = async () => {
    try {
      gadsAccounts.value = await useGoogleAdsAccounts().list()
    } catch {
      // Silencioso para perfis sem permissão de contas Google Ads
    }
  }

  const loadAllTags = async () => {
    try {
      const data = await api.get<CrmTag[]>('/admin/crm/tags')
      allTags.value = data || []
    } catch {}
  }

  watch(selectedPixel, () => {
    if (suppressPixelWatch) return
    void fetchBoard(false)
  })

  let liveRefreshTimer: ReturnType<typeof setTimeout> | null = null
  let offCrmEvent: (() => void) | null = null
  const scheduleSilentRefresh = () => {
    if (liveRefreshTimer) clearTimeout(liveRefreshTimer)
    liveRefreshTimer = setTimeout(() => {
      void fetchBoard(true)
      void loadAllTags()
    }, 350)
  }

  onMounted(() => {
    offCrmEvent = liveStore.onCustomEvent('crm_updated', scheduleSilentRefresh)
  })

  onUnmounted(() => {
    offCrmEvent?.()
    if (liveRefreshTimer) clearTimeout(liveRefreshTimer)
  })

  const dragDrop = useCrmDragDrop({ stages, leadsByStage, fetchBoard, canManageStages })

  return {
    isLoading, stages, leads, leadsByStage, selectedPixel, collapsedStages, allTags, gadsAccounts,
    canManageStages, canDeleteLeads,
    searchQuery, selectedTagFilter, selectedOriginFilter, isFilterActive, clearFilters, matchesFilter,
    totalPipelineValue, totalLeadsCount, totalWonValue, averageTicket, filteredTotalCount,
    toggleStageCollapse, getStageTotalValue, getLeadsForStage,
    fetchBoard, resetPixelAndFetchBoard, loadGadsAccounts, loadAllTags,
    ...dragDrop,
  }
}
