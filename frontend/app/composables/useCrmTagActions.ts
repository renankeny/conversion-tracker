import type { Ref } from 'vue'
import type { CrmTag } from '#shared/types'
import { useAPI } from '~/composables/useAPI'

export const useCrmTagActions = (
  modals: ReturnType<typeof import('~/composables/useCrmModals').useCrmModals>,
  allTags: Ref<CrmTag[]>,
  fetchBoard: (silent?: boolean) => Promise<void>,
) => {
  const api = useAPI()
  const toast = useToast()

  const openCreateTagModal = () => {
    modals.newTag.name = ''
    modals.newTag.color = '#3B82F6'
    modals.showLeadModal.value = false
    modals.showCreateTagModal.value = true
  }

  const cancelCreateTagModal = () => {
    modals.showCreateTagModal.value = false
    if (modals.selectedLeadId.value) {
      modals.showLeadModal.value = true
    }
  }

  const createTag = async () => {
    if (!modals.newTag.name.trim()) return
    modals.isCreatingTag.value = true
    try {
      const res = await api.post<CrmTag>('/admin/crm/tags', {
        name: modals.newTag.name.trim(),
        color: modals.newTag.color,
      })
      allTags.value.push(res)
      modals.selectedTagToAdd.value = String(res.id)
      modals.showCreateTagModal.value = false
      if (modals.selectedLeadId.value) {
        modals.showLeadModal.value = true
      }
      toast.success('Sucesso', 'Tag criada.')
    } catch (e: any) {
      toast.error('Erro', e?.message || 'Erro ao criar tag.')
    } finally {
      modals.isCreatingTag.value = false
    }
  }

  const addTagToLead = async () => {
    if (!modals.selectedTagToAdd.value || !modals.selectedLeadId.value) return
    const tagId = Number(modals.selectedTagToAdd.value)
    try {
      await api.post(`/admin/crm/leads/${modals.selectedLeadId.value}/tags/${tagId}`, {})
      const tag = allTags.value.find((t) => t.id === tagId)
      if (tag && !modals.leadEdit.tags.some((t) => t.id === tag.id)) {
        modals.leadEdit.tags.push(tag)
      }
      modals.selectedTagToAdd.value = ''
      await fetchBoard(true)
    } catch (err: any) {
      toast.error('Erro', err?.message || 'Não foi possível vincular a tag.')
    }
  }

  const removeTag = async (tagId: number) => {
    if (!modals.selectedLeadId.value) return
    try {
      await api.delete(`/admin/crm/leads/${modals.selectedLeadId.value}/tags/${tagId}`)
      modals.leadEdit.tags = modals.leadEdit.tags.filter((t) => t.id !== tagId)
      await fetchBoard(true)
    } catch (err: any) {
      toast.error('Erro', err?.message || 'Não foi possível desvincular a tag.')
    }
  }

  const deleteTagFromAccount = async (tagId: number) => {
    try {
      await api.delete(`/admin/crm/tags/${tagId}`)
      allTags.value = allTags.value.filter((t) => t.id !== tagId)
      modals.leadEdit.tags = modals.leadEdit.tags.filter((t) => t.id !== tagId)
      if (modals.selectedTagToAdd.value === String(tagId)) {
        modals.selectedTagToAdd.value = ''
      }
      toast.success('Sucesso', 'Tag excluída da empresa.')
      await fetchBoard(true)
    } catch (err: any) {
      toast.error('Erro', err?.message || 'Não foi possível excluir a tag.')
    }
  }

  return {
    openCreateTagModal,
    cancelCreateTagModal,
    createTag,
    addTagToLead,
    removeTag,
    deleteTagFromAccount,
  }
}

