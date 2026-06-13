<template>
  <div class="period-selector">
    <button
      v-for="opt in options"
      :key="opt.value"
      type="button"
      class="period-btn"
      :class="{ active: modelValue === opt.value }"
      @click="$emit('update:modelValue', opt.value)"
    >
      {{ opt.label }}
    </button>
  </div>
</template>

<script setup lang="ts">
export type PeriodValue = '24h' | '7d' | '30d' | '90d'

defineProps<{
  modelValue: PeriodValue
}>()

defineEmits<{
  'update:modelValue': [value: PeriodValue]
}>()

const options: Array<{ value: PeriodValue; label: string }> = [
  { value: '24h', label: 'Últimas 24h' },
  { value: '7d', label: '7 dias' },
  { value: '30d', label: '30 dias' },
  { value: '90d', label: '90 dias' },
]
</script>

<style scoped>
.period-selector {
  display: inline-flex;
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 0.25rem;
  gap: 0.125rem;
}

.period-btn {
  padding: 0.5rem 0.875rem;
  border: none;
  background: transparent;
  color: var(--text-2);
  font-size: 0.8125rem;
  font-weight: 500;
  border-radius: 7px;
  cursor: pointer;
  transition: all 0.15s ease;
  white-space: nowrap;
}

.period-btn:hover {
  color: var(--text-1);
  background: var(--surface-soft);
}

.period-btn.active {
  background: var(--brand);
  color: white;
}
</style>
