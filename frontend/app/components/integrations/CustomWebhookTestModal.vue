<template>
  <BaseModal
    :model-value="modelValue"
    title="Logs de Auditoria"
    :description="webhookName"
    size="xl"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <div v-if="loading" class="logs-loading">Carregando logs...</div>
    <div v-else-if="logs.length === 0" class="logs-empty">
      <p>Nenhum webhook recebido ainda.</p>
      <p class="logs-hint">Assim que alguém fizer uma compra na plataforma, o payload aparecerá aqui para auditoria.</p>
    </div>
    <div v-else class="logs-list">
      <div v-for="log in logs" :key="log.id" class="log-entry" :class="log.status">
        <div class="log-entry-header" @click="toggleLog(log.id)">
          <div class="log-meta">
            <span class="log-status-badge" :class="log.status">{{ log.status }}</span>
            <span class="log-event">{{ log.event_name }}</span>
            <span v-if="log.external_id" class="log-email">{{ log.external_id }}</span>
          </div>
          <div class="log-time-chevron">
            <span class="log-time">{{ formatDate(log.received_at) }}</span>
            <span class="log-chevron">{{ expandedLogs.has(log.id) ? '▲' : '▼' }}</span>
          </div>
        </div>
        <div v-if="expandedLogs.has(log.id)" class="log-entry-detail">
          <div class="log-panels">
            <div class="log-panel">
              <div class="log-panel-title">📥 Payload Recebido</div>
              <pre class="log-json">{{ JSON.stringify(log.raw_payload, null, 2) }}</pre>
            </div>
            <div class="log-panel">
              <div class="log-panel-title">✅ Dados Processados</div>
              <pre class="log-json">{{ JSON.stringify(log.processed_input, null, 2) }}</pre>
            </div>
          </div>
          <div v-if="log.error" class="log-error">❌ Erro: {{ log.error }}</div>
        </div>
      </div>
    </div>
    <template #footer>
      <BaseButton variant="ghost" @click="$emit('update:modelValue', false)">Fechar</BaseButton>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import { formatDateTimeSP } from '~/utils/formatDate'

interface CustomWebhookLog {
  id: number
  status: string
  event_name: string
  external_id?: string
  received_at: string
  raw_payload: unknown
  processed_input: unknown
  error?: string
}

const props = defineProps<{
  modelValue: boolean
  webhookId: string
  webhookName: string
}>()

defineEmits<{
  'update:modelValue': [value: boolean]
}>()

const toast = useToast()
const loading = ref(false)
const logs = ref<CustomWebhookLog[]>([])
const expandedLogs = ref(new Set<number>())

const fetchLogs = async (id: string) => {
  if (!id) return
  loading.value = true
  expandedLogs.value = new Set()
  logs.value = []
  try {
    const api = useAPI()
    const data = await api.get<CustomWebhookLog[]>(`/admin/custom-webhooks/${id}/logs`)
    logs.value = data || []
  } catch {
    toast.error('Erro', 'Não foi possível carregar os logs.')
  } finally {
    loading.value = false
  }
}

watch(
  () => [props.modelValue, props.webhookId] as const,
  ([open, id]) => {
    if (open && id) fetchLogs(id)
  },
  { immediate: true }
)

const toggleLog = (id: number) => {
  if (expandedLogs.value.has(id)) expandedLogs.value.delete(id)
  else expandedLogs.value.add(id)
  expandedLogs.value = new Set(expandedLogs.value)
}

const formatDate = (d: string) =>
  formatDateTimeSP(d, { year: '2-digit', second: '2-digit' })
</script>

<style scoped>
.logs-loading, .logs-empty { padding: 3rem; text-align: center; color: var(--text-3); }
.logs-hint { font-size: var(--fs-base); margin-top: 0.5rem; }
.logs-list { padding: 1rem 1.5rem 1.5rem; display: flex; flex-direction: column; gap: 0.5rem; overflow-y: auto; max-height: 65vh; }
.log-entry { background: var(--surface-2); border-radius: 8px; border: 1px solid var(--border); overflow: hidden; }
.log-entry.queued { border-left: 3px solid var(--brand); }
.log-entry.failed { border-left: 3px solid var(--danger); }
.log-entry-header { display: flex; align-items: center; justify-content: space-between; padding: 0.75rem 1rem; cursor: pointer; user-select: none; }
.log-entry-header:hover { background: var(--surface-soft); }
.log-meta { display: flex; align-items: center; gap: 0.75rem; flex-wrap: wrap; }
.log-status-badge { font-size: var(--fs-xs); font-weight: var(--fw-bold); text-transform: uppercase; padding: 0.2rem 0.5rem; border-radius: 99px; }
.log-status-badge.queued { background: var(--brand-soft); color: var(--brand); }
.log-status-badge.failed { background: var(--danger-soft); color: var(--danger); }
.log-event { font-weight: var(--fw-semibold); font-size: var(--fs-md); color: var(--text-1); }
.log-email { font-size: var(--fs-base); color: var(--text-3); }
.log-time-chevron { display: flex; align-items: center; gap: 0.75rem; flex-shrink: 0; }
.log-time { font-size: var(--fs-sm); color: var(--text-3); white-space: nowrap; }
.log-chevron { font-size: var(--fs-xs); color: var(--text-3); }
.log-entry-detail { padding: 1rem; border-top: 1px solid var(--border-soft); }
.log-panels { display: grid; grid-template-columns: 1fr 1fr; gap: 1rem; }
@media (max-width: 767px) { .log-panels { grid-template-columns: 1fr; } }
.log-panel { background: var(--surface-1); border-radius: 8px; overflow: hidden; border: 1px solid var(--border-soft); }
.log-panel-title { padding: 0.5rem 0.75rem; font-size: var(--fs-sm); font-weight: var(--fw-bold); color: var(--text-2); background: var(--surface-3); border-bottom: 1px solid var(--border-soft); }
.log-json { padding: 0.75rem; font-family: var(--font-mono); font-size: var(--fs-sm); color: var(--brand); white-space: pre; overflow-x: auto; max-height: 300px; overflow-y: auto; margin: 0; }
.log-error { margin-top: 0.75rem; padding: 0.5rem 0.75rem; background: var(--danger-soft); color: var(--danger); border-radius: 6px; font-size: var(--fs-base); }
</style>
