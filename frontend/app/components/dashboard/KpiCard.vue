<template>
  <div class="kpi-card" :class="`variant-${variant}`">
    <div class="kpi-top-bar" />
    <div class="kpi-header">
      <p class="kpi-label">{{ label }}</p>
      <div class="kpi-icon">
        <component :is="icon" :size="16" />
      </div>
    </div>
    <div class="kpi-content">
      <p class="kpi-value">{{ formatted }}</p>
      <p v-if="hint" class="kpi-hint">{{ hint }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Component } from 'vue'

interface Props {
  label: string
  value: number | string
  hint?: string
  icon: Component
  variant?: 'primary' | 'success' | 'warning' | 'danger' | 'info'
  format?: 'number' | 'currency' | 'percent' | 'roas'
  currency?: string
}

const props = withDefaults(defineProps<Props>(), {
  variant: 'primary',
  format: 'number',
  currency: 'BRL',
})

const formatted = computed(() => {
  if (typeof props.value === 'string') return props.value
  if (props.format === 'roas') {
    return `${new Intl.NumberFormat('pt-BR', { minimumFractionDigits: 2, maximumFractionDigits: 2 }).format(props.value)}x`
  }
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
  position: relative;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  padding: 1rem 1.125rem;
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: 10px;
  transition: all 0.2s cubic-bezier(0.16, 1, 0.3, 1);
  height: 100%;
  box-sizing: border-box;
  overflow: hidden;
}

.kpi-card:hover {
  border-color: var(--brand-border);
  transform: translateY(-2px);
  box-shadow: 0 8px 20px -4px rgba(0, 0, 0, 0.35), 0 0 12px -2px var(--brand-soft-2);
}

.kpi-top-bar {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 2px;
  background: transparent;
  transition: background 0.2s ease;
}

.variant-primary:hover .kpi-top-bar {
  background: var(--brand);
}

.variant-success:hover .kpi-top-bar {
  background: var(--success);
}

.variant-warning:hover .kpi-top-bar {
  background: var(--warning);
}

.variant-danger:hover .kpi-top-bar {
  background: var(--danger);
}

.variant-info:hover .kpi-top-bar {
  background: var(--info);
}

.kpi-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  margin-bottom: 0.5rem;
}

.kpi-label {
  font-size: var(--fs-sm);
  color: var(--text-3);
  font-weight: var(--fw-semibold);
  text-transform: uppercase;
  letter-spacing: 0.04em;
  margin: 0;
}

.kpi-icon {
  flex-shrink: 0;
  width: 28px;
  height: 28px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: transform 0.2s ease;
}

.kpi-card:hover .kpi-icon {
  transform: scale(1.08);
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
}

.kpi-value {
  font-family: 'Instrument Sans', sans-serif;
  font-size: var(--fs-2xl);
  font-weight: var(--fw-bold);
  color: var(--text-strong);
  line-height: 1.15;
  letter-spacing: -0.02em;
  overflow-wrap: break-word;
  margin: 0;
}

@media (max-width: 640px) {
  .kpi-value {
    font-size: var(--fs-2xl);
  }
}

.kpi-hint {
  font-size: var(--fs-xs);
  color: var(--text-3);
  margin-top: 0.25rem;
  margin-bottom: 0;
}
</style>
