<template>
  <div class="events-page">
    <div class="page-header">
      <div>
        <h1 class="page-title">Eventos</h1>
        <p class="page-subtitle">
          Histórico completo de todos os eventos. (PAGEVIEW E VIEWCONTENT NÃO APARECEM)
        </p>
      </div>
      <div class="header-actions">
        <Transition name="fade">
          <div v-if="selectedIds.length > 0" class="selection-bar">
            <span class="selection-count">{{ selectedIds.length }} selecionado{{ selectedIds.length > 1 ? 's' : '' }}</span>
            <BaseButton variant="danger" size="sm" :loading="deleting" @click="confirmDelete">
              <template #iconLeft>
                <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
                  <path fill-rule="evenodd" d="M9 2a1 1 0 00-.894.553L7.382 4H4a1 1 0 000 2v10a2 2 0 002 2h8a2 2 0 002-2V6a1 1 0 100-2h-3.382l-.724-1.447A1 1 0 0011 2H9zM7 8a1 1 0 012 0v6a1 1 0 11-2 0V8zm5-1a1 1 0 00-1 1v6a1 1 0 102 0V8a1 1 0 00-1-1z" clip-rule="evenodd" />
                </svg>
              </template>
              Deletar selecionados
            </BaseButton>
            <BaseButton variant="ghost" size="sm" @click="clearSelection">
              Cancelar
            </BaseButton>
          </div>
        </Transition>

        <BaseButton variant="outline" size="sm" :loading="loading" @click="refresh">
          Atualizar
        </BaseButton>
      </div>
    </div>

    <EventsFilters v-model="filters" />

    <EventsTable
      :events="events"
      :loading="loading"
      @select="onSelect"
      @selection-change="onSelectionChange"
      @retry="onRetry"
    />

    <div v-if="total > 0" class="pagination">
      <span class="pag-info">
        Mostrando {{ offsetFrom }}-{{ offsetTo }} de {{ formatNumber(total) }}
      </span>
      <div class="pag-buttons">
        <BaseButton variant="ghost" size="sm" :disabled="offset === 0" @click="prevPage">
          ← Anterior
        </BaseButton>
        <BaseButton variant="ghost" size="sm" :disabled="offsetTo >= total" @click="nextPage">
          Próximo →
        </BaseButton>
      </div>
    </div>

    <EventDetailsModal v-model="modalOpen" :source="selected" />

    <ConfirmModal
      v-model="deleteModalOpen"
      title="Deletar eventos?"
      :message="`Isso vai remover permanentemente ${selectedIds.length} evento${selectedIds.length > 1 ? 's' : ''} (todos os canais). Não é possível desfazer.`"
      confirm-label="Deletar"
      variant="danger"
      @confirm="doDelete"
    />
  </div>
</template>

<script setup lang="ts">
import EventsFilters from '~/components/events/EventsFilters.vue'
import EventsTable from '~/components/events/EventsTable.vue'
import EventDetailsModal from '~/components/events/EventDetailsModal.vue'
import ConfirmModal from '~/components/ConfirmModal.vue'
import { useLiveEventsStore } from '~/stores/useLiveEventsStore'
import { useTenantStore } from '~/stores/useTenantStore'
import type { EventFilters, GroupedEvent } from '#shared/types'

definePageMeta({ layout: 'app', middleware: 'auth' })
const { formatTitle } = useBranding()
useHead({ title: computed(() => formatTitle('Eventos')) })

const eventsApi = useEvents()
const accountsApi = useAccounts()
const toast = useToast()
const liveStore = useLiveEventsStore()
const tenantStore = useTenantStore()

const PAGE_SIZE = 50
const filters = ref<EventFilters>({})
const eventsFilterMap = ref<Record<string, string[]>>({})
let eventsFilterLoaded = false

const events = ref<GroupedEvent[]>([])
const total = ref(0)
const offset = ref(0)
const loading = ref(false)

const modalOpen = ref(false)
const selected = ref<GroupedEvent | null>(null)

const selectedIds = ref<string[]>([])
const deleteModalOpen = ref(false)
const deleting = ref(false)

const offsetFrom = computed(() => (events.value.length ? offset.value + 1 : 0))
const offsetTo = computed(() => offset.value + events.value.length)

const fetchEvents = async () => {
  loading.value = true
  try {
    const response = await eventsApi.listGrouped({
      ...filters.value,
      limit: PAGE_SIZE,
      offset: offset.value,
      include_user_context: true,
    })
    events.value = response.data || []
    total.value = response.total || 0
  } catch (error: any) {
    toast.error('Erro', error.message || 'Não foi possível carregar os eventos')
  } finally {
    loading.value = false
  }
}

const refresh = () => { offset.value = 0; fetchEvents() }
const prevPage = () => { offset.value = Math.max(0, offset.value - PAGE_SIZE); fetchEvents() }
const nextPage = () => { offset.value += PAGE_SIZE; fetchEvents() }

const onSelect = (ev: GroupedEvent) => { selected.value = ev; modalOpen.value = true }
const onSelectionChange = (ids: string[]) => { selectedIds.value = ids }
const clearSelection = () => { selectedIds.value = [] }
const confirmDelete = () => { deleteModalOpen.value = true }

const onRetry = async (eventIds: string[]) => {
  try {
    const r = await eventsApi.retryByEventIDs(eventIds)
    toast.success('Reenfileirado', `${r.queued} evento${r.queued !== 1 ? 's' : ''} adicionado${r.queued !== 1 ? 's' : ''} à fila.`)
    await fetchEvents()
  } catch (error: any) {
    toast.error('Erro', error.message || 'Não foi possível reenfileirar')
  }
}

const doDelete = async () => {
  deleteModalOpen.value = false
  deleting.value = true
  try {
    const { deleted } = await eventsApi.deleteByEventIDs(selectedIds.value)
    toast.success('Eventos deletados', `${deleted} registro${deleted !== 1 ? 's' : ''} removido${deleted !== 1 ? 's' : ''}.`)
    selectedIds.value = []
    offset.value = 0
    await fetchEvents()
  } catch (error: any) {
    toast.error('Erro', error.message || 'Não foi possível deletar os eventos')
  } finally {
    deleting.value = false
  }
}

let isRestoringEventFilter = false

watch(filters, () => {
  if (isRestoringEventFilter) return
  offset.value = 0
  fetchEvents()
}, { deep: true })

watch(() => filters.value.pixel, (pixel) => {
  if (!eventsFilterLoaded) return
  const key = pixel || ''
  const saved = (eventsFilterMap.value[key] || []).join(',')
  if ((filters.value.event || '') !== saved) {
    isRestoringEventFilter = true
    filters.value = { ...filters.value, event: saved || undefined }
    isRestoringEventFilter = false
    offset.value = 0
    fetchEvents()
  }
})

watch(() => filters.value.event, (eventCSV) => {
  if (!eventsFilterLoaded) return
  const key = filters.value.pixel || ''
  const arr = (eventCSV || '').split(',').map((s) => s.trim()).filter(Boolean)
  eventsFilterMap.value = { ...eventsFilterMap.value, [key]: arr }
  accountsApi.updateEventsFilter(key, arr).catch(() => {})
})

const formatNumber = (n: number): string => new Intl.NumberFormat('pt-BR').format(n)

let liveRefreshTimer: ReturnType<typeof setTimeout> | null = null
watch(() => liveStore.lastEventSeq, () => {
  if (offset.value > 0 || selectedIds.value.length > 0 || modalOpen.value) return
  if (liveRefreshTimer) clearTimeout(liveRefreshTimer)
  liveRefreshTimer = setTimeout(fetchEvents, 2000)
})

const loadAccountFilterAndEvents = async () => {
  eventsFilterLoaded = false
  try {
    const res = await accountsApi.getEventsFilter()
    eventsFilterMap.value = res.events_filter || {}
  } catch {
    eventsFilterMap.value = {}
  }
  eventsFilterLoaded = true
  const saved = (eventsFilterMap.value[filters.value.pixel || ''] || []).join(',')
  if (saved && saved !== (filters.value.event || '')) {
    filters.value = { ...filters.value, event: saved }
  } else {
    fetchEvents()
  }
}

watch(() => tenantStore.switchVersion, () => {
  modalOpen.value = false
  selected.value = null
  selectedIds.value = []
  offset.value = 0
  isRestoringEventFilter = true
  filters.value = {}
  isRestoringEventFilter = false
  void loadAccountFilterAndEvents()
})

onMounted(() => void loadAccountFilterAndEvents())
onUnmounted(() => { if (liveRefreshTimer) clearTimeout(liveRefreshTimer) })
</script>

<style scoped>
.events-page { display: flex; flex-direction: column; gap: 1.25rem; }
.page-header { display: flex; justify-content: space-between; align-items: flex-start; gap: 1rem; flex-wrap: wrap; }
.page-header > div:first-child { flex: 1; min-width: 280px; }
.page-title { font-family: 'Instrument Sans', sans-serif; font-size: var(--fs-3xl); font-weight: var(--fw-bold); color: var(--text-strong); margin-bottom: 0.25rem; }
.page-subtitle { color: var(--text-2); font-size: var(--fs-lg); max-width: 640px; }
.header-actions { display: flex; align-items: center; gap: 0.625rem; flex-wrap: wrap; }
.selection-bar { display: flex; align-items: center; gap: 0.5rem; background: var(--danger-soft); border: 1px solid var(--danger); border-radius: 8px; padding: 0.375rem 0.75rem; }
.selection-count { font-size: var(--fs-base); font-weight: var(--fw-semibold); color: var(--danger); white-space: nowrap; }
.pagination { display: flex; justify-content: space-between; align-items: center; gap: 1rem; padding: 0.75rem 0.25rem; }
.pag-info { font-size: var(--fs-base); color: var(--text-3); }
.pag-buttons { display: flex; gap: 0.375rem; }
.fade-enter-active, .fade-leave-active { transition: opacity 0.2s, transform 0.2s; }
.fade-enter-from, .fade-leave-to { opacity: 0; transform: translateY(-4px); }
</style>
