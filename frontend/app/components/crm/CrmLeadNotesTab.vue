<template>
  <div class="lead-col-notes">
    <h4 class="box-subtitle">Histórico & Anotações</h4>

    <div class="add-note-box">
      <textarea
        :value="newNoteContent"
        class="note-textarea"
        rows="3"
        placeholder="Escreva uma nova anotação sobre este cliente..."
        @input="emit('update:newNoteContent', ($event.target as HTMLTextAreaElement).value)"
        @keydown.ctrl.enter="emit('add-note')"
        @keydown.meta.enter="emit('add-note')"
      ></textarea>
      <div class="add-note-footer">
        <span class="shortcut-hint">Ctrl + Enter para enviar</span>
        <BaseButton
          variant="primary"
          size="sm"
          :disabled="!newNoteContent.trim() || isAddingNote"
          :loading="isAddingNote"
          @click="emit('add-note')"
        >
          Adicionar
        </BaseButton>
      </div>
    </div>

    <div class="notes-container">
      <div v-if="!notes?.length" class="empty-notes-box">
        <StickyNote :size="20" class="empty-icon" />
        <p>Nenhuma anotação registrada ainda.</p>
      </div>

      <div v-for="note in notes" :key="note.id" class="note-card">
        <p class="note-content">{{ note.content }}</p>
        <div class="note-meta">
          <span class="note-date">{{ timeAgo(note.created_at) }}</span>
          <button
            type="button"
            class="delete-note-btn"
            title="Excluir anotação"
            @click="emit('delete-note', note.id)"
          >
            &times;
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { StickyNote } from 'lucide-vue-next'
import type { CrmNote } from '#shared/types'
import { useCrmHelpers } from '~/composables/useCrmHelpers'

defineProps<{
  notes: CrmNote[]
  newNoteContent: string
  isAddingNote: boolean
}>()

const emit = defineEmits<{
  (e: 'update:newNoteContent', val: string): void
  (e: 'add-note'): void
  (e: 'delete-note', noteId: number): void
}>()

const { timeAgo } = useCrmHelpers()
</script>

<style scoped>
.lead-col-notes {
  display: flex;
  flex-direction: column;
}

.box-subtitle {
  font-size: var(--fs-base);
  font-weight: var(--fw-bold);
  color: var(--text-1);
  margin: 0 0 0.625rem 0;
  text-transform: uppercase;
  letter-spacing: 0.03em;
}

.add-note-box {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  margin-bottom: 0.75rem;
}

.note-textarea {
  padding: 0.5rem 0.75rem;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--surface-2);
  color: var(--text-1);
  font-size: var(--fs-base);
  resize: vertical;
  transition: all 0.15s ease;
}

.note-textarea:focus {
  outline: none;
  border-color: var(--brand);
  box-shadow: 0 0 0 2px var(--brand-soft);
  background: var(--surface-1);
}

.add-note-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.shortcut-hint {
  font-size: var(--fs-xs);
  color: var(--text-3);
}

.notes-container {
  max-height: 340px;
  overflow-y: auto;
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 0.75rem;
  display: flex;
  flex-direction: column;
  gap: 0.625rem;
  background: var(--surface-deep);
}

.empty-notes-box {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  padding: 1.5rem 1rem;
  color: var(--text-3);
  gap: 0.35rem;
}

.empty-icon {
  opacity: 0.4;
}

.empty-notes-box p {
  font-weight: var(--fw-medium);
  font-size: var(--fs-base);
  color: var(--text-2);
}

.note-card {
  background: var(--surface-2);
  padding: 0.625rem 0.75rem;
  border-radius: 6px;
  border: 1px solid var(--border);
}

.note-content {
  font-size: var(--fs-base);
  color: var(--text-1);
  margin-bottom: 0.35rem;
  white-space: pre-wrap;
  line-height: 1.4;
}

.note-meta {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: var(--fs-xs);
  color: var(--text-3);
}

.delete-note-btn {
  background: none;
  border: none;
  color: var(--danger);
  cursor: pointer;
  font-size: var(--fs-lg);
  line-height: 1;
}
</style>
