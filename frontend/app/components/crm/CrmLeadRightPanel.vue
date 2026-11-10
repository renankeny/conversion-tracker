<template>
  <div class="right-panel">
    <div class="panel-switch">
      <button
        type="button"
        class="switch-btn"
        :class="{ active: activeTab === 'whatsapp' }"
        @click="activeTab = 'whatsapp'"
      >
        <MessageCircle :size="14" />
        <span>Conversa WhatsApp</span>
      </button>

      <button
        type="button"
        class="switch-btn"
        :class="{ active: activeTab === 'notes' }"
        @click="activeTab = 'notes'"
      >
        <StickyNote :size="14" />
        <span>Anotações</span>
        <span v-if="notes?.length" class="count-pill">{{ notes.length }}</span>
      </button>
    </div>

    <WhatsAppChatPanel
      v-show="activeTab === 'whatsapp'"
      :crm-lead-id="leadId"
      :tracked-user-id="trackedUserId"
    />

    <CrmLeadNotesTab
      v-show="activeTab === 'notes'"
      :notes="notes"
      :new-note-content="newNoteContent"
      :is-adding-note="isAddingNote"
      @update:new-note-content="emit('update:newNoteContent', $event)"
      @add-note="emit('add-note')"
      @delete-note="emit('delete-note', $event)"
    />
  </div>
</template>

<script setup lang="ts">
import { MessageCircle, StickyNote } from 'lucide-vue-next'
import type { CrmNote } from '#shared/types'
import WhatsAppChatPanel from '~/components/whatsapp/WhatsAppChatPanel.vue'
import CrmLeadNotesTab from '~/components/crm/CrmLeadNotesTab.vue'

defineProps<{
  leadId: number | null
  trackedUserId?: number | null
  notes: CrmNote[]
  newNoteContent: string
  isAddingNote: boolean
}>()

const emit = defineEmits<{
  'update:newNoteContent': [val: string]
  'add-note': []
  'delete-note': [noteId: number]
}>()

const activeTab = ref<'whatsapp' | 'notes'>('whatsapp')
</script>

<style scoped>
.right-panel { display: flex; flex-direction: column; gap: 0.75rem; min-width: 0; }
.panel-switch {
  display: flex; gap: 0.25rem; padding: 0.25rem;
  background: var(--surface-deep); border: 1px solid var(--border-soft); border-radius: 9px;
}
.switch-btn {
  flex: 1; display: inline-flex; align-items: center; justify-content: center; gap: 0.4rem;
  padding: 0.45rem 0.75rem; border-radius: 6px; border: none; background: transparent;
  color: var(--text-3); font-size: var(--fs-sm); font-weight: var(--fw-semibold); cursor: pointer;
}
.switch-btn.active { background: var(--surface-soft); color: var(--brand); }
.count-pill {
  font-size: var(--fs-2xs); padding: 0.05rem 0.375rem; border-radius: 999px;
  background: var(--brand-soft); color: var(--brand);
}
</style>
