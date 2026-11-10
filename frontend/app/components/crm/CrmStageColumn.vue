<template>
  <div
    class="kanban-column"
    :class="{ 'collapsed-column': isCollapsed }"
    :style="{ borderTopColor: stage.color }"
    :title="isCollapsed ? 'Clique para expandir ' + stage.name : undefined"
    @click="isCollapsed ? emit('toggle-collapse', stage.id) : undefined"
  >
    <template v-if="isCollapsed">
      <div class="collapsed-indicator" :style="{ backgroundColor: stage.color }"></div>
      <div class="collapsed-title">
        <span>{{ stage.name }}</span>
        <span class="count-badge-collapsed">{{ visibleStageLeads.length }}</span>
      </div>
      <ChevronRight :size="16" class="expand-icon" />
    </template>

    <template v-else>
      <div class="column-header">
        <div class="column-title" :class="{ 'column-drag-handle': canManageStages }">
          <span class="color-dot" :style="{ backgroundColor: stage.color }"></span>
          <div class="title-and-metrics">
            <h3>{{ stage.name }}</h3>
            <span class="stage-metric-sub">
              {{ visibleStageLeads.length }} {{ visibleStageLeads.length === 1 ? 'lead' : 'leads' }}
              <span class="metric-separator">•</span>
              {{ formatCurrency(stageTotalValue) }}
            </span>
          </div>
        </div>

        <div class="column-header-actions">
          <button
            type="button"
            class="icon-btn"
            title="Adicionar Lead nesta Etapa"
            @click.stop="emit('add-lead', stage.id)"
          >
            <Plus :size="15" />
          </button>
          <button
            v-if="canManageStages"
            type="button"
            class="icon-btn"
            title="Configurar Etapa"
            @click.stop="emit('config-stage', stage)"
          >
            <Settings :size="15" />
          </button>
          <button
            type="button"
            class="icon-btn"
            title="Recolher Coluna"
            @click.stop="emit('toggle-collapse', stage.id)"
          >
            <ChevronDown :size="15" />
          </button>
        </div>
      </div>

      <draggable
        v-model="stageLeads"
        class="kanban-list"
        group="leads"
        item-key="id"
        :animation="200"
        ghost-class="ghost-card"
        :fallback-tolerance="6"
        :data-stage-id="stage.id"
        @start="onLeadDragStart"
        @end="onLeadDragEnd"
      >
        <template #item="{ element }">
          <CrmLeadCard
            v-show="matchesFilter(element)"
            :lead="element"
            :is-dragging="isDraggingLead"
            :is-terminal="isTerminalStage"
            @select="emit('select-lead', $event)"
          />
        </template>
      </draggable>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { Plus, Settings, ChevronDown, ChevronRight } from 'lucide-vue-next'
import draggable from 'vuedraggable'
import type { CrmLead, CrmStage } from '#shared/types'
import CrmLeadCard from '~/components/crm/CrmLeadCard.vue'
import { useCrmHelpers } from '~/composables/useCrmHelpers'

const props = withDefaults(defineProps<{
  stage: CrmStage
  modelValue?: CrmLead[]
  isCollapsed: boolean
  canManageStages?: boolean
  matchesFilter: (lead: CrmLead) => boolean
}>(), {
  canManageStages: true,
})

const emit = defineEmits<{
  (e: 'update:modelValue', val: CrmLead[]): void
  (e: 'toggle-collapse', stageId: string | number): void
  (e: 'add-lead', stageId: number): void
  (e: 'config-stage', stage: CrmStage): void
  (e: 'select-lead', lead: CrmLead): void
  (e: 'lead-drag-end', evt: any): void
}>()

const { formatCurrency, isWonStage, isLostStage } = useCrmHelpers()
const isDraggingLead = ref(false)

const isTerminalStage = computed(() => isWonStage(props.stage) || isLostStage(props.stage))

const stageLeads = computed({
  get: () => props.modelValue || [],
  set: (val) => emit('update:modelValue', val),
})

const visibleStageLeads = computed(() => stageLeads.value.filter(props.matchesFilter))

const stageTotalValue = computed(() => {
  return visibleStageLeads.value.reduce((acc, l) => acc + (Number(l.value) || 0), 0)
})

const onLeadDragStart = () => {
  isDraggingLead.value = true
}

const onLeadDragEnd = (evt: any) => {
  setTimeout(() => {
    isDraggingLead.value = false
  }, 100)
  emit('lead-drag-end', evt)
}
</script>

<style scoped>
.kanban-column {
  flex: 0 0 310px;
  background: var(--surface-deep);
  border-radius: 12px;
  border: 1px solid var(--border-soft);
  border-top-width: 3px;
  display: flex;
  flex-direction: column;
  max-height: 100%;
  transition: all 0.2s ease;
}
.column-header {
  padding: 0.875rem 1rem;
  border-bottom: 1px solid var(--border-soft);
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: var(--surface-3);
  border-top-left-radius: 9px;
  border-top-right-radius: 9px;
}
.column-title { display: flex; align-items: center; gap: 0.625rem; flex: 1; }
.column-drag-handle { cursor: grab; }
.column-drag-handle:active { cursor: grabbing; }
.color-dot { width: 10px; height: 10px; border-radius: 50%; flex-shrink: 0; }
.title-and-metrics h3 { font-size: var(--fs-md); font-weight: var(--fw-bold); color: var(--text-strong); margin: 0; line-height: 1.2; }
.stage-metric-sub { font-size: var(--fs-xs); color: var(--text-2); font-weight: var(--fw-medium); }
.metric-separator { margin: 0 3px; opacity: 0.5; }
.column-header-actions { display: flex; align-items: center; gap: 0.25rem; }
.icon-btn {
  background: none; border: none; color: var(--text-2); cursor: pointer;
  padding: 0.35rem; border-radius: 6px; display: flex; align-items: center;
  justify-content: center; transition: background 0.15s, color 0.15s;
}
.icon-btn:hover { background: var(--surface-1); color: var(--text-strong); }
.collapsed-column {
  flex: 0 0 52px; cursor: pointer; align-items: center;
  padding: 1rem 0; gap: 1rem; position: relative;
}
.collapsed-indicator { width: 6px; height: 24px; border-radius: 3px; }
.collapsed-title {
  writing-mode: vertical-rl; transform: rotate(180deg); font-weight: var(--fw-bold);
  font-size: var(--fs-base); color: var(--text-strong); display: flex;
  align-items: center; gap: 0.75rem; letter-spacing: 0.03em;
}
.count-badge-collapsed {
  background: var(--surface-1); padding: 0.15rem 0.4rem;
  border-radius: 8px; font-size: var(--fs-xs); color: var(--text-2);
}
.expand-icon { color: var(--text-2); margin-top: auto; }
.kanban-list {
  flex: 1; padding: 0.75rem; overflow-y: auto; min-height: 120px;
  display: flex; flex-direction: column; gap: 0.625rem;
}
:deep(.ghost-card) { opacity: 0.35; background: var(--brand-soft); border: 2px dashed var(--brand); }
@media (max-width: 1023px) {
  .kanban-column {
    flex: 0 0 calc(100vw - 3.5rem); max-width: 325px; scroll-snap-align: start;
    max-height: calc(100vh - 120px); min-height: 420px;
  }
  .kanban-list { max-height: calc(100vh - 200px); overflow-y: auto; -webkit-overflow-scrolling: touch; }
  .collapsed-column { flex: 0 0 44px; scroll-snap-align: start; }
}
</style>
