<template>
  <div class="kpi-card" :class="`variant-${variant}`">
    <div class="kpi-icon" v-html="icon" />
    <div class="kpi-content">
      <p class="kpi-label">{{ label }}</p>
      <p class="kpi-value">{{ formatted }}</p>
      <p v-if="hint" class="kpi-hint">{{ hint }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
interface Props {
  label: string
  value: number | string
  hint?: string
  icon: string
  variant?: 'primary' | 'success' | 'warning' | 'danger' | 'info'
  format?: 'number' | 'currency' | 'percent'
  currency?: string
}

const props = withDefaults(defineProps<Props>(), {
  variant: 'primary',
  format: 'number',
  currency: 'BRL',
})

const formatted = computed(() => {
  if (typeof props.value === 'string') return props.value
  if (props.format === 'currency') {
    return new Intl.NumberFormat('pt-BR', {
      style: 'currency',
      currency: props.currency,
    }).format(props.value)
  }
  if (props.format === 'percent') {
    return new Intl.NumberFormat('pt-BR', {
      style: 'percent',
      minimumFractionDigits: 1,
      maximumFractionDigits: 1,
    }).format(props.value)
  }
  return new Intl.NumberFormat('pt-BR').format(props.value)
})
</script>

<style scoped>
.kpi-card {
  display: flex;
  align-items: flex-start;
  gap: 1rem;
  padding: 1.25rem;
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: 12px;
  box-shadow: var(--shadow-card);
  transition: all 0.2s;
}

.kpi-card:hover {
  border-color: var(--brand-border);
}

.kpi-icon {
  flex-shrink: 0;
  width: 40px;
  height: 40px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.kpi-icon :deep(svg) {
  width: 1.25rem;
  height: 1.25rem;
}

.variant-primary .kpi-icon {
  background: var(--brand-soft);
  color: var(--brand);
}

.variant-success .kpi-icon {
  background: var(--success-soft);
  color: var(--success);
}

.variant-warning .kpi-icon {
  background: var(--warning-soft);
  color: var(--warning);
}

.variant-danger .kpi-icon {
  background: var(--danger-soft);
  color: var(--danger);
}

.variant-info .kpi-icon {
  background: var(--info-soft);
  color: var(--info);
}

.kpi-content {
  min-width: 0;
  flex: 1;
}

.kpi-label {
  font-size: 0.8125rem;
  color: var(--text-2);
  font-weight: 500;
  margin-bottom: 0.25rem;
}

.kpi-value {
  font-family: 'Instrument Sans', sans-serif;
  font-size: 1.625rem;
  font-weight: 700;
  color: var(--text-strong);
  line-height: 1.2;
  word-break: break-all;
}

.kpi-hint {
  font-size: 0.75rem;
  color: var(--text-3);
  margin-top: 0.25rem;
}
</style>
