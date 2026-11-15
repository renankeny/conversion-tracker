<template>
  <div v-if="event" class="modal-event-header">
    <div class="header-top-row">
      <span class="category-pill" :class="eventCategoryClass">{{ eventCategoryLabel }}</span>
      <h2 class="event-title">{{ event.event_name }}</h2>
      <span class="pixel-pill" :title="`Pixel: ${event.pixel_name}`">
        <span class="pixel-dot" />
        {{ event.pixel_name }}
      </span>
    </div>

    <div class="header-meta-row">
      <span class="meta-item">
        <Clock class="w-3.5 h-3.5 opacity-60" />
        {{ formatTime(event.created_at) }}
      </span>

      <span class="meta-separator">•</span>

      <span class="meta-item" :style="{ color: statusColor }">
        <span class="status-dot" :class="statusDotClass" />
        {{ statusLabel }}
      </span>

      <template v-if="event.event_id">
        <span class="meta-separator">•</span>
        <span class="meta-item event-id-chip">
          <span class="id-prefix">ID:</span>
          <code class="id-code" :title="event.event_id">{{ truncateId(event.event_id) }}</code>
          <button
            type="button"
            class="btn-copy-id"
            :class="{ copied: copiedId }"
            :title="copiedId ? 'ID copiado!' : 'Copiar ID do evento'"
            @click="copyEventId"
          >
            <Check v-if="copiedId" class="w-3 h-3" />
            <Copy v-else class="w-3 h-3" />
          </button>
        </span>
      </template>
    </div>
  </div>

  <div v-else class="header-top-row">
    <h2 class="event-title">Detalhes do Evento</h2>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { Check, Clock, Copy } from 'lucide-vue-next'
import type { EventDetail } from '#shared/types'
import { formatDateTimeSP } from '~/utils/formatDate'

const props = defineProps<{
  event: EventDetail | null
}>()

const copiedId = ref(false)

const externalChannels = computed(() =>
  (props.event?.channels || []).filter((c) => c.channel !== 'internal'),
)

const isInternalOnly = computed(() =>
  Boolean(props.event?.channels.length) && externalChannels.value.length === 0,
)

const successfulChannelsCount = computed(() =>
  externalChannels.value.filter(
    (c) => c.status === 'sent' && c.response_code >= 200 && c.response_code < 300,
  ).length,
)

const hasFailedChannels = computed(() =>
  externalChannels.value.some(
    (c) => c.status === 'failed' || (c.response_code >= 400 && c.response_code !== 0) || Boolean(c.error),
  ),
)

const statusLabel = computed(() => {
  if (!props.event) return 'Aguardando'
  if (isInternalOnly.value) return 'Registrado Internamente (sem canais externos ativos)'
  if (externalChannels.value.length === 0) return 'Em Processamento'
  const suffix = `(${successfulChannelsCount.value} de ${externalChannels.value.length} canais)`
  if (hasFailedChannels.value) return `Falha na Transmissão ${suffix}`
  if (successfulChannelsCount.value > 0) return `Transmissão Completa ${suffix}`
  return `Em Processamento ${suffix}`
})

const statusDotClass = computed(() => {
  if (isInternalOnly.value) return 'dot-info'
  if (externalChannels.value.length === 0) return 'dot-warning'
  if (hasFailedChannels.value) return 'dot-error'
  return successfulChannelsCount.value > 0 ? 'dot-success' : 'dot-warning'
})

const statusColor = computed(() => {
  if (isInternalOnly.value) return 'var(--text-2)'
  if (externalChannels.value.length === 0) return 'var(--warning)'
  if (hasFailedChannels.value) return 'var(--danger)'
  return successfulChannelsCount.value > 0 ? 'var(--brand)' : 'var(--warning)'
})

const CATEGORY_MAP: Record<string, { label: string; cls: string }> = {
  Purchase: { label: 'Conversão', cls: 'cat-purchase' },
  InitiateCheckout: { label: 'Checkout', cls: 'cat-checkout' },
  CompleteRegistration: { label: 'Cadastro', cls: 'cat-reg' },
  Lead: { label: 'Lead', cls: 'cat-lead' },
  QualifiedLead: { label: 'Lead', cls: 'cat-lead' },
  Contact: { label: 'Contato', cls: 'cat-contact' },
  LeadSubmitted: { label: 'Contato', cls: 'cat-contact' },
}

const eventCategoryLabel = computed(() => CATEGORY_MAP[props.event?.event_name || '']?.label || 'Evento')
const eventCategoryClass = computed(() => CATEGORY_MAP[props.event?.event_name || '']?.cls || 'cat-default')

const formatTime = (iso: string): string => formatDateTimeSP(iso, true)
const truncateId = (id: string): string => (id.length > 20 ? `${id.slice(0, 18)}…` : id)

const copyEventId = async () => {
  if (!props.event?.event_id) return
  try {
    await navigator.clipboard.writeText(props.event.event_id)
    copiedId.value = true
    setTimeout(() => { copiedId.value = false }, 2000)
  } catch { /* silent */ }
}
</script>

<style scoped>
.modal-event-header { display: flex; flex-direction: column; gap: 0.25rem; }
.header-top-row { display: flex; align-items: center; flex-wrap: wrap; gap: 0.5rem; }
.event-title { font-size: var(--fs-xl); font-weight: var(--fw-bold); color: var(--text-1); letter-spacing: -0.01em; margin: 0; }
.header-meta-row { display: flex; align-items: center; flex-wrap: wrap; gap: 0.5rem; font-size: var(--fs-sm); color: var(--text-3); }
.meta-item { display: inline-flex; align-items: center; gap: 0.35rem; }
.meta-separator { opacity: 0.3; }
.status-dot { width: 6px; height: 6px; border-radius: 9999px; display: inline-block; flex-shrink: 0; }
.status-dot.dot-success { background: var(--success); box-shadow: 0 0 6px var(--brand-border); }
.status-dot.dot-error { background: var(--danger); box-shadow: 0 0 6px var(--danger-border); }
.status-dot.dot-warning { background: var(--warning); box-shadow: 0 0 6px var(--warning-border); }
.status-dot.dot-info { background: var(--accent-sky); }

.event-id-chip { gap: 0.25rem; color: var(--text-3); }
.id-prefix { font-weight: var(--fw-semibold); }
.id-code { font-family: var(--font-mono); font-size: var(--fs-xs); color: var(--text-2); background: var(--surface-deep); padding: 0.0625rem 0.375rem; border-radius: 4px; border: 1px solid var(--border-soft); }
.btn-copy-id { display: inline-flex; align-items: center; justify-content: center; padding: 0.15rem; border-radius: 4px; border: 1px solid transparent; background: transparent; color: var(--text-3); cursor: pointer; transition: all 0.15s ease; }
.btn-copy-id:hover { color: var(--brand); background: var(--surface-soft); border-color: var(--border-soft); }
.btn-copy-id.copied { color: var(--success); }

.category-pill { font-size: var(--fs-2xs); font-weight: var(--fw-bold); text-transform: uppercase; letter-spacing: 0.05em; padding: 0.125rem 0.5rem; border-radius: 9999px; }
.category-pill.cat-purchase { background: var(--brand-soft); color: var(--brand); border: 1px solid var(--brand-border); }
.category-pill.cat-checkout { background: var(--warning-soft); color: var(--warning-light); border: 1px solid var(--warning-border); }
.category-pill.cat-reg { background: var(--accent-sky-soft); color: var(--accent-sky); border: 1px solid var(--accent-sky-border); }
.category-pill.cat-lead { background: var(--accent-purple-soft); color: var(--accent-purple); border: 1px solid var(--accent-purple-border); }
.category-pill.cat-contact { background: var(--brand-soft); color: var(--brand); border: 1px solid var(--brand-border); }
.category-pill.cat-default { background: var(--surface-soft); color: var(--text-2); border: 1px solid var(--border); }

.pixel-pill { display: inline-flex; align-items: center; gap: 0.375rem; font-size: var(--fs-sm); font-weight: var(--fw-medium); color: var(--text-2); background: var(--surface-deep); padding: 0.125rem 0.5rem; border-radius: 6px; border: 1px solid var(--border-soft); }
.pixel-dot { width: 6px; height: 6px; border-radius: 9999px; background: var(--brand); }
</style>
