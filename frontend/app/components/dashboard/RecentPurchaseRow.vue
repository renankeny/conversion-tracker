<template>
  <div class="purchase-row" @click="$emit('select', event)">
    <div class="purchase-icon">
      <ShoppingCart :size="16" />
    </div>

    <div class="purchase-content">
      <div class="purchase-line-1">
        <div class="purchase-buyer" :class="{ 'is-blurred': blurred && hasBuyerIdentity }" :title="buyerTooltip">
          <span class="purchase-buyer-name">{{ primaryBuyerLabel }}</span>
          <span v-if="secondaryPhoneLabel" class="purchase-buyer-phone">· {{ secondaryPhoneLabel }}</span>
        </div>

        <div class="purchase-value-container">
          <span v-if="event.purchase_value > 0" class="purchase-value" :class="{ 'is-blurred': blurred }">
            {{ formatCurrency(event.purchase_value) }}
          </span>
          <span
            class="badge-aggregated"
            :class="event.aggregated_count && event.aggregated_count > 1 ? 'is-active' : 'is-single'"
            :title="event.aggregated_count && event.aggregated_count > 1
              ? `Agrupado: ${event.aggregated_count} produtos somados nesta compra`
              : 'Conversão individual (1 produto)'"
          >
            <Layers :size="10" class="agg-icon" />
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
        LTV do cliente: <strong :class="{ 'is-blurred': blurred }">{{ formatCurrency(event.ltv) }}</strong>
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
import { computed } from 'vue'
import { ShoppingCart, ChevronRight, Layers } from 'lucide-vue-next'
import BrandIcon from '~/components/events/BrandIcon.vue'
import type { Channel, GroupedEvent } from '#shared/types'
import { formatMoney } from '~/utils/dashboardCardDetails'
import { formatShortDateTimeSP } from '~/utils/formatDate'

const props = withDefaults(defineProps<{
  event: GroupedEvent
  currency?: string
  blurred?: boolean
}>(), {
  currency: 'BRL',
  blurred: false,
})

defineEmits<{ select: [event: GroupedEvent] }>()

const CHANNELS: Array<{ id: Channel; label: string }> = [
  { id: 'ga4', label: 'GA4' },
  { id: 'meta', label: 'Meta CAPI' },
  { id: 'tiktok', label: 'TikTok Events' },
  { id: 'google_ads', label: 'Google Ads' },
]

type ChannelState = 'ok' | 'pending' | 'error' | 'absent'

const buyerFullName = computed(() => {
  const first = (props.event.buyer_first_name || '').trim()
  const last = (props.event.buyer_last_name || '').trim()
  if (!first) return last
  if (!last) return first
  const fl = first.toLowerCase()
  const ll = last.toLowerCase()
  if (fl === ll || fl.endsWith(` ${ll}`)) return first
  if (ll.startsWith(`${fl} `)) return last
  return `${first} ${last}`
})

const formattedPhone = computed(() => {
  const raw = (props.event.buyer_phone || '').trim()
  if (!raw) return ''
  const d = raw.replace(/\D/g, '')
  const isIntlNonBR = raw.startsWith('+') && !d.startsWith('55')
  if (!isIntlNonBR) {
    if (d.length === 13 && d.startsWith('55')) return `(${d.slice(2, 4)}) ${d.slice(4, 9)}-${d.slice(9)}`
    if (d.length === 12 && d.startsWith('55')) return `(${d.slice(2, 4)}) ${d.slice(4, 8)}-${d.slice(8)}`
    if (d.length === 11) return `(${d.slice(0, 2)}) ${d.slice(2, 7)}-${d.slice(7)}`
    if (d.length === 10) return `(${d.slice(0, 2)}) ${d.slice(2, 6)}-${d.slice(6)}`
  }
  return d ? `+${d}` : raw
})

const hasBuyerIdentity = computed(() => Boolean(buyerFullName.value || formattedPhone.value))
const primaryBuyerLabel = computed(() => buyerFullName.value || formattedPhone.value || props.event.pixel_name)
const secondaryPhoneLabel = computed(() => (buyerFullName.value && formattedPhone.value ? formattedPhone.value : ''))

const buyerTooltip = computed(() => {
  if (!hasBuyerIdentity.value) return props.event.pixel_name
  const parts = [buyerFullName.value, formattedPhone.value].filter(Boolean)
  return `${parts.join(' · ')} (${props.event.pixel_name})`
})

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

const formatCurrency = (v: number): string => formatMoney(v, props.currency)
const truncate = (s: string, n: number): string => (s && s.length > n ? s.slice(0, n) + '…' : s || '')
const formatDate = (iso: string): string => formatShortDateTimeSP(iso)
</script>

<style scoped>
.purchase-row {
  display: flex; align-items: center; gap: 0.625rem;
  padding: 0.5rem 0.65rem; border-radius: 6px;
  background: var(--surface-soft-2); border: 1px solid var(--border-soft);
  transition: all 0.15s ease; cursor: pointer;
}
.purchase-row:hover {
  background: var(--surface-soft); border-color: var(--brand-border); transform: translateX(2px);
}
.purchase-icon {
  width: 32px; height: 32px; border-radius: 8px; flex-shrink: 0;
  background: var(--success-soft); color: var(--success);
  display: flex; align-items: center; justify-content: center;
}
.purchase-content { flex: 1; min-width: 0; }
.purchase-line-1 {
  display: flex; align-items: baseline; justify-content: space-between;
  gap: 0.5rem; margin-bottom: 0.2rem;
}
.purchase-buyer {
  display: flex; align-items: baseline; flex-wrap: wrap;
  column-gap: 0.375rem; row-gap: 0.1rem; min-width: 0;
}
.purchase-buyer-name {
  font-size: var(--fs-md); font-weight: var(--fw-semibold); color: var(--text-1);
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis; max-width: 100%;
}
.purchase-buyer-phone {
  font-size: var(--fs-sm); font-weight: var(--fw-normal); color: var(--text-2); white-space: nowrap;
}
.purchase-value { font-size: var(--fs-md); font-weight: var(--fw-bold); color: var(--success); white-space: nowrap; }
.is-blurred { filter: blur(6px); user-select: none; }
.purchase-line-2 {
  display: flex; align-items: center; justify-content: space-between;
  gap: 0.5rem; font-size: var(--fs-sm); margin-bottom: 0.375rem;
}
.purchase-id {
  font-family: var(--font-mono); background: var(--surface-soft);
  padding: 0.0625rem 0.375rem; border-radius: 4px; color: var(--text-2);
}
.purchase-time { color: var(--text-3); }
.purchase-ltv { font-size: var(--fs-sm); color: var(--text-3); margin-bottom: 0.375rem; }
.purchase-ltv strong { color: var(--text-1); font-weight: var(--fw-semibold); }
.purchase-channels { display: flex; gap: 0.3125rem; }
.channel-chip {
  display: inline-flex; align-items: center; gap: 0.25rem;
  padding: 0.125rem 0.375rem; border-radius: 5px; height: 20px;
  border: 1px solid var(--border-soft); background: var(--surface-soft);
  color: var(--text-3); font-size: var(--fs-xs);
}
.channel-chip[data-state="ok"] { background: var(--success-soft); color: var(--success); border-color: var(--success); }
.channel-chip[data-state="pending"] { background: var(--warning-soft); color: var(--warning); border-color: var(--warning); }
.channel-chip[data-state="error"] { background: var(--danger-soft); color: var(--danger); border-color: var(--danger); }
.purchase-arrow { color: var(--text-3); flex-shrink: 0; }
.purchase-value-container { display: flex; align-items: center; gap: 0.35rem; flex-shrink: 0; }
.badge-aggregated {
  display: inline-flex; align-items: center; gap: 0.25rem;
  font-size: var(--fs-2xs); font-weight: var(--fw-bold);
  padding: 0.1rem 0.4rem; border-radius: 4px; white-space: nowrap; transition: all 0.15s ease;
}
.badge-aggregated.is-active { background: var(--brand-soft); color: var(--brand); border: 1px solid var(--brand-border); }
.badge-aggregated.is-single { background: var(--surface-soft); color: var(--text-3); border: 1px solid var(--border-soft); }
.agg-icon { flex-shrink: 0; }
</style>
