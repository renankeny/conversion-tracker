import { ref, reactive, type Ref } from 'vue'
import type { CrmCreateLeadForm, CrmLead, CrmLeadEditState, CrmNote, CrmStage, CrmTag, TrackedUser } from '#shared/types'
import { resolveStageEffectiveEvent } from '~/utils/crmEffectiveEvent'
import { getDefaultStageColor } from '~/utils/colorPalette'
import { errorMessage } from '~/utils/errorMessage'

export const usePublicCrmActions = (
  token: string,
  stages: Ref<CrmStage[]>,
  allTags: Ref<CrmTag[]>,
  allowEdit: Ref<boolean>,
  fetchBoard: (silent?: boolean) => Promise<void>,
) => {
  const api = useAPI()
  const toast = useToast()
  const baseUrl = `/public/dashboard/${encodeURIComponent(token)}/crm`

  const showCreateLeadModal = ref(false)
  const isCreatingManualLead = ref(false)
  const newLeadForm = reactive<CrmCreateLeadForm>({
    first_name: '', last_name: '', email: '', phone: '', value: 0, stage_id: null,
  })

  const showLeadModal = ref(false)
  const selectedLeadId = ref<number | null>(null)
  const selectedLeadTrackedUser = ref<TrackedUser | null>(null)
  let originalStageId: number | null = null

  const leadEdit = reactive<CrmLeadEditState>({
    first_name: '', last_name: '', email: '', phone: '', value: 0, stage_id: null, notes: [], tags: [],
  })

  const selectedTagToAdd = ref('')
  const newNoteContent = ref('')
  const isSavingLead = ref(false)
  const isAddingNote = ref(false)

  const showDeleteNoteModal = ref(false)
  const noteToDeleteId = ref<number | null>(null)

  const showCreateTagModal = ref(false)
  const isCreatingTag = ref(false)
  const newTag = reactive({ name: '', color: getDefaultStageColor() })

  let detailSeq = 0

  const openCreateLeadModal = (defaultStageId?: number) => {
    if (!allowEdit.value) return
    newLeadForm.first_name = ''; newLeadForm.last_name = ''; newLeadForm.email = ''; newLeadForm.phone = ''
    newLeadForm.value = 0; newLeadForm.stage_id = defaultStageId || stages.value[0]?.id || null
    showCreateLeadModal.value = true
  }

  const createManualLead = async () => {
    if (!allowEdit.value) return
    if (!newLeadForm.phone.trim() && !newLeadForm.email.trim()) {
      toast.error('Atenção', 'Informe telefone ou e-mail do contato.')
      return
    }
    if (!newLeadForm.stage_id) {
      toast.error('Atenção', 'Selecione uma etapa inicial.')
      return
    }
    const targetStage = stages.value.find((s) => Number(s.id) === Number(newLeadForm.stage_id))
    if (resolveStageEffectiveEvent(targetStage, null) === 'Purchase' && (!Number(newLeadForm.value) || Number(newLeadForm.value) <= 0)) {
      toast.error('Valor obrigatório', 'Informe o valor da venda para esta etapa.')
      return
    }
    isCreatingManualLead.value = true
    try {
      await api.publicPost(`${baseUrl}/leads`, {
        first_name: newLeadForm.first_name.trim(),
        last_name: newLeadForm.last_name.trim(),
        email: newLeadForm.email.trim(),
        phone: newLeadForm.phone.trim(),
        value: Number(newLeadForm.value) || 0,
        stage_id: newLeadForm.stage_id,
      })
      showCreateLeadModal.value = false
      toast.success('Sucesso', 'Lead adicionado ao funil.')
      await fetchBoard(true)
    } catch (err: unknown) {
      toast.error('Erro', errorMessage(err, 'Não foi possível adicionar o lead.'))
    } finally {
      isCreatingManualLead.value = false
    }
  }

  const openLeadDetails = async (lead: CrmLead) => {
    const openedId = lead.id
    const seq = ++detailSeq
    selectedLeadId.value = openedId
    selectedLeadTrackedUser.value = lead.tracked_user ? { ...lead.tracked_user } : null
    originalStageId = lead.stage_id ?? stages.value[0]?.id ?? null
    leadEdit.first_name = lead.tracked_user?.first_name || ''
    leadEdit.last_name = lead.tracked_user?.last_name || ''
    leadEdit.email = lead.tracked_user?.email || ''
    leadEdit.phone = lead.tracked_user?.phone || ''
    leadEdit.value = lead.value || 0
    leadEdit.stage_id = originalStageId
    leadEdit.notes = lead.notes || []
    leadEdit.tags = lead.tags ? [...lead.tags] : []
    selectedTagToAdd.value = ''
    showLeadModal.value = true

    try {
      const data = await api.publicGet<CrmLead>(`${baseUrl}/leads/${openedId}`)
      if (seq !== detailSeq || selectedLeadId.value !== openedId) return
      selectedLeadTrackedUser.value = data.tracked_user ? { ...(lead.tracked_user || {}), ...data.tracked_user } : selectedLeadTrackedUser.value
      leadEdit.first_name = data.tracked_user?.first_name || ''
      leadEdit.last_name = data.tracked_user?.last_name || ''
      leadEdit.email = data.tracked_user?.email || ''
      leadEdit.phone = data.tracked_user?.phone || ''
      leadEdit.value = data.value || 0
      originalStageId = data.stage_id ?? stages.value[0]?.id ?? null
      leadEdit.stage_id = originalStageId
      leadEdit.notes = data.notes || []
      leadEdit.tags = data.tags || []
    } catch {}
  }

  const saveLeadInfo = async () => {
    if (!allowEdit.value || !selectedLeadId.value) return
    const nextValue = Number(leadEdit.value) || 0
    const isMoving = Boolean(leadEdit.stage_id && leadEdit.stage_id !== originalStageId)
    if (isMoving) {
      const targetStage = stages.value.find((s) => Number(s.id) === Number(leadEdit.stage_id))
      if (resolveStageEffectiveEvent(targetStage, selectedLeadTrackedUser.value) === 'Purchase' && nextValue <= 0) {
        toast.error('Valor obrigatório', 'Informe o valor da venda antes de mover para esta etapa.')
        return
      }
    }
    isSavingLead.value = true
    try {
      if (isMoving && leadEdit.stage_id) {
        await api.publicPut(`${baseUrl}/leads/${selectedLeadId.value}/move`, { stage_id: leadEdit.stage_id, value: nextValue })
        originalStageId = leadEdit.stage_id
      }
      await api.publicPut(`${baseUrl}/leads/${selectedLeadId.value}`, {
        first_name: leadEdit.first_name, last_name: leadEdit.last_name,
        email: leadEdit.email, phone: leadEdit.phone, value: nextValue,
      })
      await fetchBoard(true)
      showLeadModal.value = false
      toast.success('Sucesso', 'Lead atualizado.')
    } catch (err: unknown) {
      toast.error('Erro', errorMessage(err, 'Falha ao salvar lead.'))
    } finally {
      isSavingLead.value = false
    }
  }

  const addNote = async () => {
    if (!allowEdit.value || !selectedLeadId.value || !newNoteContent.value.trim()) return
    isAddingNote.value = true
    try {
      const note = await api.publicPost<CrmNote>(`${baseUrl}/leads/${selectedLeadId.value}/notes`, { content: newNoteContent.value.trim() })
      leadEdit.notes.unshift(note)
      newNoteContent.value = ''
      await fetchBoard(true)
    } catch (err: unknown) {
      toast.error('Erro', errorMessage(err, 'Não foi possível salvar anotação.'))
    } finally {
      isAddingNote.value = false
    }
  }

  const confirmDeleteNote = (noteId: number) => {
    noteToDeleteId.value = noteId
    showDeleteNoteModal.value = true
  }

  const doDeleteNote = async () => {
    if (!allowEdit.value || !selectedLeadId.value || !noteToDeleteId.value) return
    try {
      await api.publicDelete(`${baseUrl}/leads/${selectedLeadId.value}/notes/${noteToDeleteId.value}`)
      leadEdit.notes = leadEdit.notes.filter((n) => n.id !== noteToDeleteId.value)
      showDeleteNoteModal.value = false
      await fetchBoard(true)
    } catch (err: unknown) {
      toast.error('Erro', errorMessage(err, 'Não foi possível remover anotação.'))
    }
  }

  const openCreateTagModal = () => {
    newTag.name = ''; newTag.color = getDefaultStageColor()
    showLeadModal.value = false; showCreateTagModal.value = true
  }

  const cancelCreateTagModal = () => {
    showCreateTagModal.value = false
    if (selectedLeadId.value) showLeadModal.value = true
  }

  const createTag = async () => {
    if (!allowEdit.value || !newTag.name.trim()) return
    isCreatingTag.value = true
    try {
      const created = await api.publicPost<CrmTag>(`${baseUrl}/tags`, { name: newTag.name.trim(), color: newTag.color })
      allTags.value.push(created)
      selectedTagToAdd.value = String(created.id)
      showCreateTagModal.value = false
      if (selectedLeadId.value) showLeadModal.value = true
    } catch (err: unknown) {
      toast.error('Erro', errorMessage(err, 'Não foi possível criar tag.'))
    } finally {
      isCreatingTag.value = false
    }
  }

  const addTagToLead = async () => {
    if (!allowEdit.value || !selectedLeadId.value || !selectedTagToAdd.value) return
    const tagId = Number(selectedTagToAdd.value)
    try {
      await api.publicPost(`${baseUrl}/leads/${selectedLeadId.value}/tags/${tagId}`, {})
      const found = allTags.value.find((t) => t.id === tagId)
      if (found && !leadEdit.tags.some((t) => t.id === found.id)) leadEdit.tags.push(found)
      selectedTagToAdd.value = ''
      await fetchBoard(true)
    } catch (err: unknown) {
      toast.error('Erro', errorMessage(err, 'Não foi possível vincular tag.'))
    }
  }

  const removeTag = async (tagId: number) => {
    if (!allowEdit.value || !selectedLeadId.value) return
    try {
      await api.publicDelete(`${baseUrl}/leads/${selectedLeadId.value}/tags/${tagId}`)
      leadEdit.tags = leadEdit.tags.filter((t) => t.id !== tagId)
      await fetchBoard(true)
    } catch (err: unknown) {
      toast.error('Erro', errorMessage(err, 'Não foi possível remover tag.'))
    }
  }

  return {
    showCreateLeadModal, isCreatingManualLead, newLeadForm, openCreateLeadModal, createManualLead,
    showLeadModal, selectedLeadId, selectedLeadTrackedUser, leadEdit, openLeadDetails, saveLeadInfo,
    selectedTagToAdd, newNoteContent, isSavingLead, isAddingNote,
    addNote, showDeleteNoteModal, confirmDeleteNote, doDeleteNote,
    showCreateTagModal, isCreatingTag, newTag, openCreateTagModal, cancelCreateTagModal, createTag, addTagToLead, removeTag,
  }
}
