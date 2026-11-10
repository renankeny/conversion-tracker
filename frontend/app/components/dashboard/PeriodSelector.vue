<template>
  <div class="period-selector">
    <BaseDropdown v-model="internalValue" :options="options" class="period-dropdown" />

    <div v-if="internalValue === 'custom'" class="custom-dates">
      <input type="date" v-model="internalFrom" class="date-input" title="Data Inicial" />
      <span class="date-separator">até</span>
      <input type="date" v-model="internalTo" class="date-input" title="Data Final" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import BaseDropdown from '~/components/BaseDropdown.vue'

export type PeriodValue = 'today' | 'yesterday' | '7d' | '30d' | '60d' | '90d' | 'custom'

const props = defineProps<{
  modelValue: string
  customFrom?: string
  customTo?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
  'update:customFrom': [value: string]
  'update:customTo': [value: string]
}>()

const internalValue = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v)
})

const internalFrom = computed({
  get: () => props.customFrom || '',
  set: (v) => emit('update:customFrom', v)
})

const internalTo = computed({
  get: () => props.customTo || '',
  set: (v) => emit('update:customTo', v)
})

const options: Array<{ value: PeriodValue; label: string }> = [
  { value: 'today', label: 'Hoje' },
  { value: 'yesterday', label: 'Ontem' },
  { value: '7d', label: 'Últimos 7 dias' },
  { value: '30d', label: 'Últimos 30 dias' },
  { value: '60d', label: 'Últimos 60 dias' },
  { value: '90d', label: 'Últimos 90 dias' },
  { value: 'custom', label: 'Personalizado' }
]
</script>

<style scoped>
.period-selector {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  flex-wrap: wrap;
}

.period-dropdown {
  min-width: 150px;
}

.custom-dates {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  flex-wrap: wrap;
  animation: fadeIn 0.15s ease-out;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateX(-6px); }
  to { opacity: 1; transform: translateX(0); }
}

.date-separator {
  color: var(--text-3);
  font-size: var(--fs-sm);
  font-weight: var(--fw-medium);
}

.date-input {
  background: var(--surface-2);
  color: var(--text-1);
  border: 1px solid var(--border);
  height: 36px;
  padding: 0.35rem 0.65rem;
  font-size: var(--fs-sm);
  border-radius: 6px;
  outline: none;
  transition: all 0.15s ease;
  color-scheme: inherit;
}

.date-input:focus {
  border-color: var(--brand);
  box-shadow: 0 0 0 2px var(--brand-soft);
}
</style>
