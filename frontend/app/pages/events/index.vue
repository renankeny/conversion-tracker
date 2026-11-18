<template>
  <div class="events-page">
    <div class="page-header">
      <div>
        <div class="title-row">
          <h1 class="page-title">Eventos</h1>
          <span class="live-badge" :class="liveStore.connected ? 'is-live' : 'is-offline'">
            <span class="live-dot" />
            {{ liveStore.connected ? 'Ao vivo' : 'Reconectando' }}
          </span>
        </div>
        <p class="page-subtitle">
          Auditoria em tempo real de conversões e eventos enviados aos canais de mídia (exceto PageView e ViewContent, contabilizados no Dashboard).
        </p>
      </div>

      <div class="header-actions">
        <Transition name="fade">
          <div v-if="canManageUsers && selectedItems.length > 0" class="selection-bar">
            <span class="selection-count">
              {{ selectedItems.length }} selecionado{{ selectedItems.length > 1 ? 's' : '' }}
            </span>
            <BaseButton
              variant="outline"
              size="sm"
              :loading="retrying"
              @click="onRetry(selectedItems)"
            >
              <template #iconLeft>
                <RefreshCw class="w-3.5 h-3.5" />
              </template>
              Reenviar selecionados
            </BaseButton>
            <BaseButton
              variant="danger"
              size="sm"
              :loading="deleting"
              @click="confirmDelete"
            >
              <template #iconLeft>
                <Trash2 class="w-3.5 h-3.5" />
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

    <EventsFilters v-model="filters" @clear="clearFilters" />

    <EventsTable
      :events="events"
      :loading="loading"
      :can-manage="canManageUsers"
      :selected-items="selectedItems"
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

    <EventDetailsModal
      v-model="modalOpen"
      :source="selected"
      :currency="selectedCurrency"
      @retried="onModalRetried"
    />

    <ConfirmModal
      v-if="canManageUsers"
      v-model="deleteModalOpen"
      title="Deletar eventos?"
      :message="`Isso vai remover permanentemente ${selectedItems.length} evento${selectedItems.length > 1 ? 's' : ''} (todos os canais). Não é possível desfazer.`"
      confirm-label="Deletar"
      variant="danger"
      :loading="deleting"
      @confirm="doDelete"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onUnmounted } from 'vue'
import { RefreshCw, Trash2 } from 'lucide-vue-next'
import EventsFilters from '~/components/events/EventsFilters.vue'
import EventsTable from '~/components/events/EventsTable.vue'
import EventDetailsModal from '~/components/events/EventDetailsModal.vue'
import ConfirmModal from '~/components/ConfirmModal.vue'
import { useEventsFilters } from '~/composables/useEventsFilters'
import { useAuthStore } from '~/stores/useAuthStore'
import { useLiveEventsStore } from '~/stores/useLiveEventsStore'
import { usePixelStore } from '~/stores/usePixelStore'
import { errorMessage } from '~/utils/errorMessage'
import type { EventKeyRef, GroupedEvent } from '#shared/types'

definePageMeta({ layout: 'app', middleware: 'auth' })
const { formatTitle } = useBranding()
useHead({ title: computed(() => formatTitle('Eventos')) })

const eventsApi = useEvents()
const toast = useToast()
const authStore = useAuthStore()
const liveStore = useLiveEventsStore()
const pixelStore = usePixelStore()

const canManageUsers = computed(() => authStore.canManageUsers)

const {
  filters, events, total, offset, loading, modalOpen, selected, selectedItems,
  offsetFrom, offsetTo, fetchEvents, clearFilters, refresh, prevPage, nextPage,
} = useEventsFilters()

const selectedCurrency = computed(() =>
  pixelStore.pixels.find((p) => p.name === selected.value?.pixel_name)?.tracker_config?.product_currency || 'BRL',
)

const deleteModalOpen = ref(false)
const deleting = ref(false)
const retrying = ref(false)
let retryRefreshTimer: ReturnType<typeof setTimeout> | null = null

onUnmounted(() => {
  if (retryRefreshTimer) clearTimeout(retryRefreshTimer)
})

const onSelect = (ev: GroupedEvent) => { selected.value = ev; modalOpen.value = true }
const onSelectionChange = (items: EventKeyRef[]) => { selectedItems.value = items }
const clearSelection = () => { selectedItems.value = [] }
const confirmDelete = () => { deleteModalOpen.value = true }
const onModalRetried = () => { void fetchEvents(true) }

const onRetry = async (items: EventKeyRef[]) => {
  if (!items.length || retrying.value) return
  retrying.value = true
  try {
    const r = await eventsApi.retryEvents(items)
    if (r.queued === 0) {
      toast.warning(
        'Reenvio indisponível',
        'O payload original deste evento já expirou da janela de retenção.',
      )
    } else {
      toast.success(
        'Reenfileirado',
        `${r.queued} evento${r.queued !== 1 ? 's' : ''} enviado${r.queued !== 1 ? 's' : ''} para reprocessamento.`,
      )
      selectedItems.value = []
      if (retryRefreshTimer) clearTimeout(retryRefreshTimer)
      retryRefreshTimer = setTimeout(() => void fetchEvents(true), 1200)
    }
  } catch (err: unknown) {
    toast.error('Erro', errorMessage(err, 'Não foi possível reenfileirar'))
  } finally {
    retrying.value = false
  }
}

const doDelete = async () => {
  deleteModalOpen.value = false
  deleting.value = true
  try {
    const { deleted } = await eventsApi.deleteEvents(selectedItems.value)
    toast.success('Eventos deletados', `${deleted} registro${deleted !== 1 ? 's' : ''} removido${deleted !== 1 ? 's' : ''}.`)
    selectedItems.value = []
    offset.value = 0
    await fetchEvents()
  } catch (err: unknown) {
    toast.error('Erro', errorMessage(err, 'Não foi possível deletar os eventos'))
  } finally {
    deleting.value = false
  }
}

const formatNumber = (n: number): string => new Intl.NumberFormat('pt-BR').format(n)
</script>

<style scoped>
.events-page { display: flex; flex-direction: column; gap: 1.25rem; }
.page-header { display: flex; justify-content: space-between; align-items: flex-start; gap: 1rem; flex-wrap: wrap; }
.page-header > div:first-child { flex: 1; min-width: 280px; }
.title-row { display: flex; align-items: center; gap: 0.75rem; flex-wrap: wrap; margin-bottom: 0.25rem; }
.page-title { font-family: var(--font-heading); font-size: var(--fs-3xl); font-weight: var(--fw-bold); color: var(--text-strong); margin: 0; }
.page-subtitle { color: var(--text-2); font-size: var(--fs-md); max-width: 680px; line-height: 1.45; }
.live-badge { display: inline-flex; align-items: center; gap: 0.375rem; padding: 0.1875rem 0.5625rem; border-radius: 9999px; font-size: var(--fs-xs); font-weight: var(--fw-semibold); border: 1px solid var(--border); background: var(--surface-2); color: var(--text-3); }
.live-badge.is-live { background: var(--brand-soft); border-color: var(--brand-border); color: var(--brand); }
.live-dot { width: 6px; height: 6px; border-radius: 9999px; background: var(--text-3); }
.live-badge.is-live .live-dot { background: var(--brand); animation: livePulse 1.8s ease-in-out infinite; }
@keyframes livePulse { 0%, 100% { opacity: 1; transform: scale(1); } 50% { opacity: 0.45; transform: scale(0.85); } }
.header-actions { display: flex; align-items: center; gap: 0.625rem; flex-wrap: wrap; }
.selection-bar { display: flex; align-items: center; gap: 0.5rem; flex-wrap: wrap; background: var(--surface-2); border: 1px solid var(--brand-border); border-radius: 8px; padding: 0.375rem 0.75rem; }
.selection-count { font-size: var(--fs-base); font-weight: var(--fw-semibold); color: var(--text-1); white-space: nowrap; margin-right: 0.25rem; }
.pagination { display: flex; justify-content: space-between; align-items: center; gap: 1rem; padding: 0.75rem 0.25rem; }
.pag-info { font-size: var(--fs-base); color: var(--text-3); }
.pag-buttons { display: flex; gap: 0.375rem; }
.fade-enter-active, .fade-leave-active { transition: opacity 0.2s, transform 0.2s; }
.fade-enter-from, .fade-leave-to { opacity: 0; transform: translateY(-4px); }
</style>
