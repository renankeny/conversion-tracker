<template>
  <div class="kpi-grid">
    <div class="kpi-card">
      <span class="kpi-label">LTV do Cliente</span>
      <span class="kpi-value kpi-value-success">{{ formatCurrency(ltv) }}</span>
    </div>
    <div class="kpi-card">
      <span class="kpi-label">Tempo até Compra</span>
      <span class="kpi-value kpi-value-time truncate" :title="timeToPurchase">
        {{ timeToPurchase || 'Sem compra' }}
      </span>
    </div>
    <div class="kpi-card">
      <span class="kpi-label">Visitas ao Site</span>
      <span class="kpi-value kpi-value-strong">
        {{ visitsCount }} <span class="kpi-unit">acessos</span>
      </span>
    </div>
    <div class="kpi-card">
      <span class="kpi-label">Checkouts Iniciados</span>
      <span class="kpi-value kpi-value-warning">
        {{ checkoutsCount }} <span class="kpi-unit">cliques</span>
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  ltv?: number
  timeToPurchase?: string
  pageviewCount?: number
  checkoutCount?: number
  totalPageviews?: number
  totalCheckouts?: number
}

const props = withDefaults(defineProps<Props>(), {
  ltv: 0,
  timeToPurchase: '',
  pageviewCount: 1,
  checkoutCount: 0,
  totalPageviews: 0,
  totalCheckouts: 0,
})

const visitsCount = computed(() => {
  return props.totalPageviews || props.pageviewCount || 1
})

const checkoutsCount = computed(() => {
  return props.totalCheckouts || props.checkoutCount || 0
})

const formatCurrency = (val: number | undefined): string => {
  return new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(val || 0)
}
</script>

<style scoped>
.kpi-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 0.625rem;
}

.kpi-card {
  padding: 0.75rem 1rem;
  border-radius: 10px;
  border: 1px solid var(--border-soft);
  background: var(--surface-soft-2);
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  min-width: 0;
}

.kpi-label {
  font-size: var(--fs-2xs);
  font-weight: var(--fw-semibold);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--text-3);
}

.kpi-value {
  font-size: var(--fs-xl);
  font-weight: var(--fw-bold);
  line-height: 1.2;
}

.kpi-value-success { color: var(--success); }
.kpi-value-strong { color: var(--text-strong); }
.kpi-value-warning { color: var(--warning); }
.kpi-value-time {
  color: var(--text-1);
  font-size: var(--fs-lg);
  font-weight: var(--fw-semibold);
}

.kpi-unit {
  font-size: var(--fs-xs);
  font-weight: var(--fw-normal);
  color: var(--text-3);
}

@media (max-width: 768px) {
  .kpi-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}
</style>
