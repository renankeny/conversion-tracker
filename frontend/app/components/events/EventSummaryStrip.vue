<template>
  <div class="purchase-summary-wrap">
    <!-- Detalhes de Agrupamento de Conversão (Order Bump e Upsell em Purchase) -->
    <div
      class="aggregated-block"
      :class="source?.aggregated_count && source.aggregated_count > 1 ? 'is-active' : 'is-single'"
    >
      <div class="aggregated-header">
        <div class="aggregated-badge">
          <svg class="agg-icon-detail" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <polygon points="12 2 2 7 12 12 22 7 12 2" />
            <polyline points="2 17 12 22 22 17" />
            <polyline points="2 12 12 17 22 12" />
          </svg>
          <span v-if="source?.aggregated_count && source.aggregated_count > 1">
            Conversão Agrupada ({{ source.aggregated_count }} produtos somados)
          </span>
          <span v-else>Conversão Individual (1 produto)</span>
        </div>
        <span v-if="source?.aggregated_count && source.aggregated_count > 1" class="aggregated-summary">
          Total Consolidado: <strong>{{ formatCurrency(source.purchase_value) }}</strong>
        </span>
        <span v-else class="aggregated-summary">
          Valor: <strong>{{ formatCurrency(source?.purchase_value) }}</strong>
        </span>
      </div>
      <p class="aggregated-hint">
        {{ source?.aggregated_count && source.aggregated_count > 1
          ? 'Múltiplas compras deste cliente dentro da janela de agrupamento foram consolidadas em 1 único envio de Purchase para as plataformas de anúncio.'
          : 'Sem orderbump e upsell'
        }}
      </p>
      <div v-if="source?.aggregated_items && source.aggregated_items.length > 1" class="aggregated-items-table">
        <div v-for="(item, idx) in source.aggregated_items" :key="idx" class="aggregated-item-row">
          <span class="item-type-badge" :class="itemTypeClass(item, idx)">
            {{ itemTypeLabel(item, idx) }}
          </span>
          <span class="item-name" :title="item.product_title || 'Produto'">
            {{ item.product_title || `Produto #${idx + 1}` }}
          </span>
          <span v-if="item.order_id" class="item-ord">ID: #{{ item.order_id }}</span>
          <span class="item-price">{{ formatCurrency(item.value) }}</span>
        </div>
      </div>
    </div>

    <!-- LTV no contexto de uma compra -->
    <div v-if="trackedUser" class="customer-block">
      <h4 class="customer-title">Resumo do Cliente</h4>
      <div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
        <div class="customer-stat">
          <span class="stat-label">LTV do Cliente</span>
          <span class="stat-value text-success">{{ formatCurrency(trackedUser.ltv) }}</span>
        </div>
        <div class="customer-stat">
          <span class="stat-label">Tempo até Compra</span>
          <span class="stat-value stat-mono truncate" style="color: var(--text-1)" :title="trackedUser.time_to_purchase">
            {{ trackedUser.time_to_purchase || '-' }}
          </span>
        </div>
        <div class="customer-stat">
          <span class="stat-label">Primeiro Acesso</span>
          <span class="stat-value stat-mono truncate" style="color: var(--text-2)" :title="decodeUTM(trackedUser.first_utm_source || trackedUser.last_utm_source)">
            {{ decodeUTM(trackedUser.first_utm_source || trackedUser.last_utm_source) || 'Direto / Orgânico' }}
          </span>
        </div>
        <div class="customer-stat">
          <span class="stat-label">Último Acesso</span>
          <span class="stat-value stat-mono truncate" style="color: var(--text-2)" :title="decodeUTM(trackedUser.last_utm_source)">
            {{ decodeUTM(trackedUser.last_utm_source) || 'Direto / Orgânico' }}
          </span>
        </div>
      </div>
    </div>
    <div v-else-if="source?.ltv !== undefined" class="customer-block">
      <h4 class="customer-title">Resumo do Cliente</h4>
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
        <div class="customer-stat">
          <span class="stat-label">LTV Acumulado do Cliente</span>
          <span class="stat-value text-success">{{ formatCurrency(source.ltv) }}</span>
        </div>
        <div class="customer-stat">
          <span class="stat-label">Valor desta Compra</span>
          <span class="stat-value" style="color: var(--text-1)">{{ formatCurrency(source.purchase_value) }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { GroupedEvent, TrackedUser, AggregatedItem } from '#shared/types'
import { formatMoney } from '~/utils/dashboardCardDetails'

const props = withDefaults(defineProps<{
  source: GroupedEvent | null
  trackedUser: TrackedUser | null
  currency?: string
}>(), {
  currency: 'BRL',
})

const decodeUTM = (val: string | null | undefined): string => {
  if (!val) return ''
  try {
    return decodeURIComponent(val.replace(/\+/g, ' '))
  } catch {
    return val.replace(/\+/g, ' ')
  }
}

const formatCurrency = (v: number | undefined): string => formatMoney(v || 0, props.currency)

const itemTypeLabel = (item: AggregatedItem, idx: number): string => {
  if (item.item_type === 'order_bump') return 'Order Bump'
  if (item.item_type === 'upsell') return 'Upsell'
  if (item.item_type === 'main' || idx === 0) return 'Principal'
  return `Item ${idx + 1}`
}

const itemTypeClass = (item: AggregatedItem, idx: number): string => {
  if (item.item_type === 'order_bump') return 'type-orderbump'
  if (item.item_type === 'upsell') return 'type-upsell'
  if (item.item_type === 'main' || idx === 0) return 'type-main'
  return 'type-default'
}
</script>

<style scoped>
.purchase-summary-wrap {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.aggregated-block,
.customer-block {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  padding: 1rem 1.25rem;
  border-radius: 10px;
  background: var(--surface-soft-2);
  border: 1px solid var(--border-soft);
}

.aggregated-block.is-active {
  background: var(--brand-soft);
  border-color: var(--brand-border);
}

.aggregated-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
}

.aggregated-badge {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: var(--fs-base);
  font-weight: var(--fw-semibold);
  color: var(--text-1);
}

.agg-icon-detail { width: 18px; height: 18px; color: var(--brand); }
.aggregated-summary { font-size: var(--fs-md); color: var(--text-2); }
.aggregated-summary strong { color: var(--success); font-size: var(--fs-lg); }
.aggregated-hint { font-size: var(--fs-sm); color: var(--text-3); margin: 0; }

.aggregated-items-table {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
  margin-top: 0.25rem;
}

.aggregated-item-row {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.5rem 0.75rem;
  border-radius: 6px;
  background: var(--surface-deep);
  border: 1px solid var(--border-soft);
  font-size: var(--fs-base);
}

.item-type-badge {
  font-size: var(--fs-2xs);
  font-weight: var(--fw-bold);
  text-transform: uppercase;
  letter-spacing: 0.04em;
  padding: 0.0625rem 0.375rem;
  border-radius: 4px;
  background: var(--surface-soft);
  color: var(--text-2);
}

.item-type-badge.type-main { background: var(--accent-sky-soft); color: var(--accent-sky); }
.item-type-badge.type-orderbump { background: var(--warning-soft); color: var(--warning-light); }
.item-type-badge.type-upsell { background: var(--accent-purple-soft); color: var(--accent-purple); }

.item-name {
  flex: 1;
  font-weight: var(--fw-medium);
  color: var(--text-1);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.item-ord { font-size: var(--fs-sm); font-family: var(--font-mono); color: var(--text-3); }
.item-price { font-weight: var(--fw-semibold); color: var(--success); }

.customer-title {
  font-size: var(--fs-xs);
  font-weight: var(--fw-semibold);
  color: var(--text-3);
  text-transform: uppercase;
  letter-spacing: 0.06em;
  margin: 0;
}

.customer-stat {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
  background: var(--surface-deep);
  padding: 0.5rem 0.75rem;
  border-radius: 8px;
  border: 1px solid var(--border-soft);
}

.stat-label { font-size: var(--fs-2xs); font-weight: var(--fw-semibold); text-transform: uppercase; color: var(--text-3); }
.stat-value { font-size: var(--fs-md); font-weight: var(--fw-semibold); }
.stat-mono { font-family: var(--font-mono); font-size: var(--fs-base); }
</style>
