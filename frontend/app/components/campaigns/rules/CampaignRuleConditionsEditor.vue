<template>
  <section class="cond-section">
    <div class="cond-header">
      <div>
        <h4 class="cond-title">Condições (todas precisam ser verdadeiras)</h4>
        <p class="cond-sub">
          {{ isSchedule ? 'Opcional em agendamentos: deixe vazio para disparar sempre no horário configurado.' : 'Combine métricas de mídia com vendas reais líquidas, reembolsos e WhatsApp IA.' }}
        </p>
      </div>
      <BaseButton variant="outline" size="sm" type="button" @click="addCondition">
        <Plus :size="14" />
        <span>Adicionar condição</span>
      </BaseButton>
    </div>

    <div v-if="modelValue.length === 0" class="cond-empty">
      Nenhuma condição definida{{ isSchedule ? ' — a ação executará diretamente no horário agendado.' : '. Adicione ao menos uma condição.' }}
    </div>

    <div v-else class="cond-list">
      <div v-for="(cond, idx) in modelValue" :key="idx" class="cond-row">
        <span class="cond-prefix">{{ idx === 0 ? 'SE' : 'E' }}</span>
        <div class="cond-metric">
          <BaseDropdown
            :model-value="cond.metric"
            :options="RULE_METRIC_OPTIONS"
            size="sm"
            @update:model-value="(v) => updateField(idx, 'metric', String(v || 'spend'))"
          />
        </div>
        <div class="cond-op">
          <BaseDropdown
            :model-value="cond.operator"
            :options="RULE_OPERATOR_OPTIONS"
            size="sm"
            @update:model-value="(v) => updateField(idx, 'operator', String(v || '>') as CampaignRuleOperator)"
          />
        </div>
        <div class="cond-val">
          <BaseInput
            :model-value="String(cond.value)"
            type="number"
            placeholder="0"
            @update:model-value="(v) => updateField(idx, 'value', Number(v || 0))"
          />
        </div>
        <button
          type="button"
          class="cond-remove"
          :disabled="!isSchedule && modelValue.length <= 1"
          title="Remover condição"
          @click="removeCondition(idx)"
        >
          <Trash2 :size="14" />
        </button>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { Plus, Trash2 } from 'lucide-vue-next'
import type { CampaignRuleCondition, CampaignRuleOperator } from '#shared/types'
import { RULE_METRIC_OPTIONS, RULE_OPERATOR_OPTIONS } from '~/utils/campaignRuleOptions'

const props = defineProps<{
  modelValue: CampaignRuleCondition[]
  isSchedule?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: CampaignRuleCondition[]]
}>()

const addCondition = () => {
  emit('update:modelValue', [...props.modelValue, { metric: 'spend', operator: '>', value: 100 }])
}

const removeCondition = (idx: number) => {
  emit('update:modelValue', props.modelValue.filter((_, i) => i !== idx))
}

const updateField = <K extends keyof CampaignRuleCondition>(idx: number, key: K, val: CampaignRuleCondition[K]) => {
  const next = props.modelValue.map((item, i) => (i === idx ? { ...item, [key]: val } : item))
  emit('update:modelValue', next)
}
</script>

<style scoped>
.cond-section {
  display: flex;
  flex-direction: column;
  gap: 0.65rem;
  padding: 0.85rem 0.95rem;
  border-radius: 10px;
  background: var(--surface-soft-2);
  border: 1px solid var(--border-soft);
}
.cond-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 0.75rem;
  flex-wrap: wrap;
}
.cond-title { margin: 0; font-size: var(--fs-md); font-weight: var(--fw-semibold); color: var(--text-1); }
.cond-sub { margin: 0.15rem 0 0; font-size: var(--fs-sm); color: var(--text-3); }
.cond-empty {
  padding: 0.65rem 0.75rem;
  border-radius: 8px;
  background: var(--surface-deep);
  border: 1px dashed var(--border);
  font-size: var(--fs-sm);
  color: var(--text-3);
}
.cond-list { display: flex; flex-direction: column; gap: 0.5rem; }
.cond-row {
  display: grid;
  grid-template-columns: 32px minmax(0, 1.5fr) minmax(0, 1fr) 110px 32px;
  align-items: center;
  gap: 0.45rem;
}
.cond-prefix {
  font-size: var(--fs-xs);
  font-weight: var(--fw-bold);
  color: var(--brand);
  text-align: center;
  padding: 0.25rem 0;
  border-radius: 6px;
  background: var(--brand-soft);
}
.cond-remove {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  border: 1px solid var(--border-soft);
  background: transparent;
  color: var(--text-3);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
}
.cond-remove:hover:not(:disabled) { color: var(--danger); border-color: var(--danger-border); background: var(--danger-soft); }
.cond-remove:disabled { opacity: 0.35; cursor: not-allowed; }
@media (max-width: 720px) {
  .cond-row { grid-template-columns: 1fr 1fr; }
  .cond-prefix { grid-column: span 2; text-align: left; padding: 0.2rem 0.5rem; }
}
</style>
