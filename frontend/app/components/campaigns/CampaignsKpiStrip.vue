<template>
  <section class="kpi-strip">
    <article class="kpi-card">
      <span class="kpi-label">Gasto em Anúncios</span>
      <strong class="kpi-value">{{ formatCampaignMoney(summary.spend, currency) }}</strong>
      <span class="kpi-sub">
        {{ formatCompactInt(summary.clicks) }} cliques · {{ formatCompactInt(summary.impressions) }} impr.
      </span>
    </article>

    <article class="kpi-card">
      <span class="kpi-label">Conversas &amp; Leads</span>
      <strong class="kpi-value">{{ formatCompactInt(summary.conversations || summary.leads) }}</strong>
      <span class="kpi-sub">
        <template v-if="summary.conversations > 0">
          {{ formatCompactInt(summary.ai_qualified) }} qualif. IA · {{ formatCompactInt(summary.leads) }} leads
        </template>
        <template v-else>
          {{ formatCompactInt(summary.leads) }} leads rastreados
        </template>
      </span>
    </article>

    <article class="kpi-card">
      <span class="kpi-label">Vendas Líquidas</span>
      <div class="kpi-value-row">
        <strong class="kpi-value">{{ formatCompactInt(summary.net_sales) }}</strong>
        <span v-if="totalRefunds > 0" class="kpi-refund-badge">
          -{{ totalRefunds }} ({{ formatCampaignPct(summary.refund_rate) }})
        </span>
      </div>
      <span class="kpi-sub">
        Brutas: {{ formatCompactInt(summary.gross_sales) }} · Estornos: {{ formatCampaignMoney(summary.refund_amount, currency) }}
      </span>
    </article>

    <article class="kpi-card">
      <span class="kpi-label">Receita Líquida</span>
      <strong class="kpi-value">{{ formatCampaignMoney(summary.net_revenue, currency) }}</strong>
      <span class="kpi-sub">Bruta: {{ formatCampaignMoney(summary.gross_revenue, currency) }}</span>
    </article>

    <article class="kpi-card">
      <span class="kpi-label">ROAS Líquido / CPA Real</span>
      <strong class="kpi-value" :class="{ 'text-ok': summary.net_roas >= 1.5 }">
        {{ formatRoas(summary.net_roas) }}
      </strong>
      <span class="kpi-sub">
        CPA: {{ summary.real_cpa > 0 ? formatCampaignMoney(summary.real_cpa, currency) : '—' }} · Bruto: {{ formatRoas(summary.gross_roas) }}
      </span>
    </article>

    <article class="kpi-card">
      <span class="kpi-label">Lucro Líquido Real</span>
      <strong
        class="kpi-value"
        :class="summary.net_profit >= 0 ? 'text-ok' : 'text-danger'"
      >
        {{ formatCampaignMoney(summary.net_profit, currency) }}
      </strong>
      <span class="kpi-sub">
        ROI: {{ formatCampaignPct(summary.roi) }} · Custos: {{ formatCampaignMoney(summary.operational_costs, currency) }}
      </span>
    </article>
  </section>
</template>

<script setup lang="ts">
import type { CampaignsKpiSummary } from '#shared/types'
import {
  formatCampaignMoney,
  formatCampaignPct,
  formatCompactInt,
  formatRoas,
} from '~/utils/campaignFormatters'

const props = defineProps<{
  summary: CampaignsKpiSummary
  currency: string
}>()

const totalRefunds = computed(() => (props.summary.refund_count || 0) + (props.summary.chargeback_count || 0))
</script>

<style scoped>
.kpi-strip {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(175px, 1fr));
  gap: 0.75rem;
}
.kpi-card {
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 0.9rem 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.3rem;
  min-width: 0;
}
.kpi-label {
  font-size: var(--fs-sm);
  font-weight: var(--fw-medium);
  color: var(--text-3);
}
.kpi-value-row {
  display: flex;
  align-items: center;
  gap: 0.45rem;
  flex-wrap: wrap;
}
.kpi-value {
  font-size: var(--fs-2xl);
  font-weight: var(--fw-bold);
  color: var(--text-strong);
  line-height: 1.15;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.kpi-value.text-ok {
  color: var(--status-ok);
}
.kpi-value.text-danger {
  color: var(--danger);
}
.kpi-refund-badge {
  font-size: var(--fs-2xs);
  font-weight: var(--fw-semibold);
  padding: 0.14rem 0.4rem;
  border-radius: 999px;
  background: var(--danger-soft);
  color: var(--danger);
  border: 1px solid var(--danger-border);
}
.kpi-sub {
  font-size: var(--fs-xs);
  color: var(--text-3);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
</style>
