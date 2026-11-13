import { ref, reactive, computed, type Ref } from 'vue'
import type {
  CrmCreateLeadForm,
  CrmLead,
  CrmLeadEditState,
  CrmStage,
  CrmStageForm,
  CrmTag,
  GoogleAdsAccount,
  GoogleAdsRemoteConversionAction,
  TrackedUser,
} from '#shared/types'
import { useAPI } from '~/composables/useAPI'
import { useGoogleAdsAccounts } from '~/composables/useGoogleAdsAccounts'
import { getDefaultStageColor } from '~/utils/colorPalette'
import { errorMessage } from '~/utils/errorMessage'

interface UseCrmModalsDeps {
  stages: Ref<CrmStage[]>
  allTags: Ref<CrmTag[]>
  gadsAccounts: Ref<GoogleAdsAccount[]>
  selectedPixel: Ref<string>
  fetchBoard: (silent?: boolean) => Promise<void>
  loadAllTags: () => Promise<void>
}

export const useCrmModals = (deps: UseCrmModalsDeps) => {
  const api = useAPI()
  const toast = useToast()
  const gadsApi = useGoogleAdsAccounts()

  // Modal Pipeline & Etapas
  const showPipelineModal = ref(false)
  const showStageModal = ref(false)
  const isCreatingStage = ref(false)
  const isSavingStage = ref(false)
  const isLoadingGadsActions = ref(false)
  const remoteGadsActions = ref<GoogleAdsRemoteConversionAction[]>([])
  let reopenPipelineAfterStage = false

  const selectedStage = ref<CrmStageForm>({
    name: '', color: getDefaultStageColor(), trigger_event: '', trigger_event_ctwa: '',
    google_ads_account_id: null, google_ads_action_id: '',
  })

  const loadRemoteActionsForAccount = async (accountId: number | null) => {
    remoteGadsActions.value = []
    if (!accountId) return
    isLoadingGadsActions.value = true
    try {
      const res = await gadsApi.listConversionActions(accountId)
      remoteGadsActions.value = res.conversion_actions || []
    } catch {
      remoteGadsActions.value = []
    } finally {
      isLoadingGadsActions.value = false
    }
  }

  const gadsActions = computed(() => {
    const accId = selectedStage.value.google_ads_account_id
    if (!accId) return []
    const acc = deps.gadsAccounts.value.find((a) => a.id === accId)
    const merged = new Map<string, { conversion_action_id: string; name: string; event_type: string }>()
    for (const local of acc?.conversion_actions || []) {
      merged.set(local.conversion_action_id, {
        conversion_action_id: local.conversion_action_id,
        name: local.name || local.conversion_action_id,
        event_type: local.event_type || 'Manual',
      })
    }
    for (const rem of remoteGadsActions.value) {
      if (!merged.has(rem.id)) {
        merged.set(rem.id, {
          conversion_action_id: rem.id,
          name: rem.name || rem.id,
          event_type: rem.category || rem.type || 'OAuth',
        })
      }
    }
    return Array.from(merged.values())
  })

  const onStageGoogleAdsAccountChange = (nextAccountId: number | null) => {
    if (selectedStage.value.google_ads_account_id !== nextAccountId) {
      selectedStage.value.google_ads_account_id = nextAccountId
      selectedStage.value.google_ads_action_id = ''
      void loadRemoteActionsForAccount(nextAccountId)
    }
  }

  // Modal de Criação Manual de Lead
  const showCreateLeadModal = ref(false)
  const isCreatingManualLead = ref(false)
  const newLeadForm = reactive<CrmCreateLeadForm>({
    first_name: '',
    last_name: '',
    email: '',
    phone: '',
    value: 0,
    stage_id: null,
  })

  // Modal de Detalhes do Lead
  const showLeadModal = ref(false)
  const selectedLeadId = ref<number | null>(null)
  const selectedLeadTrackedUser = ref<TrackedUser | null>(null)
  const showTrackedUserModal = ref(false)
  const selectedTrackedUser = ref<TrackedUser | null>(null)
  let originalLeadStageId: number | null = null

  const leadEdit = reactive<CrmLeadEditState>({
    first_name: '',
    last_name: '',
    email: '',
    phone: '',
    value: 0,
    stage_id: null,
    notes: [],
    tags: [],
  })

  const showCreateTagModal = ref(false)
  const isCreatingTag = ref(false)
  const newTag = reactive({ name: '', color: getDefaultStageColor() })

  const selectedTagToAdd = ref('')
  const newNoteContent = ref('')
  const isSavingLead = ref(false)
  const isAddingNote = ref(false)

  // Modais de Exclusão
  const showDeleteLeadModal = ref(false)
  const showDeleteNoteModal = ref(false)
  const showDeleteStageModal = ref(false)
  const noteToDeleteId = ref<number | null>(null)

  const openPipelineSettings = () => {
    showPipelineModal.value = true
  }

  const openNewStageModal = (fromPipeline = false) => {
    reopenPipelineAfterStage = fromPipeline || showPipelineModal.value
    showPipelineModal.value = false
    isCreatingStage.value = true
    selectedStage.value = {
      name: '', color: getDefaultStageColor(), trigger_event: '', trigger_event_ctwa: '',
      google_ads_account_id: null, google_ads_action_id: '',
    }
    remoteGadsActions.value = []
    showStageModal.value = true
  }

  const openStageConfig = (stage: CrmStage, fromPipeline = false) => {
    reopenPipelineAfterStage = fromPipeline || showPipelineModal.value
    showPipelineModal.value = false
    isCreatingStage.value = false
    selectedStage.value = {
      id: stage.id,
      name: stage.name,
      color: stage.color || getDefaultStageColor(),
      trigger_event: stage.trigger_event || '',
      trigger_event_ctwa: stage.trigger_event_ctwa || '',
      google_ads_account_id: stage.google_ads_account_id ?? null,
      google_ads_action_id: stage.google_ads_action_id || '',
    }
    void loadRemoteActionsForAccount(selectedStage.value.google_ads_account_id)
    showStageModal.value = true
  }

  const closeStageModal = () => {
    showStageModal.value = false
    if (reopenPipelineAfterStage) {
      reopenPipelineAfterStage = false
      showPipelineModal.value = true
    }
  }

  const saveStage = async () => {
    if (!selectedStage.value.name?.trim()) {
      toast.error('Atenção', 'O nome da etapa é obrigatório.')
      return
    }
    if (!selectedStage.value.google_ads_account_id) {
      selectedStage.value.google_ads_action_id = ''
    }
    isSavingStage.value = true
    try {
      if (isCreatingStage.value) {
        await api.post('/admin/crm/stages', selectedStage.value)
      } else {
        await api.put(`/admin/crm/stages/${selectedStage.value.id}`, selectedStage.value)
      }
      await deps.fetchBoard(true)
      toast.success('Sucesso', 'Etapa salva com sucesso.')
      closeStageModal()
    } catch (err: unknown) {
      toast.error('Erro', errorMessage(err, 'Falha ao salvar etapa.'))
    } finally {
      isSavingStage.value = false
    }
  }

  const askDeleteStage = () => {
    if (deps.stages.value.length <= 1) {
      toast.error('Atenção', 'Não é possível excluir a última etapa do funil.')
      return
    }
    showDeleteStageModal.value = true
  }

  const deleteStage = async () => {
    isSavingStage.value = true
    try {
      await api.delete(`/admin/crm/stages/${selectedStage.value.id}`)
      showDeleteStageModal.value = false
      await deps.fetchBoard(true)
      toast.success('Sucesso', 'Etapa excluída. Os leads foram movidos para a primeira coluna.')
      closeStageModal()
    } catch (err: unknown) {
      toast.error('Erro', errorMessage(err, 'Falha ao excluir etapa.'))
    } finally {
      isSavingStage.value = false
    }
  }

  return {
    showPipelineModal, showStageModal, isCreatingStage, isSavingStage, isLoadingGadsActions,
    selectedStage, gadsActions, onStageGoogleAdsAccountChange,
    openPipelineSettings, openNewStageModal, openStageConfig, closeStageModal, saveStage, askDeleteStage, deleteStage,
    showCreateLeadModal, isCreatingManualLead, newLeadForm,
    showLeadModal, selectedLeadId, selectedLeadTrackedUser, showTrackedUserModal, selectedTrackedUser,
    leadEdit, showCreateTagModal, isCreatingTag, newTag, selectedTagToAdd, newNoteContent,
    isSavingLead, isAddingNote, showDeleteLeadModal, showDeleteNoteModal, showDeleteStageModal, noteToDeleteId,
    getOriginalStageId: () => originalLeadStageId,
    setOriginalStageId: (id: number | null) => { originalLeadStageId = id },
  }
}
