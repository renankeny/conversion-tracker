<template>
  <div class="form-section goal-card">
    <div class="goal-header">
      <BaseCheckbox
        :model-value="enabled"
        @update:model-value="$emit('update:enabled', $event)"
      >
        <span class="goal-title">Meta Mensal de Vendas deste Pixel</span>
      </BaseCheckbox>
    </div>
    <p class="section-hint">
      Exibe na Dashboard o progresso acumulado do mês atual (e do mês correspondente ao filtrar Mês passado ou datas anteriores).
    </p>

    <div v-if="enabled" class="goal-fields-box">
      <div class="goal-grid">
        <BaseDropdown
          v-model="selectedMonth"
          label="Mês da meta"
          :options="monthOptions"
        />

        <BaseDropdown
          :model-value="goalType"
          label="Tipo de meta"
          :options="goalTypeOptions"
          @update:model-value="$emit('update:goalType', ($event === 'count' ? 'count' : 'revenue'))"
        />

        <BaseInput
          :model-value="inputText"
          type="text"
          :label="goalType === 'count' ? 'Meta em vendas no mês' : `Meta de faturamento no mês (${currency})`"
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
import {
  buildEditableGoalMonthOptions,
  getCurrentSaoPauloYM,
  resolveSalesGoalTargetForMonth,
} from '~/utils/salesGoalMonths'

const props = withDefaults(defineProps<{
  enabled: boolean
  goalType: SalesGoalType
  target: number
  goalMonths?: Record<string, number>
  showOnPublicLink: boolean
  currency?: string
  error?: string
}>(), {
  goalMonths: () => ({}),
  currency: 'BRL',
  error: '',
})

const emit = defineEmits<{
  'update:enabled': [v: boolean]
  'update:goalType': [v: SalesGoalType]
  'update:target': [v: number]
  'update:goalMonths': [v: Record<string, number>]
  'update:showOnPublicLink': [v: boolean]
}>()

const currentYM = getCurrentSaoPauloYM()
const selectedMonth = ref(currentYM)
const inputText = ref('')

const monthOptions = computed(() => buildEditableGoalMonthOptions(props.goalMonths))

const goalTypeOptions = [
  { value: 'revenue', label: 'Meta em valores (faturamento)' },
  { value: 'count', label: 'Meta em número de vendas' },
]

const parseGoalNumber = (raw: string): number => {
  const cleaned = (raw || '').replace(/[^\d.,]/g, '').trim()
  if (!cleaned) return 0
  let normalized = cleaned
  const lastComma = cleaned.lastIndexOf(',')
  const lastDot = cleaned.lastIndexOf('.')
  if (lastComma > lastDot) {
    normalized = cleaned.replace(/\./g, '').replace(',', '.')
  } else if (/^\d{1,3}(\.\d{3})+$/.test(cleaned)) {
    normalized = cleaned.replace(/\./g, '')
  } else if (lastDot > lastComma && lastComma !== -1) {
    normalized = cleaned.replace(/,/g, '')
  }
  const n = Number.parseFloat(normalized)
  if (!Number.isFinite(n) || n <= 0) return 0
  return props.goalType === 'count' ? Math.round(n) : Math.round(n * 100) / 100
}

const resolveDisplayedTarget = (): number => {
  const map = props.goalMonths || {}
  if (Object.prototype.hasOwnProperty.call(map, selectedMonth.value)) {
    return Number(map[selectedMonth.value]) || 0
  }
  return resolveSalesGoalTargetForMonth(props.target, map, selectedMonth.value)
}

const formatGoalNumber = (val: number): string => {
  if (!val || val <= 0) return ''
  return Number.isInteger(val) ? String(val) : String(val).replace('.', ',')
}

watch(
  [selectedMonth, () => props.enabled],
  () => { inputText.value = formatGoalNumber(resolveDisplayedTarget()) },
  { immediate: true },
)

watch(
  () => [props.target, props.goalMonths] as const,
  () => {
    const resolved = resolveDisplayedTarget()
    if (parseGoalNumber(inputText.value) !== resolved) {
      inputText.value = formatGoalNumber(resolved)
    }
  },
)

const onTargetInput = (val: string | number) => {
  const str = String(val ?? '')
  inputText.value = str
  const clean = parseGoalNumber(str)
  const nextMap: Record<string, number> = { ...(props.goalMonths || {}) }
  if (clean > 0) {
    nextMap[selectedMonth.value] = clean
  } else {
    delete nextMap[selectedMonth.value]
  }
  emit('update:goalMonths', nextMap)

  if (selectedMonth.value === currentYM) {
    emit('update:target', clean)
  } else if (props.target <= 0 && clean > 0) {
    emit('update:target', clean)
  }
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
  display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 0.75rem;
}
.public-toggle-row {
  padding-top: 0.25rem;
}
@media (max-width: 768px) {
  .goal-grid { grid-template-columns: minmax(0, 1fr); }
}
</style>
