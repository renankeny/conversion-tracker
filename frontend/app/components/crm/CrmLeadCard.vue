<template>
  <div class="kanban-card" @pointerdown="onPointerDown" @click="onClick">
    <div class="card-header">
      <div class="lead-info">
        <div class="lead-avatar">{{ getLeadInitials(lead) }}</div>
        <div class="lead-text-group">
          <h4 class="lead-name">{{ getLeadFullName(lead) }}</h4>
          <span v-if="lead.tracked_user?.phone" class="lead-sub">{{ lead.tracked_user.phone }}</span>
          <span v-else-if="lead.tracked_user?.email" class="lead-sub">{{ lead.tracked_user.email }}</span>
          <span v-else class="lead-sub muted">Sem contato</span>
        </div>
      </div>

      <button
        v-if="lead.tracked_user?.phone || lead.tracked_user_id"
        type="button"
        class="quick-wpp-btn"
        title="Ver conversa WhatsApp"
        @click.stop="onClickChat"
      >
        <MessageCircle :size="13" />
      </button>
    </div>

    <div class="card-meta">
      <span class="card-value">{{ formatCurrency(lead.value || 0) }}</span>
      <div class="card-pills">
        <span v-if="originBadge" class="origin-pill">
          {{ originBadge.label }}
        </span>
        <span v-if="lead.has_notes" class="note-pill" title="Possui anotações">
          <StickyNote :size="11" />
        </span>
      </div>
    </div>

    <div v-if="visibleTags.length" class="card-tags">
      <span
        v-for="tag in visibleTags"
        :key="tag.id"
        class="card-tag-chip"
        :style="{ backgroundColor: tag.color + '1a', color: tag.color, borderColor: tag.color + '40' }"
      >
        {{ tag.name }}
      </span>
      <span v-if="extraTagsCount > 0" class="card-tag-more">+{{ extraTagsCount }}</span>
    </div>

    <div class="card-footer">
      <span class="card-date">{{ timeAgo(lead.updated_at || lead.created_at) }}</span>
      <span
        v-if="!isTerminal && stagnantDays >= 2"
        class="stagnant-pill"
        title="Sem movimentação recente"
      >
        <Clock :size="10" /> {{ stagnantDays }}d
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { MessageCircle, StickyNote, Clock } from 'lucide-vue-next'
import type { CrmLead } from '#shared/types'
import { useCrmHelpers } from '~/composables/useCrmHelpers'

const props = defineProps<{
  lead: CrmLead
  isDragging?: boolean
  isTerminal?: boolean
}>()

const emit = defineEmits<{
  (e: 'select', lead: CrmLead): void
}>()

const {
  formatCurrency, timeAgo, getOriginBadge, getStagnantDays,
  getLeadFullName, getLeadInitials,
} = useCrmHelpers()

const originBadge = computed(() => getOriginBadge(props.lead.tracked_user))
const stagnantDays = computed(() => getStagnantDays(props.lead.updated_at))
const visibleTags = computed(() => (props.lead.tags || []).slice(0, 3))
const extraTagsCount = computed(() => Math.max(0, (props.lead.tags?.length || 0) - 3))

const onClickChat = () => {
  emit('select', props.lead)
}

let pointerStartX = 0
let pointerStartY = 0
let hasPointerStart = false

const onPointerDown = (e: PointerEvent | MouseEvent) => {
  pointerStartX = e.clientX
  pointerStartY = e.clientY
  hasPointerStart = true
}

const onClick = (e: MouseEvent) => {
  if (props.isDragging) return
  if (hasPointerStart) {
    const dx = Math.abs(e.clientX - pointerStartX)
    const dy = Math.abs(e.clientY - pointerStartY)
    hasPointerStart = false
    if (dx > 10 || dy > 10) return
  }
  emit('select', props.lead)
}
</script>

<style scoped>
.kanban-card {
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 0.75rem 0.875rem;
  cursor: grab;
  transition: border-color 0.15s ease, transform 0.15s ease, box-shadow 0.15s ease;
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}
.kanban-card:hover {
  border-color: var(--border-hover);
  transform: translateY(-1px);
  box-shadow: var(--shadow-card);
}
.kanban-card:active { cursor: grabbing; }
.card-header { display: flex; justify-content: space-between; align-items: flex-start; gap: 0.5rem; }
.lead-info { display: flex; align-items: center; gap: 0.625rem; flex: 1; min-width: 0; }
.lead-avatar {
  width: 28px; height: 28px; border-radius: 6px;
  background: var(--surface-soft); color: var(--text-1);
  display: flex; align-items: center; justify-content: center;
  font-size: var(--fs-sm); font-weight: var(--fw-bold); flex-shrink: 0;
}
.lead-text-group { display: flex; flex-direction: column; min-width: 0; }
.lead-name {
  font-size: var(--fs-md); font-weight: var(--fw-semibold); color: var(--text-strong);
  line-height: 1.25; margin: 0; white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.lead-sub {
  font-size: var(--fs-sm); color: var(--text-3); margin-top: 2px;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.lead-sub.muted { font-style: italic; opacity: 0.6; }
.quick-wpp-btn {
  width: 26px; height: 26px; border-radius: 6px;
  background: var(--success-soft); color: var(--success);
  display: flex; align-items: center; justify-content: center;
  flex-shrink: 0; text-decoration: none; transition: all 0.15s ease;
}
.quick-wpp-btn:hover { background: var(--success); color: var(--brand-text); transform: scale(1.05); }
.card-meta { display: flex; justify-content: space-between; align-items: center; margin-top: 0.25rem; }
.card-value {
  font-size: var(--fs-base); font-weight: var(--fw-bold); color: var(--brand);
  background: var(--brand-soft); padding: 0.15rem 0.45rem; border-radius: 4px;
}
.card-pills { display: flex; align-items: center; gap: 0.35rem; }
.origin-pill {
  font-size: var(--fs-xs); color: var(--text-2); background: var(--surface-soft);
  border: 1px solid var(--border); padding: 0.15rem 0.45rem; border-radius: 4px; font-weight: var(--fw-medium);
}
.note-pill { color: var(--brand); display: flex; align-items: center; opacity: 0.8; }
.card-tags { display: flex; flex-wrap: wrap; align-items: center; gap: 0.25rem; margin-top: 0.15rem; }
.card-tag-chip {
  font-size: var(--fs-2xs); font-weight: var(--fw-semibold);
  padding: 0.1rem 0.375rem; border-radius: 4px; border: 1px solid transparent;
  max-width: 110px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
}
.card-tag-more { font-size: var(--fs-2xs); color: var(--text-3); font-weight: var(--fw-medium); }
.card-footer {
  display: flex; justify-content: space-between; align-items: center;
  margin-top: 0.25rem; padding-top: 0.35rem; border-top: 1px solid var(--border-soft);
}
.card-date { font-size: var(--fs-xs); color: var(--text-3); }
.stagnant-pill {
  display: inline-flex; align-items: center; gap: 3px;
  color: var(--warning); background: var(--warning-soft);
  padding: 0.1rem 0.35rem; border-radius: 4px; font-size: var(--fs-xs); font-weight: var(--fw-medium);
}
</style>
