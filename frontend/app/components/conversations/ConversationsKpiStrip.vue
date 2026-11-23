<template>
  <div class="kpi-strip">
    <div class="kpi-item">
      <div class="kpi-head">
        <span class="kpi-label">Conversas no período</span>
        <MessageSquare :size="15" class="kpi-icon brand" />
      </div>
      <div class="kpi-value">{{ kpis.total_conversations }}</div>
      <span class="kpi-sub">Apenas canais rastreados</span>
    </div>

    <div class="kpi-item" :class="{ 'kpi-alert': kpis.waiting_reply_count > 0 }">
      <div class="kpi-head">
        <span class="kpi-label">Aguardando resposta</span>
        <ClockAlert :size="15" class="kpi-icon" :class="kpis.waiting_reply_count > 0 ? 'warn' : 'muted'" />
      </div>
      <div class="kpi-value" :class="{ 'text-warn': kpis.waiting_reply_count > 0 }">
        {{ kpis.waiting_reply_count }}
      </div>
      <span class="kpi-sub">
        {{ kpis.unanswered_count > 0 ? `${kpis.unanswered_count} sem 1ª resposta` : 'Atendimento em dia' }}
      </span>
    </div>

    <div class="kpi-item">
      <div class="kpi-head">
        <span class="kpi-label">Tempo médio 1ª resposta</span>
        <Timer :size="15" class="kpi-icon" :class="slaClass" />
      </div>
      <div class="kpi-value" :class="slaClass">
        {{ formatDurationSLA(kpis.avg_first_response_sec) }}
      </div>
      <span class="kpi-sub">SLA da primeira abordagem</span>
    </div>

    <div class="kpi-item">
      <div class="kpi-head">
        <span class="kpi-label">Vendas em conversas</span>
        <BadgeDollarSign :size="15" class="kpi-icon brand" />
      </div>
      <div class="kpi-value" :class="{ blurred }">
        {{ formatAuditMoney(kpis.won_revenue, currency) }}
      </div>
      <span class="kpi-sub">{{ kpis.won_count }} {{ kpis.won_count === 1 ? 'venda convertida' : 'vendas convertidas' }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { MessageSquare, ClockAlert, Timer, BadgeDollarSign } from 'lucide-vue-next'
import type { ConversationAuditKPIs } from '#shared/types'
import { formatDurationSLA, formatAuditMoney, responseToneClass } from '~/utils/conversationFormatters'

const props = withDefaults(defineProps<{
  kpis: ConversationAuditKPIs
  currency?: string
  blurred?: boolean
}>(), {
  currency: 'BRL',
  blurred: false,
})

const slaClass = computed(() => responseToneClass(props.kpis.avg_first_response_sec))
</script>

<style scoped>
.kpi-strip {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 0.875rem;
}
.kpi-item {
  background: var(--surface-2);
  border: 1px solid var(--border-soft);
  border-radius: 12px;
  padding: 0.9rem 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.3rem;
}
.kpi-item.kpi-alert {
  border-color: var(--warning-border);
}
.kpi-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
}
.kpi-label {
  font-size: var(--fs-xs);
  font-weight: var(--fw-semibold);
  color: var(--text-3);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}
.kpi-icon { flex-shrink: 0; color: var(--text-3); }
.kpi-icon.brand { color: var(--brand); }
.kpi-icon.warn, .text-warn { color: var(--warning); }
.kpi-value {
  font-size: var(--fs-2xl);
  font-weight: var(--fw-bold);
  color: var(--text-strong);
  letter-spacing: -0.02em;
  line-height: 1.2;
}
.kpi-value.blurred { filter: blur(6px); user-select: none; }
.tone-good { color: var(--status-ok); }
.tone-warn { color: var(--warning); }
.tone-bad { color: var(--danger); }
.tone-muted { color: var(--text-2); }
.kpi-sub { font-size: var(--fs-xs); color: var(--text-3); }

@media (max-width: 1023px) {
  .kpi-strip { grid-template-columns: repeat(2, minmax(0, 1fr)); }
}
@media (max-width: 520px) {
  .kpi-strip { grid-template-columns: minmax(0, 1fr); }
}
</style>
