<template>
  <BaseModal
    :model-value="modelValue"
    title="Configurações do Funil & Etapas"
    size="md"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <p class="modal-desc">
      Organize o fluxo de vendas da sua empresa, ajuste a ordem das etapas ou adicione novas fases.
    </p>

    <div class="pipeline-stages-list">
      <div
        v-for="(st, idx) in stages"
        :key="st.id"
        class="pipeline-stage-item"
      >
        <div class="stage-item-left">
          <span class="stage-order-badge">{{ idx + 1 }}</span>
          <span class="stage-color-pill" :style="{ backgroundColor: st.color }"></span>
          <div class="stage-name-info">
            <span class="stage-item-name">{{ st.name }}</span>
            <span class="stage-item-count">
              {{ getLeadsForStage(st.id).length }} {{ getLeadsForStage(st.id).length === 1 ? 'lead' : 'leads' }}
            </span>
          </div>
        </div>

        <div class="stage-item-actions">
          <span v-if="st.trigger_event" class="stage-trigger-tag">
            {{ st.trigger_event }}
          </span>
          <button
            type="button"
            class="stage-arrow-btn"
            :disabled="idx === 0"
            title="Mover para cima"
            @click="emit('move-up', idx)"
          >
            <ChevronUp :size="14" />
          </button>
          <button
            type="button"
            class="stage-arrow-btn"
            :disabled="idx === stages.length - 1"
            title="Mover para baixo"
            @click="emit('move-down', idx)"
          >
            <ChevronDown :size="14" />
          </button>
          <button
            type="button"
            class="stage-edit-btn"
            title="Editar Etapa"
            @click="emit('edit-stage', st)"
          >
            <Settings :size="14" />
          </button>
        </div>
      </div>
    </div>

    <div class="modal-actions">
      <BaseButton variant="outline" @click="emit('new-stage')">
        <template #iconLeft><Plus :size="15" /></template>
        Nova Etapa
      </BaseButton>
      <BaseButton variant="primary" @click="emit('update:modelValue', false)">
        Concluir
      </BaseButton>
    </div>
  </BaseModal>
</template>

<script setup lang="ts">
import { Plus, Settings, ChevronUp, ChevronDown } from 'lucide-vue-next'
import type { CrmLead, CrmStage } from '#shared/types'

defineProps<{
  modelValue: boolean
  stages: CrmStage[]
  getLeadsForStage: (stageId: string | number) => CrmLead[]
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', val: boolean): void
  (e: 'move-up', idx: number): void
  (e: 'move-down', idx: number): void
  (e: 'edit-stage', stage: CrmStage): void
  (e: 'new-stage'): void
}>()
</script>

<style scoped>
.modal-desc {
  color: var(--text-3);
  font-size: var(--fs-base);
  margin-bottom: 1.25rem;
  line-height: 1.4;
}

.pipeline-stages-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  max-height: 380px;
  overflow-y: auto;
}

.pipeline-stage-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.625rem 0.875rem;
  background: var(--surface-deep);
  border: 1px solid var(--border);
  border-radius: 8px;
}

.stage-item-left {
  display: flex;
  align-items: center;
  gap: 0.625rem;
}

.stage-order-badge {
  width: 22px;
  height: 22px;
  border-radius: 4px;
  background: var(--surface-soft);
  color: var(--text-3);
  font-size: var(--fs-sm);
  font-weight: var(--fw-semibold);
  display: flex;
  align-items: center;
  justify-content: center;
}

.stage-color-pill {
  width: 10px;
  height: 10px;
  border-radius: 50%;
}

.stage-name-info {
  display: flex;
  flex-direction: column;
}

.stage-item-name {
  font-size: var(--fs-md);
  font-weight: var(--fw-semibold);
  color: var(--text-strong);
}

.stage-item-count {
  font-size: var(--fs-xs);
  color: var(--text-3);
}

.stage-item-actions {
  display: flex;
  align-items: center;
  gap: 0.35rem;
}

.stage-trigger-tag {
  font-size: var(--fs-xs);
  font-weight: var(--fw-semibold);
  color: var(--brand);
  background: var(--brand-soft);
  padding: 0.15rem 0.45rem;
  border-radius: 4px;
}

.stage-arrow-btn {
  background: var(--surface-soft);
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--text-2);
  width: 26px;
  height: 24px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all 0.15s;
}

.stage-arrow-btn:disabled {
  opacity: 0.3;
  cursor: not-allowed;
}

.stage-arrow-btn:hover:not(:disabled) {
  background: var(--border-hover);
  color: var(--text-strong);
}

.stage-edit-btn {
  background: none;
  border: none;
  color: var(--text-3);
  cursor: pointer;
  padding: 4px;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: color 0.15s;
}

.stage-edit-btn:hover {
  color: var(--text-strong);
}

.modal-actions {
  display: flex;
  justify-content: space-between;
  gap: 0.75rem;
  margin-top: 1.5rem;
}
</style>
