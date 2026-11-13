import type { CrmNote } from '#shared/types'
import { useAPI } from '~/composables/useAPI'
import { errorMessage } from '~/utils/errorMessage'

export const useCrmLeadNotes = (
  modals: ReturnType<typeof import('~/composables/useCrmModals').useCrmModals>,
  fetchBoard: (silent?: boolean) => Promise<void>,
) => {
  const api = useAPI()
  const toast = useToast()

  const addNote = async () => {
    if (!modals.newNoteContent.value.trim() || !modals.selectedLeadId.value) return
    modals.isAddingNote.value = true
    try {
      const note = await api.post<CrmNote>(`/admin/crm/leads/${modals.selectedLeadId.value}/notes`, {
        content: modals.newNoteContent.value.trim(),
      })
      modals.leadEdit.notes.unshift(note)
      modals.newNoteContent.value = ''
      await fetchBoard(true)
    } catch (err: unknown) {
      toast.error('Erro', errorMessage(err, 'Não foi possível salvar a anotação.'))
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
    } catch (err: unknown) {
      toast.error('Erro', errorMessage(err, 'Falha ao excluir anotação.'))
    }
  }

  return { addNote, confirmDeleteNote, doDeleteNote }
}

