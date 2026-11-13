<template>
  <div class="channel-card" :data-state="state" :data-expanded="expanded">
    <button class="card-summary" type="button" @click="$emit('toggle')">
      <div class="card-header">
        <span class="card-icon" :data-channel="channelId">
          <BrandIcon :channel="channelId" size="md" />
        </span>
        <div class="card-title-block">
          <span class="card-label">{{ label }}</span>
          <span class="card-status">{{ statusText }}</span>
        </div>
        <span class="card-code" :data-state="state">
          <span class="dot-indicator" />
          {{ statusCode }}
        </span>
        <svg class="chevron" :class="{ rotated: expanded }" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
          <path fill-rule="evenodd" d="M5.293 7.293a1 1 0 011.414 0L10 10.586l3.293-3.293a1 1 0 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414z" clip-rule="evenodd" />
        </svg>
      </div>
    </button>

    <EventPayloadViewer
      v-if="expanded"
      :label="label"
      :data="data"
    />
  </div>
</template>

<script setup lang="ts">
import BrandIcon from '~/components/events/BrandIcon.vue'
import EventPayloadViewer from '~/components/events/EventPayloadViewer.vue'
import type { Channel, SentEvent } from '#shared/types'

interface Props {
  channelId: Channel
  label: string
  data: SentEvent | null
  expanded: boolean
}

const props = defineProps<Props>()
defineEmits<{ toggle: [] }>()

type State = 'ok' | 'pending' | 'error' | 'absent'

const state = computed<State>(() => {
  if (!props.data) return 'absent'
  if (props.data.error || props.data.status === 'failed') return 'error'
  if (props.data.status === 'queued') return 'pending'
  if (props.data.status === 'sent' && props.data.response_code >= 200 && props.data.response_code < 300) {
    return 'ok'
  }
  return 'error'
})

const statusCode = computed(() => {
  if (!props.data) return '—'
  if (props.data.status === 'queued' && !props.data.response_code) return 'FILA'
  return props.data.response_code || '—'
})

const statusText = computed(() => {
  if (!props.data) return 'Não enviado'
  if (state.value === 'ok') return 'Sucesso'
  if (state.value === 'error') return 'Falhou'
  return 'Pendente'
})
</script>

<style scoped>
.channel-card {
  background: var(--surface-2);
  border: 1px solid var(--border-soft);
  border-radius: 10px;
  overflow: hidden;
  transition: all 0.18s;
}

.channel-card[data-state="ok"] { border-color: var(--brand-border); }
.channel-card[data-state="pending"] { border-color: var(--warning-border); }
.channel-card[data-state="error"] { border-color: var(--danger-border); }
.channel-card[data-state="absent"] { opacity: 0.65; }
.channel-card[data-expanded="true"] { grid-column: 1 / -1; }

.card-summary {
  width: 100%;
  background: transparent;
  border: none;
  padding: 0.75rem 0.875rem;
  cursor: pointer;
  text-align: left;
  transition: background 0.15s;
}

.card-summary:hover { background: var(--surface-soft); }

.card-header {
  display: flex;
  align-items: center;
  gap: 0.625rem;
}

.card-icon {
  flex-shrink: 0;
  width: 32px;
  height: 32px;
  border-radius: 8px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: var(--surface-qr);
  border: 1px solid var(--border-soft);
}

[data-theme="light"] .card-icon { background: var(--surface-1); }

.card-title-block {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
  flex: 1;
  min-width: 0;
}

.card-label {
  font-size: var(--fs-md);
  font-weight: var(--fw-semibold);
  color: var(--text-1);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.card-status { font-size: var(--fs-sm); color: var(--text-3); }

.card-code {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: var(--fs-base);
  font-weight: var(--fw-bold);
  padding: 0.1875rem 0.5rem;
  border-radius: 6px;
  background: var(--surface-soft);
  color: var(--text-2);
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
}

.dot-indicator {
  width: 6px;
  height: 6px;
  border-radius: 9999px;
  background: currentColor;
  flex-shrink: 0;
}

.card-code[data-state="ok"] {
  background: var(--success-soft);
  color: var(--status-ok);
  border: 1px solid var(--brand-border);
}

.card-code[data-state="pending"] {
  background: var(--warning-soft);
  color: var(--warning);
  border: 1px solid var(--warning-border);
}

.card-code[data-state="error"] {
  background: var(--danger-soft);
  color: var(--danger-light);
  border: 1px solid var(--danger-border);
}

.chevron {
  width: 18px;
  height: 18px;
  color: var(--text-3);
  transition: transform 0.2s;
}

.chevron.rotated { transform: rotate(180deg); }
</style>
