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
import { useEventsFilters } from '~/composables/useEventsFilters'
import { errorMessage } from '~/utils/errorMessage'
import type { GroupedEvent } from '#shared/types'

definePageMeta({ layout: 'app', middleware: 'auth' })
const { formatTitle } = useBranding()
useHead({ title: computed(() => formatTitle('Eventos')) })

const eventsApi = useEvents()
const toast = useToast()

const {
  filters, events, total, offset, loading, modalOpen, selected, selectedIds,
  offsetFrom, offsetTo, fetchEvents, refresh, prevPage, nextPage,
} = useEventsFilters()

const deleteModalOpen = ref(false)
const deleting = ref(false)

const onSelect = (ev: GroupedEvent) => { selected.value = ev; modalOpen.value = true }
const onSelectionChange = (ids: string[]) => { selectedIds.value = ids }
const clearSelection = () => { selectedIds.value = [] }
const confirmDelete = () => { deleteModalOpen.value = true }

const onRetry = async (eventIds: string[]) => {
  try {
    const r = await eventsApi.retryByEventIDs(eventIds)
    toast.success('Reenfileirado', `${r.queued} evento${r.queued !== 1 ? 's' : ''} adicionado${r.queued !== 1 ? 's' : ''} à fila.`)
    await fetchEvents()
  } catch (err: unknown) {
    toast.error('Erro', errorMessage(err, 'Não foi possível reenfileirar'))
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
.page-title { font-family: var(--font-heading); font-size: var(--fs-3xl); font-weight: var(--fw-bold); color: var(--text-strong); margin-bottom: 0.25rem; }
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
