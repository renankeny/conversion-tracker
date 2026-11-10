<template>
  <div class="payload-viewer">
    <div v-if="!data" class="empty-card">
      <p>Esse evento não foi enviado para {{ label }}.</p>
      <p class="empty-hint">
        Configure as credenciais do canal nas configurações do pixel para
        que próximos eventos sejam espelhados.
      </p>
    </div>

    <template v-else>
      <div v-if="data.error" class="block error">
        <h5>Erro</h5>
        <pre><code>{{ data.error }}</code></pre>
      </div>

      <div v-if="data.payload" class="block">
        <div class="block-header">
          <h5>Request enviado</h5>
          <button class="copy-btn" type="button" @click="copy(formattedPayload, 'payload')">
            {{ copied === 'payload' ? '✓ Copiado' : 'Copiar' }}
          </button>
        </div>
        <pre><code>{{ formattedPayload }}</code></pre>
      </div>

      <div v-if="data.response_body" class="block">
        <div class="block-header">
          <h5>Resposta da plataforma</h5>
          <button class="copy-btn" type="button" @click="copy(formattedResponse, 'response')">
            {{ copied === 'response' ? '✓ Copiado' : 'Copiar' }}
          </button>
        </div>
        <pre><code>{{ formattedResponse }}</code></pre>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import type { SentEvent } from '#shared/types'

const props = defineProps<{
  label: string
  data: SentEvent | null
}>()

const formatJson = (raw?: string) => {
  if (!raw) return ''
  try {
    return JSON.stringify(JSON.parse(raw), null, 2)
  } catch {
    return raw
  }
}

const formattedPayload = computed(() => formatJson(props.data?.payload))
const formattedResponse = computed(() => formatJson(props.data?.response_body))

const copied = ref<'payload' | 'response' | null>(null)
const copy = async (text: string, kind: 'payload' | 'response') => {
  if (!text) return
  try {
    await navigator.clipboard.writeText(text)
    copied.value = kind
    setTimeout(() => (copied.value = null), 1800)
  } catch {
    /* ignore */
  }
}
</script>

<style scoped>
.payload-viewer {
  padding: 0.875rem;
  border-top: 1px solid var(--border-soft);
  display: flex;
  flex-direction: column;
  gap: 0.875rem;
  background: var(--surface-soft-2);
}

.empty-card {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
  color: var(--text-3);
  font-size: var(--fs-base);
}

.empty-hint {
  font-size: var(--fs-sm);
  font-style: italic;
}

.block {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
}

.block h5 {
  font-size: var(--fs-sm);
  font-weight: var(--fw-semibold);
  color: var(--text-2);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.block-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
}

.copy-btn {
  background: var(--surface-soft);
  border: 1px solid var(--border-soft);
  border-radius: 5px;
  padding: 0.1875rem 0.5rem;
  font-size: var(--fs-xs);
  color: var(--text-2);
  cursor: pointer;
  transition: all 0.15s;
}

.copy-btn:hover {
  background: var(--surface-3);
  color: var(--text-1);
}

.block pre {
  background: var(--surface-deep);
  border: 1px solid var(--border-soft);
  border-radius: 6px;
  padding: 0.75rem;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: var(--fs-xs);
  line-height: 1.55;
  color: var(--text-1);
  overflow-x: auto;
  max-height: 280px;
}

.block.error pre {
  border-color: var(--danger);
  background: var(--danger-soft);
}
</style>
