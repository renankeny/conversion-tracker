import type { Ref } from 'vue'
import type { CrmLead, CrmStage, CrmTag, TrackedUser } from '#shared/types'
import { useAPI } from '~/composables/useAPI'
import { useTrackedUsers } from '~/composables/useTrackedUsers'

export const useCrmLeadActions = (
  modals: ReturnType<typeof import('~/composables/useCrmModals').useCrmModals>,
  stages: Ref<CrmStage[]>,
  allTags: Ref<CrmTag[]>,
  selectedPixel: Ref<string>,
  fetchBoard: (silent?: boolean) => Promise<void>,
  loadAllTags: () => Promise<void>,
) => {
  const api = useAPI()
  const toast = useToast()
  const trackedUsersApi = useTrackedUsers()

  const openCreateLeadModal = (defaultStageId?: number) => {
    modals.newLeadForm.first_name = ''
    modals.newLeadForm.last_name = ''
    modals.newLeadForm.email = ''
    modals.newLeadForm.phone = ''
    modals.newLeadForm.value = 0
    modals.newLeadForm.stage_id = defaultStageId || stages.value[0]?.id || null
    modals.showCreateLeadModal.value = true
  }

  const createManualLead = async () => {
    const f = modals.newLeadForm
    if (!f.phone.trim() && !f.email.trim() && !f.first_name.trim()) {
      toast.error('Atenção', 'Informe ao menos o nome, telefone ou e-mail.')
      return
    }
    if (!f.stage_id) {
      toast.error('Atenção', 'Selecione uma etapa inicial.')
      return
    }
    const targetStage = stages.value.find((s) => Number(s.id) === Number(f.stage_id))
    if (targetStage?.trigger_event === 'Purchase' && (!Number(f.value) || Number(f.value) <= 0)) {
      toast.error('Valor obrigatório', 'Informe um valor maior que R$ 0,00 para cadastrar na etapa de Venda (Purchase).')
      return
    }

    modals.isCreatingManualLead.value = true
    try {
      const fallbackExternalId = (!f.phone.trim() && !f.email.trim())
        ? `crm_${Date.now()}_${Math.random().toString(36).slice(2, 7)}`
        : undefined
      await api.post('/admin/crm/leads', {
        first_name: f.first_name.trim(),
        last_name: f.last_name.trim(),
        email: f.email.trim(),
        phone: f.phone.trim(),
        external_id: fallbackExternalId,
        value: Number(f.value) || 0,
        stage_id: f.stage_id,
        pixel_name: selectedPixel.value || '',
      })
      toast.success('Sucesso', 'Lead adicionado ao funil.')
      modals.showCreateLeadModal.value = false
      await fetchBoard(true)
    } catch (err: any) {
      toast.error('Erro', err?.message || 'Falha ao cadastrar lead.')
    } finally {
      modals.isCreatingManualLead.value = false
    }
  }

  const openLeadDetails = async (lead: CrmLead) => {
    modals.selectedLeadId.value = lead.id
    modals.selectedLeadTrackedUser.value = lead.tracked_user ? { ...lead.tracked_user } : null
    modals.setOriginalStageId(lead.stage_id ?? stages.value[0]?.id ?? null)
    modals.leadEdit.first_name = lead.tracked_user?.first_name || ''
    modals.leadEdit.last_name = lead.tracked_user?.last_name || ''
    modals.leadEdit.email = lead.tracked_user?.email || ''
    modals.leadEdit.phone = lead.tracked_user?.phone || ''
    modals.leadEdit.value = lead.value || 0
    modals.leadEdit.stage_id = lead.stage_id ?? stages.value[0]?.id ?? null
    modals.leadEdit.notes = lead.notes || []
    modals.leadEdit.tags = lead.tags ? [...lead.tags] : []
    modals.selectedTagToAdd.value = ''
    modals.showLeadModal.value = true
    void loadAllTags()

    try {
      const data = await api.get<CrmLead>(`/admin/crm/leads/${lead.id}`)
      modals.selectedLeadTrackedUser.value = data.tracked_user
        ? { ...(lead.tracked_user || {}), ...data.tracked_user }
        : modals.selectedLeadTrackedUser.value
      modals.leadEdit.first_name = data.tracked_user?.first_name || ''
      modals.leadEdit.last_name = data.tracked_user?.last_name || ''
      modals.leadEdit.email = data.tracked_user?.email || ''
      modals.leadEdit.phone = data.tracked_user?.phone || ''
      modals.leadEdit.value = data.value || 0
      const resolvedStageId = data.stage_id ?? stages.value[0]?.id ?? null
      modals.leadEdit.stage_id = resolvedStageId
      modals.setOriginalStageId(resolvedStageId)
      modals.leadEdit.notes = data.notes || []
      modals.leadEdit.tags = data.tags || []
    } catch (err: any) {
      toast.error('Erro', err?.message || 'Não foi possível carregar os detalhes completos do lead.')
    }
  }

  const openTrackedUserJourney = async () => {
    const tu = modals.selectedLeadTrackedUser.value
    if (!tu) return
    try {
      let user: TrackedUser | null = null
      if (tu.id) {
        user = await trackedUsersApi.find(tu.id)
      } else if (tu.external_id) {
        user = await trackedUsersApi.lookup({ external_id: tu.external_id })
      }
      if (user) {
        modals.selectedTrackedUser.value = user
        modals.showTrackedUserModal.value = true
      } else {
        toast.error('Atenção', 'Dados de rastreamento do visitante não encontrados.')
      }
    } catch (err: any) {
      toast.error('Erro', err?.message || 'Falha ao abrir jornada do visitante.')
    }
  }

  const saveLeadInfo = async () => {
    if (!modals.selectedLeadId.value) return
    const nextValue = Number(modals.leadEdit.value) || 0
    const originalStageId = modals.getOriginalStageId()
    const isMovingStage = !!(modals.leadEdit.stage_id && modals.leadEdit.stage_id !== originalStageId)

    if (isMovingStage) {
      const targetStage = stages.value.find((s) => Number(s.id) === Number(modals.leadEdit.stage_id))
      if (targetStage?.trigger_event === 'Purchase' && nextValue <= 0) {
        toast.error('Valor obrigatório', 'Defina um valor maior que R$ 0,00 antes de mover o lead para a etapa de Venda (Purchase).')
        return
      }
    }

    modals.isSavingLead.value = true
    try {
      await api.put(`/admin/crm/leads/${modals.selectedLeadId.value}`, {
        first_name: modals.leadEdit.first_name,
        last_name: modals.leadEdit.last_name,
        email: modals.leadEdit.email,
        phone: modals.leadEdit.phone,
        value: nextValue,
      })

      if (isMovingStage && modals.leadEdit.stage_id) {
        await api.put(`/admin/crm/leads/${modals.selectedLeadId.value}/move`, {
          stage_id: modals.leadEdit.stage_id,
          value: nextValue,
        })
        modals.setOriginalStageId(modals.leadEdit.stage_id)
      }

      await fetchBoard(true)
      modals.showLeadModal.value = false
      toast.success('Sucesso', 'Lead atualizado.')
    } catch (err: any) {
      toast.error('Erro', err?.message || 'Falha ao atualizar lead.')
    } finally {
      modals.isSavingLead.value = false
    }
  }

  const addNote = async () => {
    if (!modals.newNoteContent.value.trim() || !modals.selectedLeadId.value) return
    modals.isAddingNote.value = true
    try {
      const note = await api.post<any>(`/admin/crm/leads/${modals.selectedLeadId.value}/notes`, {
        content: modals.newNoteContent.value.trim(),
      })
      modals.leadEdit.notes.unshift(note)
      modals.newNoteContent.value = ''
      await fetchBoard(true)
    } catch (err: any) {
      toast.error('Erro', err?.message || 'Não foi possível salvar a anotação.')
    } finally {
      modals.isAddingNote.value = false
    }
  }

  const confirmDeleteNote = (noteId: number) => {
    modals.noteToDeleteId.value = noteId
    modals.showDeleteNoteModal.value = true
  }

  const doDeleteNote = async () => {
    if (!modals.noteToDeleteId.value) return
    try {
      await api.delete(`/admin/crm/notes/${modals.noteToDeleteId.value}`)
      modals.leadEdit.notes = modals.leadEdit.notes.filter((n) => n.id !== modals.noteToDeleteId.value)
      modals.showDeleteNoteModal.value = false
      await fetchBoard(true)
    } catch (err: any) {
      toast.error('Erro', err?.message || 'Falha ao excluir anotação.')
    }
  }

  const openConfirmDeleteLead = () => {
    modals.showLeadModal.value = false
    modals.showDeleteLeadModal.value = true
  }

  const cancelDeleteLead = () => {
    modals.showDeleteLeadModal.value = false
    if (modals.selectedLeadId.value) modals.showLeadModal.value = true
  }

  const doDeleteLead = async () => {
    if (!modals.selectedLeadId.value) return
    try {
      await api.delete(`/admin/crm/leads/${modals.selectedLeadId.value}`)
      modals.selectedLeadId.value = null
      modals.showDeleteLeadModal.value = false
      modals.showLeadModal.value = false
      toast.success('Sucesso', 'Lead excluído.')
      await fetchBoard(true)
    } catch (err: any) {
      toast.error('Erro', err?.message || 'Falha ao excluir lead.')
    }
  }

  return {
    openCreateLeadModal,
    createManualLead,
    openLeadDetails,
    openTrackedUserJourney,
    saveLeadInfo,
    addNote,
    confirmDeleteNote,
    doDeleteNote,
    openConfirmDeleteLead,
    cancelDeleteLead,
    doDeleteLead,
  }
}
