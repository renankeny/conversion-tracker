<template>
  <div class="webhooks-page">
    <div class="page-header">
      <div>
        <h1 class="page-title">Webhooks</h1>
        <p class="page-subtitle">
          Histórico de webhooks recebidos dos gateways de pagamento.
          Replay disponível para reprocessar eventos elegíveis.
        </p>
      </div>
      <BaseButton variant="outline" size="sm" :loading="loading" @click="refresh">
        Atualizar
      </BaseButton>
    </div>

    <div class="filters-row">
      <div class="filter-dropdown">
        <BaseDropdown
          :model-value="filterSource"
          placeholder="Todos os gateways"
          :options="sourceOptions"
          @update:model-value="filterSource = String($event ?? '')"
        />
      </div>

      <div class="filter-dropdown">
        <BaseDropdown
          :model-value="filterStatus"
          placeholder="Todos os status"
          :options="statusOptions"
          @update:model-value="filterStatus = String($event ?? '')"
        />
      </div>
    </div>

    <div class="table-card">
      <div v-if="loading && !logs.length" class="state-row">
        <div class="spinner" />
        <p>Carregando...</p>
      </div>

      <div v-else-if="!logs.length" class="state-row">
        <p class="empty-msg">Nenhum webhook encontrado.</p>
      </div>

      <div v-else class="table-wrap">
        <table class="table">
          <thead>
            <tr>
              <th>Recebido</th>
              <th>Gateway</th>
              <th>Pixel</th>
              <th>External ID</th>
              <th>Evento</th>
              <th>Status / Detalhe</th>
              <th v-if="canManagePixels"></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="log in logs" :key="log.id" class="row">
              <td class="time">{{ formatDate(log.received_at) }}</td>
              <td><span class="source-badge">{{ log.source }}</span></td>
              <td class="pixel">{{ log.pixel_name || '-' }}</td>
              <td><code class="ext-id">{{ truncate(log.external_id, 12) || '-' }}</code></td>
              <td><span class="event-name">{{ log.event_name }}</span></td>
              <td>
                <div class="status-cell">
                  <BaseBadge :variant="statusVariant(log.status)">
                    {{ statusLabel(log.status) }}
                  </BaseBadge>
                  <span v-if="log.error" class="error-detail">{{ log.error }}</span>
                </div>
              </td>
              <td v-if="canManagePixels" class="action-cell">
                <BaseButton
                  variant="ghost"
                  size="sm"
                  :loading="replayingId === log.id"
                  :title="canReplayLog(log) ? 'Reprocessar webhook' : 'Replay indisponível para este registro'"
                  :disabled="!canReplayLog(log)"
                  @click="handleReplay(log)"
                >
                  <template #iconLeft><RefreshCw :size="14" /></template>
                  Replay
                </BaseButton>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <div v-if="total > 0" class="pagination">
      <span class="pag-info">Mostrando {{ offsetFrom }} a {{ offsetTo }} de {{ total }}</span>
      <div class="pag-buttons">
        <BaseButton variant="ghost" size="sm" :disabled="offset === 0" @click="prevPage">Anterior</BaseButton>
        <BaseButton variant="ghost" size="sm" :disabled="offsetTo >= total" @click="nextPage">Próximo</BaseButton>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { RefreshCw } from 'lucide-vue-next'
import { PLATFORMS } from '#shared/constants/platforms'
import { useAuth } from '~/composables/useAuth'
import { useTenantStore } from '~/stores/useTenantStore'
import type { WebhookLog } from '#shared/types'
import { formatShortDateTimeSP } from '~/utils/formatDate'
import { errorMessage } from '~/utils/errorMessage'

definePageMeta({ layout: 'app', middleware: 'auth' })

const { formatTitle } = useBranding()
useHead({ title: computed(() => formatTitle('Webhooks')) })

const route = useRoute()
const { canManagePixels } = useAuth()
const webhooksApi = useWebhookLogs()
const toast = useToast()
const tenantStore = useTenantStore()

const PAGE_SIZE = 50
const logs = ref<WebhookLog[]>([])
const total = ref(0)
const offset = ref(0)
const loading = ref(false)
const replayingId = ref<number | null>(null)
const initialSource = typeof route.query.source === 'string' ? route.query.source.trim().toLowerCase() : ''
const filterSource = ref(PLATFORMS.some((p) => p.id === initialSource) ? initialSource : '')
const filterStatus = ref('')

const sourceOptions = [
  { value: '', label: 'Todos os gateways' },
  ...PLATFORMS.map((p) => ({ value: p.id, label: p.name })),
]

const statusOptions = [
  { value: '', label: 'Todos os status' },
  { value: 'queued', label: 'Enfileirado' },
  { value: 'buffered', label: 'Em buffer' },
  { value: 'failed', label: 'Falhou' },
  { value: 'replayed', label: 'Reprocessado' },
]

const offsetFrom = computed(() => (logs.value.length ? offset.value + 1 : 0))
const offsetTo = computed(() => offset.value + logs.value.length)

const fetchLogs = async () => {
  loading.value = true
  try {
    const r = await webhooksApi.list({
      source: filterSource.value || undefined,
      status: filterStatus.value || undefined,
      limit: PAGE_SIZE,
      offset: offset.value,
    })
    logs.value = r.data || []
    total.value = r.total || 0
  } catch (e: unknown) {
    toast.error('Erro', errorMessage(e, 'Não foi possível carregar'))
  } finally {
    loading.value = false
  }
}

const refresh = () => { offset.value = 0; fetchLogs() }
const prevPage = () => { offset.value = Math.max(0, offset.value - PAGE_SIZE); fetchLogs() }
const nextPage = () => { offset.value += PAGE_SIZE; fetchLogs() }

watch([filterSource, filterStatus], () => { offset.value = 0; fetchLogs() })

watch(() => tenantStore.switchVersion, () => {
  offset.value = 0
  fetchLogs()
})

const canReplayLog = (log: WebhookLog): boolean => {
  if (typeof log.can_replay === 'boolean') return log.can_replay
  return Boolean(log.event_id)
}

const handleReplay = async (log: WebhookLog) => {
  replayingId.value = log.id
  try {
    const r = await webhooksApi.replay(log.id)
    if (r.event_id) toast.success('Reprocessado', `Evento ${r.event_id as string} enfileirado.`)
    else if (r.error) toast.error('Falhou', r.error as string)
    else toast.warning('Aviso', 'Usuário não encontrado. Cadastre o lead e tente novamente.')
    await fetchLogs()
  } catch (e: unknown) {
    toast.error('Erro', errorMessage(e, 'Replay falhou'))
  } finally {
    replayingId.value = null
  }
}

const statusVariant = (s: string) => {
  if (s === 'queued') return 'success'
  if (s === 'buffered') return 'warning'
  if (s === 'failed') return 'danger'
  return 'info'
}

const statusLabel = (s: string) => {
  if (s === 'queued') return 'Enfileirado'
  if (s === 'buffered') return 'Em buffer'
  if (s === 'failed') return 'Falhou'
  return 'Reprocessado'
}

const truncate = (s: string, n: number) => (s && s.length > n ? s.slice(0, n) + '...' : s || '')
const formatDate = (iso: string) => formatShortDateTimeSP(iso)

onMounted(fetchLogs)
</script>

<style scoped>
.webhooks-page { display: flex; flex-direction: column; gap: 1.25rem; }
.page-header { display: flex; justify-content: space-between; align-items: flex-start; gap: 1rem; flex-wrap: wrap; }
.page-title { font-family: var(--font-sans); font-size: var(--fs-3xl); font-weight: var(--fw-bold); color: var(--text-strong); margin-bottom: 0.25rem; }
.page-subtitle { color: var(--text-2); font-size: var(--fs-lg); }
.filters-row { display: flex; gap: 0.75rem; flex-wrap: wrap; }
.filter-dropdown { min-width: 190px; }
.table-card { background: var(--surface-2); border: 1px solid var(--border); border-radius: 12px; overflow: hidden; box-shadow: var(--shadow-card); }
.table-wrap { overflow-x: auto; }
.table { width: 100%; border-collapse: collapse; min-width: 720px; }
.table th { background: var(--surface-soft-2); color: var(--text-3); font-size: var(--fs-xs); font-weight: var(--fw-semibold); text-transform: uppercase; letter-spacing: 0.05em; text-align: left; padding: 0.625rem 1rem; border-bottom: 1px solid var(--border); }
.table td { padding: 0.75rem 1rem; font-size: var(--fs-md); color: var(--text-1); border-bottom: 1px solid var(--border-soft); vertical-align: middle; }
.row:last-child td { border-bottom: none; }
.row:hover td { background: var(--surface-soft); }
.time { font-family: var(--font-mono); font-size: var(--fs-sm); color: var(--text-2); white-space: nowrap; }
.source-badge { background: var(--brand-soft); color: var(--brand); padding: 0.125rem 0.5rem; border-radius: 5px; font-size: var(--fs-sm); font-weight: var(--fw-semibold); }
.pixel { font-weight: var(--fw-semibold); }
.ext-id { font-family: var(--font-mono); font-size: var(--fs-sm); background: var(--surface-soft); padding: 0.0625rem 0.375rem; border-radius: 4px; color: var(--text-2); }
.event-name { font-size: var(--fs-base); font-weight: var(--fw-semibold); color: var(--success); }
.status-cell { display: flex; flex-direction: column; align-items: flex-start; gap: 0.25rem; }
.error-detail { font-size: var(--fs-xs); color: var(--text-3); max-width: 280px; line-height: 1.3; }
.action-cell { width: 100px; }
.state-row { display: flex; flex-direction: column; align-items: center; justify-content: center; padding: 3rem; gap: 0.75rem; color: var(--text-3); }
.empty-msg { font-size: var(--fs-lg); }
.spinner { width: 24px; height: 24px; border: 2px solid var(--border); border-top-color: var(--brand); border-radius: 50%; animation: spin 0.7s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
.pagination { display: flex; justify-content: space-between; align-items: center; gap: 1rem; padding: 0.75rem 0.25rem; }
.pag-info { font-size: var(--fs-base); color: var(--text-3); }
.pag-buttons { display: flex; gap: 0.375rem; }
</style>
