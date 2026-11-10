<template>
  <div class="pipeline-kpis">
    <div class="kpi-card">
      <div class="kpi-icon icon-negotiation">
        <DollarSign :size="20" />
      </div>
      <div class="kpi-info">
        <span class="kpi-label">Em Negociação</span>
        <span class="kpi-value text-success-val">{{ formatCurrency(totalPipelineValue) }}</span>
      </div>
    </div>

    <div class="kpi-card">
      <div class="kpi-icon icon-leads">
        <Users :size="20" />
      </div>
      <div class="kpi-info">
        <span class="kpi-label">Leads no Funil</span>
        <span class="kpi-value">{{ totalLeadsCount }}</span>
      </div>
    </div>

    <div class="kpi-card">
      <div class="kpi-icon icon-won">
        <CheckCircle2 :size="20" />
      </div>
      <div class="kpi-info">
        <span class="kpi-label">Vendas Ganhas</span>
        <span class="kpi-value text-brand-val">{{ formatCurrency(totalWonValue) }}</span>
      </div>
    </div>

    <div class="kpi-card">
      <div class="kpi-icon icon-ticket">
        <TrendingUp :size="20" />
      </div>
      <div class="kpi-info">
        <span class="kpi-label">Ticket Médio</span>
        <span class="kpi-value">{{ formatCurrency(averageTicket) }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { DollarSign, Users, CheckCircle2, TrendingUp } from 'lucide-vue-next'
import { useCrmHelpers } from '~/composables/useCrmHelpers'

defineProps<{
  totalPipelineValue: number
  totalLeadsCount: number
  totalWonValue: number
  averageTicket: number
}>()

const { formatCurrency } = useCrmHelpers()
</script>

<style scoped>
.pipeline-kpis {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 1rem;
}

.kpi-card {
  background: var(--surface-3);
  border: 1px solid var(--border-soft);
  border-radius: 10px;
  padding: 0.875rem 1rem;
  display: flex;
  align-items: center;
  gap: 0.875rem;
}

.kpi-icon {
  width: 38px;
  height: 38px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.icon-negotiation {
  background: var(--success-soft);
  color: var(--success);
}

.icon-leads {
  background: var(--surface-soft);
  color: var(--text-1);
}

.icon-won {
  background: var(--brand-soft);
  color: var(--brand);
}

.icon-ticket {
  background: var(--warning-soft);
  color: var(--warning);
}

.kpi-info {
  display: flex;
  flex-direction: column;
}

.kpi-label {
  font-size: var(--fs-sm);
  color: var(--text-2);
  font-weight: var(--fw-medium);
  text-transform: uppercase;
  letter-spacing: 0.02em;
}

.kpi-value {
  font-size: var(--fs-xl);
  font-weight: var(--fw-bold);
  color: var(--text-strong);
}

.text-success-val {
  color: var(--success);
}

.text-brand-val {
  color: var(--brand);
}

@media (max-width: 768px) {
  .pipeline-kpis {
    grid-template-columns: repeat(2, 1fr);
    gap: 0.5rem;
  }

  .kpi-card {
    padding: 0.625rem 0.75rem;
    gap: 0.625rem;
    border-radius: 8px;
  }

  .kpi-icon {
    width: 32px;
    height: 32px;
    border-radius: 6px;
  }

  .kpi-icon svg {
    width: 16px;
    height: 16px;
  }

  .kpi-label {
    font-size: var(--fs-xs);
  }

  .kpi-value {
    font-size: var(--fs-lg);
  }
}
</style>
