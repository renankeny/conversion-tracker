<template>
  <div class="purchase-row" @click="$emit('select', event)">
    <div class="purchase-icon">
      <ShoppingCart :size="16" />
    </div>

    <div class="purchase-content">
      <div class="purchase-line-1">
        <span class="purchase-pixel">{{ event.pixel_name }}</span>
        <div class="purchase-value-container">
          <span v-if="event.purchase_value > 0" class="purchase-value">
            {{ formatCurrency(event.purchase_value) }}
          </span>
          <span
            class="badge-aggregated"
            :class="event.aggregated_count && event.aggregated_count > 1 ? 'is-active' : 'is-single'"
            :title="event.aggregated_count && event.aggregated_count > 1
              ? `Agrupado: ${event.aggregated_count} produtos somados nesta compra`
              : 'Conversão individual (1 produto)'"
          >
            <svg class="agg-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <polygon points="12 2 2 7 12 12 22 7 12 2" />
              <polyline points="2 17 12 22 22 17" />
              <polyline points="2 12 12 17 22 12" />
            </svg>
            <span>{{ event.aggregated_count && event.aggregated_count > 1 ? `${event.aggregated_count} produtos` : '1 produto' }}</span>
          </span>
        </div>
      </div>

      <div class="purchase-line-2">
        <code class="purchase-id" :title="event.external_id">
          {{ truncate(event.external_id, 14) }}
        </code>
        <span class="purchase-time">{{ formatDate(event.created_at) }}</span>
      </div>

      <div v-if="event.ltv !== undefined" class="purchase-ltv">
        LTV do cliente: <strong>{{ formatCurrency(event.ltv) }}</strong>
      </div>

      <div class="purchase-channels">
        <span
          v-for="ch in CHANNELS"
          :key="ch.id"
          class="channel-chip"
          :data-state="channelState(event, ch.id)"
          :title="channelTooltip(event, ch.id, ch.label)"
        >
          <BrandIcon :channel="ch.id" size="sm" :colored="false" />
          <span class="dot" />
        </span>
      </div>
    </div>

    <div class="purchase-arrow">
      <ChevronRight :size="16" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ShoppingCart, ChevronRight } from 'lucide-vue-next'
import BrandIcon from '~/components/events/BrandIcon.vue'
import type { Channel, GroupedEvent } from '#shared/types'

defineProps<{
  event: GroupedEvent
}>()

defineEmits<{ select: [event: GroupedEvent] }>()

const CHANNELS: Array<{ id: Channel; label: string }> = [
  { id: 'ga4', label: 'GA4' },
  { id: 'meta', label: 'Meta CAPI' },
  { id: 'tiktok', label: 'TikTok Events' },
  { id: 'google_ads', label: 'Google Ads' },
]

type ChannelState = 'ok' | 'pending' | 'error' | 'absent'

const channelState = (ev: GroupedEvent, id: Channel): ChannelState => {
  const c = ev.channels.find(x => x.channel === id)
  if (!c) return 'absent'
  if (c.has_error || c.status === 'failed') return 'error'
  if (c.status === 'queued') return 'pending'
  return 'ok'
}

const channelTooltip = (ev: GroupedEvent, id: Channel, label: string): string => {
  const c = ev.channels.find(x => x.channel === id)
  if (!c) return `${label}: não enviado`
  if (c.has_error || c.status === 'failed') return `${label}: falhou (HTTP ${c.response_code || '-'})`
  if (c.status === 'queued') return `${label}: na fila`
  return `${label}: HTTP ${c.response_code || 200}`
}

const formatCurrency = (v: number): string =>
  new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(v)

const truncate = (s: string, n: number): string =>
  s && s.length > n ? s.slice(0, n) + '…' : s || ''

const formatDate = (iso: string): string => {
  if (!iso) return ''
  const d = new Date(iso)
  return d.toLocaleString('pt-BR', { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' })
}
</script>

<style scoped>
.purchase-row {
  display: flex;
  gap: 0.625rem;
  padding: 0.5rem 0.65rem;
  border-radius: 6px;
  background: var(--surface-soft-2);
  border: 1px solid var(--border-soft);
  transition: all 0.15s ease;
  cursor: pointer;
  align-items: center;
}

.purchase-row:hover {
  background: var(--surface-soft);
  border-color: var(--brand-border);
  transform: translateX(2px);
}

.purchase-icon {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  background: var(--success-soft);
  color: var(--success);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.purchase-content { flex: 1; min-width: 0; }

.purchase-line-1 {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  margin-bottom: 0.2rem;
}

.purchase-pixel {
  font-size: var(--fs-md);
  font-weight: var(--fw-semibold);
  color: var(--text-1);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.purchase-value { font-size: var(--fs-md); font-weight: var(--fw-bold); color: var(--success); white-space: nowrap; }

.purchase-line-2 {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  font-size: var(--fs-sm);
  margin-bottom: 0.375rem;
}

.purchase-id {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  background: var(--surface-soft);
  padding: 0.0625rem 0.375rem;
  border-radius: 4px;
  color: var(--text-2);
}

.purchase-time { color: var(--text-3); }
.purchase-ltv { font-size: var(--fs-sm); color: var(--text-3); margin-bottom: 0.375rem; }
.purchase-ltv strong { color: var(--text-1); font-weight: var(--fw-semibold); }
.purchase-channels { display: flex; gap: 0.3125rem; }

.channel-chip {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  padding: 0.125rem 0.375rem;
  border-radius: 5px;
  border: 1px solid var(--border-soft);
  background: var(--surface-soft);
  color: var(--text-3);
  font-size: var(--fs-xs);
  height: 20px;
}

.channel-chip[data-state="ok"] { background: var(--success-soft); color: var(--success); border-color: var(--success); }
.channel-chip[data-state="pending"] { background: var(--warning-soft); color: var(--warning); border-color: var(--warning); }
.channel-chip[data-state="error"] { background: var(--danger-soft); color: var(--danger); border-color: var(--danger); }

.purchase-arrow { color: var(--text-3); flex-shrink: 0; }
.purchase-value-container { display: flex; align-items: center; gap: 0.35rem; }

.badge-aggregated {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  font-size: var(--fs-2xs);
  font-weight: var(--fw-bold);
  padding: 0.1rem 0.4rem;
  border-radius: 4px;
  white-space: nowrap;
  transition: all 0.15s ease;
}

.badge-aggregated.is-active { background: var(--brand-soft); color: var(--brand); border: 1px solid var(--brand-border); }
.badge-aggregated.is-active .agg-icon { color: var(--brand); }
.badge-aggregated.is-single { background: var(--surface-soft); color: var(--text-3); border: 1px solid var(--border-soft); }
.badge-aggregated.is-single .agg-icon { color: var(--text-3); }
.agg-icon { width: 10px; height: 10px; flex-shrink: 0; }
</style>
