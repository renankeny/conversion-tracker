<template>
  <BaseModal
    :model-value="modelValue"
    :title="event ? `${event.event_name} #${event.id}` : ''"
    size="lg"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <div v-if="event" class="event-details">
      <div class="meta-grid">
        <DetailRow label="Pixel" :value="event.pixel_name" />
        <DetailRow label="Evento" :value="event.event_name" />
        <DetailRow label="Event ID" :value="event.event_id" mono />
        <DetailRow label="External ID" :value="event.external_id" mono />
        <DetailRow label="Status">
          <BaseBadge :variant="badgeVariant(event.status)">
            {{ statusLabel(event.status) }}
          </BaseBadge>
        </DetailRow>
        <DetailRow label="HTTP">
          <span class="http" :class="httpClass(event.response_code)">
            {{ event.response_code || '—' }}
          </span>
        </DetailRow>
        <DetailRow label="Criado em" :value="formatTime(event.created_at)" />
        <DetailRow label="Atualizado em" :value="formatTime(event.updated_at)" />
      </div>

      <div v-if="event.error" class="error-block">
        <h4>Erro</h4>
        <pre><code>{{ event.error }}</code></pre>
      </div>

      <div v-if="event.payload" class="payload-block">
        <div class="block-header">
          <h4>Payload enviado para Meta CAPI</h4>
          <button class="copy-btn" type="button" @click="copyPayload">
            {{ copiedPayload ? '✓ Copiado' : 'Copiar' }}
          </button>
        </div>
        <p class="block-hint">
          PII hasheada (SHA-256). Access tokens vão na URL, não no body.
        </p>
        <pre><code>{{ formattedPayload }}</code></pre>
      </div>

      <div v-if="event.response_body" class="response-block">
        <div class="block-header">
          <h4>Resposta da Meta</h4>
          <button class="copy-btn" type="button" @click="copyResponse">
            {{ copiedResponse ? '✓ Copiado' : 'Copiar' }}
          </button>
        </div>
        <pre><code>{{ formattedResponse }}</code></pre>
      </div>
    </div>

    <template #footer>
      <BaseButton variant="ghost" @click="$emit('update:modelValue', false)">
        Fechar
      </BaseButton>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import type { EventStatus, SentEvent } from '#shared/types'

interface Props {
  modelValue: boolean
  event: SentEvent | null
}

const props = defineProps<Props>()

defineEmits<{
  'update:modelValue': [value: boolean]
}>()

const formatTime = (iso: string): string => {
  if (!iso) return ''
  return new Date(iso).toLocaleString('pt-BR', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}

const badgeVariant = (status: EventStatus): 'success' | 'warning' | 'danger' => {
  if (status === 'sent') return 'success'
  if (status === 'pending') return 'warning'
  return 'danger'
}

const statusLabel = (status: EventStatus): string => {
  if (status === 'sent') return 'Enviado'
  if (status === 'pending') return 'Pendente'
  return 'Falhou'
}

const httpClass = (code: number): string => {
  if (!code) return ''
  if (code >= 200 && code < 300) return 'http-ok'
  if (code >= 400) return 'http-err'
  return ''
}

const formattedResponse = computed(() => {
  if (!props.event?.response_body) return ''
  try {
    return JSON.stringify(JSON.parse(props.event.response_body), null, 2)
  } catch {
    return props.event.response_body
  }
})

const formattedPayload = computed(() => {
  if (!props.event?.payload) return ''
  try {
    return JSON.stringify(JSON.parse(props.event.payload), null, 2)
  } catch {
    return props.event.payload
  }
})

const copiedPayload = ref(false)
const copiedResponse = ref(false)

const copyToClipboard = async (text: string): Promise<boolean> => {
  if (!text) return false
  try {
    await navigator.clipboard.writeText(text)
    return true
  } catch {
    return false
  }
}

const copyPayload = async () => {
  if (await copyToClipboard(formattedPayload.value)) {
    copiedPayload.value = true
    setTimeout(() => (copiedPayload.value = false), 2000)
  }
}

const copyResponse = async () => {
  if (await copyToClipboard(formattedResponse.value)) {
    copiedResponse.value = true
    setTimeout(() => (copiedResponse.value = false), 2000)
  }
}
</script>

<script lang="ts">
import { defineComponent, h } from 'vue'

const DetailRow = defineComponent({
  props: {
    label: { type: String, required: true },
    value: { type: String, default: '' },
    mono: { type: Boolean, default: false },
  },
  setup(props, { slots }) {
    return () =>
      h('div', { class: 'detail-row' }, [
        h('span', { class: 'detail-label' }, props.label),
        slots.default
          ? h('div', { class: 'detail-value' }, slots.default())
          : h('span', { class: ['detail-value', { mono: props.mono }] }, props.value || '—'),
      ])
  },
})

export { DetailRow }
</script>

<style scoped>
.event-details {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.meta-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 0.75rem 1.25rem;
}

:deep(.detail-row) {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

:deep(.detail-label) {
  font-size: 0.6875rem;
  font-weight: 600;
  color: var(--text-3);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

:deep(.detail-value) {
  font-size: 0.875rem;
  color: var(--text-1);
  word-break: break-all;
}

:deep(.detail-value.mono) {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  background: var(--surface-soft);
  padding: 0.25rem 0.5rem;
  border-radius: 6px;
  font-size: 0.8125rem;
}

.error-block,
.response-block,
.payload-block {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.error-block h4,
.response-block h4,
.payload-block h4 {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--text-strong);
}

.error-block pre,
.response-block pre,
.payload-block pre {
  background: var(--surface-deep);
  border: 1px solid var(--border-soft);
  border-radius: 8px;
  padding: 0.875rem;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.75rem;
  line-height: 1.55;
  color: var(--text-1);
  overflow-x: auto;
  max-height: 320px;
}

.error-block pre {
  border-color: var(--danger);
  background: var(--danger-soft);
}

.payload-block pre {
  border-color: var(--brand-border);
}

.block-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
}

.block-hint {
  font-size: 0.75rem;
  color: var(--text-3);
  margin-top: -0.25rem;
}

.copy-btn {
  background: var(--surface-soft);
  border: 1px solid var(--border-soft);
  border-radius: 6px;
  padding: 0.25rem 0.625rem;
  font-size: 0.75rem;
  color: var(--text-2);
  cursor: pointer;
  transition: all 0.15s;
}

.copy-btn:hover {
  background: var(--surface-3);
  color: var(--text-1);
}

.http {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-weight: 600;
}

.http-ok {
  color: var(--success);
}

.http-err {
  color: var(--danger);
}
</style>
