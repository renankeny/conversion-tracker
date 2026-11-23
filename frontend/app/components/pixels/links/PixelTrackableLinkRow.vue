<template>
  <div class="link-row" :class="{ paused: !link.enabled }">
    <div class="row-top">
      <div class="title-col">
        <div class="title-line">
          <span class="link-name">{{ link.name }}</span>
          <BaseBadge :variant="link.enabled ? 'success' : 'default'">
            {{ link.enabled ? 'Ativo' : 'Pausado' }}
          </BaseBadge>
          <span class="routing-pill">
            <span v-if="connState" class="conn-dot" :class="`dot-${connState}`" />
            {{ routingLabel }}
          </span>
        </div>

        <div class="url-bar">
          <code class="url-code">{{ link.public_url || `/w/${link.slug}` }}</code>
          <button type="button" class="copy-btn" title="Copiar link rastreável" @click="$emit('copy', link)">
            <Check v-if="copied" :size="13" />
            <Copy v-else :size="13" />
            <span>{{ copied ? 'Copiado' : 'Copiar' }}</span>
          </button>
        </div>
      </div>

      <div v-if="canEdit" class="actions-col">
        <BaseButton variant="ghost" size="sm" @click="$emit('edit', link)">Editar</BaseButton>
        <button type="button" class="del-btn" title="Excluir link" @click="$emit('delete', link)">
          <Trash2 :size="14" />
        </button>
      </div>
    </div>

    <div class="row-bottom">
      <div class="metrics-strip">
        <div class="metric-item">
          <span class="metric-label">Cliques</span>
          <strong class="metric-val">{{ link.clicks_count }}</strong>
        </div>
        <span class="metric-sep">→</span>
        <div class="metric-item">
          <span class="metric-label">Conversas</span>
          <strong class="metric-val">{{ link.chats_count }}</strong>
          <span v-if="link.clicks_count > 0" class="metric-rate">{{ chatRate }}%</span>
        </div>
        <span class="metric-sep">→</span>
        <div class="metric-item">
          <span class="metric-label">Vendas</span>
          <strong class="metric-val highlight">{{ link.sales_count }}</strong>
          <span v-if="link.sales_revenue > 0" class="metric-rev">{{ formattedRevenue }}</span>
        </div>
      </div>

      <div class="channels-strip">
        <template v-if="link.send_events_enabled && activeChannels.length">
          <span class="ev-tag">{{ link.event_name }} · {{ triggerShortLabel }}</span>
          <span v-for="ch in activeChannels" :key="ch.key" class="chan-chip" :title="ch.label">
            <BrandIcon :channel="ch.icon" size="sm" colored />
            <span>{{ ch.short }}</span>
          </span>
        </template>
        <span v-else class="ev-tag muted">Apenas rastreio interno</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Copy, Check, Trash2 } from 'lucide-vue-next'
import BrandIcon from '~/components/events/BrandIcon.vue'
import type { TrackableLink, WhatsAppInstance } from '#shared/types'
import {
  getWhatsAppProviderLabel,
  getWhatsAppStatusLabel,
  getWhatsAppStatusState,
  isWhatsAppStatusConnected,
} from '~/utils/whatsappInstanceStatus'

const props = withDefaults(defineProps<{
  link: TrackableLink
  instances?: WhatsAppInstance[]
  canEdit?: boolean
  copied?: boolean
  currency?: string
}>(), {
  instances: () => [],
  canEdit: true,
  copied: false,
  currency: 'BRL',
})

defineEmits<{
  copy: [link: TrackableLink]
  edit: [link: TrackableLink]
  delete: [link: TrackableLink]
}>()

const connState = computed(() => {
  if (props.link.routing_mode === 'manual_phone') return null
  if (props.link.routing_mode === 'specific_instance') {
    const live = props.instances.find((i) => i.id === props.link.whatsapp_instance_id)
    const st = live ? live.connection_status : props.link.instance_status
    const en = live ? live.enabled : true
    return getWhatsAppStatusState(st, en)
  }
  if (!props.instances.length) return 'disconnected'
  if (props.instances.some((i) => isWhatsAppStatusConnected(i.connection_status, i.enabled))) return 'connected'
  if (props.instances.some((i) => getWhatsAppStatusState(i.connection_status, i.enabled) === 'connecting')) return 'connecting'
  return 'disconnected'
})

const routingLabel = computed(() => {
  if (props.link.routing_mode === 'specific_instance') {
    const live = props.instances.find((i) => i.id === props.link.whatsapp_instance_id)
    const name = live?.name || props.link.instance_name || 'WhatsApp'
    const prov = getWhatsAppProviderLabel(live?.provider || props.link.instance_provider)
    const st = getWhatsAppStatusLabel(
      live ? live.connection_status : props.link.instance_status,
      live?.provider || props.link.instance_provider,
      live ? live.enabled : true,
    )
    return `${name} · ${prov} (${st})`
  }
  if (props.link.routing_mode === 'manual_phone') {
    return props.link.fallback_phone ? `Fixo: +${props.link.fallback_phone}` : 'Número fixo'
  }
  const openCount = props.instances.filter((i) => isWhatsAppStatusConnected(i.connection_status, i.enabled)).length
  if (openCount > 1) return `Rodízio (${openCount} conectados)`
  if (openCount === 1) return 'Auto Pixel (1 conectado)'
  return props.instances.length ? 'Auto Pixel (Desconectado)' : 'Auto Pixel'
})

const triggerShortLabel = computed(() =>
  props.link.event_trigger_mode === 'on_click' ? 'no clique' : 'na 1ª msg',
)

const chatRate = computed(() => {
  if (!props.link.clicks_count) return '0'
  return Math.min(100, Math.round((props.link.chats_count / props.link.clicks_count) * 100)).toString()
})

const formattedRevenue = computed(() => {
  try {
    return new Intl.NumberFormat('pt-BR', { style: 'currency', currency: props.currency || 'BRL' }).format(props.link.sales_revenue || 0)
  } catch {
    return `R$ ${(props.link.sales_revenue || 0).toFixed(2)}`
  }
})

const activeChannels = computed(() => {
  const list: Array<{ key: string; short: string; label: string; icon: 'meta' | 'tiktok' | 'google_ads' | 'ga4' }> = []
  if (props.link.send_meta) list.push({ key: 'meta', short: 'Meta', label: 'Meta Ads', icon: 'meta' })
  if (props.link.send_tiktok) list.push({ key: 'tiktok', short: 'TikTok', label: 'TikTok Ads', icon: 'tiktok' })
  if (props.link.send_google_ads) list.push({ key: 'gads', short: 'Google', label: 'Google Ads', icon: 'google_ads' })
  if (props.link.send_ga4) list.push({ key: 'ga4', short: 'GA4', label: 'Google Analytics 4', icon: 'ga4' })
  return list
})
</script>

<style scoped>
.link-row { background: var(--surface-2); border: 1px solid var(--border-soft); border-radius: 10px; padding: 0.875rem 1rem; display: flex; flex-direction: column; gap: 0.75rem; transition: border-color 0.15s; }
.link-row:hover { border-color: var(--brand-border); }
.link-row.paused { opacity: 0.65; }
.row-top { display: flex; align-items: flex-start; justify-content: space-between; gap: 0.75rem; flex-wrap: wrap; }
.title-col { display: flex; flex-direction: column; gap: 0.4rem; flex: 1; min-width: 220px; }
.title-line { display: flex; align-items: center; gap: 0.5rem; flex-wrap: wrap; }
.link-name { font-size: var(--fs-lg); font-weight: var(--fw-semibold); color: var(--text-strong); }
.routing-pill { display: inline-flex; align-items: center; gap: 0.35rem; font-size: var(--fs-xs); font-weight: var(--fw-medium); color: var(--text-2); background: var(--surface-soft); border: 1px solid var(--border-faint); padding: 0.125rem 0.5rem; border-radius: 9999px; }
.conn-dot { width: 6px; height: 6px; border-radius: 50%; flex-shrink: 0; }
.dot-connected { background: var(--success); }
.dot-connecting { background: var(--warning); }
.dot-disconnected { background: var(--text-3); }
.url-bar { display: inline-flex; align-items: center; gap: 0.5rem; background: var(--surface-deep); border: 1px solid var(--border-soft); border-radius: 7px; padding: 0.35rem 0.55rem; max-width: 100%; width: fit-content; }
.url-code { font-family: var(--font-mono); font-size: var(--fs-sm); color: var(--brand); word-break: break-all; }
.copy-btn { display: inline-flex; align-items: center; gap: 0.25rem; background: var(--surface-soft); border: 1px solid var(--border); color: var(--text-1); border-radius: 5px; padding: 0.2rem 0.45rem; font-size: var(--fs-xs); font-weight: var(--fw-medium); cursor: pointer; flex-shrink: 0; transition: all 0.15s; }
.copy-btn:hover { border-color: var(--brand-border); color: var(--brand); }
.actions-col { display: flex; align-items: center; gap: 0.25rem; }
.del-btn { display: inline-flex; align-items: center; justify-content: center; width: 28px; height: 28px; border-radius: 6px; border: none; background: transparent; color: var(--text-3); cursor: pointer; transition: all 0.15s; }
.del-btn:hover { background: var(--danger-soft); color: var(--danger); }
.row-bottom { display: flex; align-items: center; justify-content: space-between; gap: 0.75rem; padding-top: 0.625rem; border-top: 1px solid var(--border-faint); flex-wrap: wrap; }
.metrics-strip { display: flex; align-items: center; gap: 0.5rem; flex-wrap: wrap; }
.metric-item { display: inline-flex; align-items: baseline; gap: 0.3rem; }
.metric-label { font-size: var(--fs-xs); color: var(--text-3); text-transform: uppercase; letter-spacing: 0.03em; }
.metric-val { font-size: var(--fs-md); font-weight: var(--fw-bold); color: var(--text-1); }
.metric-val.highlight { color: var(--success); }
.metric-rate { font-size: var(--fs-xs); color: var(--text-3); }
.metric-rev { font-size: var(--fs-xs); font-weight: var(--fw-semibold); color: var(--success); background: var(--success-soft); padding: 0.0625rem 0.375rem; border-radius: 4px; }
.metric-sep { color: var(--text-3); font-size: var(--fs-xs); }
.channels-strip { display: flex; align-items: center; gap: 0.35rem; flex-wrap: wrap; }
.ev-tag { font-size: var(--fs-xs); font-weight: var(--fw-medium); color: var(--brand); background: var(--brand-soft); padding: 0.125rem 0.45rem; border-radius: 5px; }
.ev-tag.muted { color: var(--text-3); background: var(--surface-soft); }
.chan-chip { display: inline-flex; align-items: center; gap: 0.25rem; font-size: var(--fs-xs); color: var(--text-2); background: var(--surface-1); border: 1px solid var(--border-faint); padding: 0.125rem 0.4rem; border-radius: 5px; }
</style>
