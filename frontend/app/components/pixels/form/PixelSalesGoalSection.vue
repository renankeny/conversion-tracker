<template>
  <div class="form-section goal-card">
    <div class="goal-header">
      <BaseCheckbox
        :model-value="enabled"
        @update:model-value="$emit('update:enabled', $event)"
      >
        <span class="goal-title">Meta de Vendas deste Pixel</span>
      </BaseCheckbox>
    </div>
    <p class="section-hint">
      Exibe uma barra de progresso de 0% até 100% acompanhando o resultado do período selecionado na Dashboard.
    </p>

    <div v-if="enabled" class="goal-fields-box">
      <div class="goal-grid">
        <BaseDropdown
          :model-value="goalType"
          label="Tipo de meta"
          :options="goalTypeOptions"
          @update:model-value="$emit('update:goalType', ($event === 'count' ? 'count' : 'revenue'))"
        />

        <BaseInput
          :model-value="target > 0 ? String(target) : ''"
          type="number"
          :label="goalType === 'count' ? 'Meta em número de vendas' : `Meta em faturamento (${currency})`"
          :placeholder="goalType === 'count' ? 'Ex: 50' : 'Ex: 25000'"
          :error="error"
          @update:model-value="onTargetInput"
        />
      </div>

      <div class="public-toggle-row">
        <BaseCheckbox
          :model-value="showOnPublicLink"
          label="Mostrar esta meta de vendas no link público do cliente"
          hint="Você também pode ligar ou desligar essa opção depois diretamente em Permissões do link na Dashboard."
          @update:model-value="$emit('update:showOnPublicLink', $event)"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { SalesGoalType } from '#shared/types'

withDefaults(defineProps<{
  enabled: boolean
  goalType: SalesGoalType
  target: number
  showOnPublicLink: boolean
  currency?: string
  error?: string
}>(), {
  currency: 'BRL',
  error: '',
})

const emit = defineEmits<{
  'update:enabled': [v: boolean]
  'update:goalType': [v: SalesGoalType]
  'update:target': [v: number]
  'update:showOnPublicLink': [v: boolean]
}>()

const goalTypeOptions = [
  { value: 'revenue', label: 'Meta em valores (faturamento)' },
  { value: 'count', label: 'Meta em número de vendas' },
]

const onTargetInput = (val: string | number) => {
  if (val === '' || val === null || val === undefined) {
    emit('update:target', 0)
    return
  }
  const n = Number(val)
  emit('update:target', Number.isFinite(n) && n >= 0 ? n : 0)
}
</script>

<style scoped>
.form-section {
  display: flex; flex-direction: column; gap: 0.5rem;
  padding-top: 0.5rem; border-top: 1px solid var(--border-soft);
}
.goal-card {
  background: var(--surface-2); border: 1px solid var(--border-soft);
  border-radius: 8px; padding: 1rem;
}
.goal-title { font-weight: var(--fw-semibold); color: var(--text-1); }
.section-hint { font-size: var(--fs-base); color: var(--text-3); margin: 0; }
.goal-fields-box {
  margin-top: 0.75rem; padding-top: 0.85rem;
  border-top: 1px dashed var(--border-soft);
  display: flex; flex-direction: column; gap: 0.85rem;
}
.goal-grid {
  display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 0.875rem;
}
.public-toggle-row {
  padding-top: 0.25rem;
}
@media (max-width: 640px) {
  .goal-grid { grid-template-columns: minmax(0, 1fr); }
}
</style>
