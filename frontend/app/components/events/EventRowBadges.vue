<template>
  <div v-if="variant === 'event'" class="event-name-container">
    <span class="event-name" :class="eventClass(event.event_name)">
      {{ event.event_name }}
    </span>
    <span
      v-if="event.event_name === 'Purchase'"
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

  <div v-else class="channels">
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
</template>

<script setup lang="ts">
import BrandIcon from '~/components/events/BrandIcon.vue'
import type { Channel, GroupedEvent } from '#shared/types'

defineProps<{
  event: GroupedEvent
  variant: 'event' | 'channels'
}>()

const CHANNELS: Array<{ id: Channel; label: string }> = [
  { id: 'ga4', label: 'GA4' },
  { id: 'meta', label: 'Meta CAPI' },
  { id: 'tiktok', label: 'TikTok Events' },
  { id: 'google_ads', label: 'Google Ads' },
]

type ChannelState = 'ok' | 'pending' | 'error' | 'absent'

const channelState = (ev: GroupedEvent, id: Channel): ChannelState => {
  const c = ev.channels.find((x) => x.channel === id)
  if (!c) return 'absent'
  if (c.has_error || c.status === 'failed') return 'error'
  if (c.status === 'queued') return 'pending'
  if (c.status === 'sent' && c.response_code >= 200 && c.response_code < 300) return 'ok'
  return 'error'
}

const channelTooltip = (ev: GroupedEvent, id: Channel, label: string): string => {
  const c = ev.channels.find((x) => x.channel === id)
  if (!c) return `${label}: não enviado`
  if (c.has_error || c.status === 'failed') return `${label}: falhou (HTTP ${c.response_code || '-'})`
  if (c.status === 'queued') return `${label}: na fila`
  return `${label}: HTTP ${c.response_code || 200}`
}

const importantEvents = new Set([
  'Lead', 'InitiateCheckout', 'Purchase', 'AddPaymentInfo',
  'CompleteRegistration', 'AddToCart', 'Contact',
])

const eventClass = (name: string): string =>
  importantEvents.has(name) ? 'event-important' : 'event-default'
</script>

<style scoped>
.event-name-container {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.event-name { font-size: var(--fs-base); font-weight: var(--fw-semibold); }
.event-important { color: var(--brand); }
.event-default { color: var(--text-2); }

.badge-aggregated {
  display: inline-flex;
  align-items: center;
  gap: 0.3rem;
  font-size: var(--fs-xs);
  font-weight: var(--fw-bold);
  padding: 0.15rem 0.5rem;
  border-radius: 4px;
  white-space: nowrap;
  transition: all 0.15s ease;
}

.badge-aggregated.is-active {
  background: var(--brand-soft);
  color: var(--brand);
  border: 1px solid var(--brand-border);
}

.badge-aggregated.is-active .agg-icon { color: var(--brand); }

.badge-aggregated.is-single {
  background: var(--surface-soft);
  color: var(--text-3);
  border: 1px solid var(--border-soft);
}

.badge-aggregated.is-single .agg-icon { color: var(--text-3); }

.agg-icon { width: 12px; height: 12px; flex-shrink: 0; }

.channels { display: flex; gap: 0.375rem; align-items: center; }

.channel-chip {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.3125rem;
  padding: 0.1875rem 0.4375rem;
  border-radius: 6px;
  border: 1px solid var(--border-soft);
  background: var(--surface-soft);
  color: var(--text-3);
  transition: all 0.15s;
  height: 22px;
}

.channel-chip[data-state="ok"] {
  background: var(--success-soft);
  color: var(--success);
  border-color: var(--success);
}

.channel-chip[data-state="pending"] {
  background: var(--warning-soft);
  color: var(--warning);
  border-color: var(--warning);
}

.channel-chip[data-state="error"] {
  background: var(--danger-soft);
  color: var(--danger);
  border-color: var(--danger);
}
</style>
