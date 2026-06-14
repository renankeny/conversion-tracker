<template>
  <div class="events-page">
    <div class="page-header">
      <div>
        <h1 class="page-title">Eventos</h1>
        <p class="page-subtitle">
          Histórico completo dos eventos críticos enviados pra Meta CAPI.
          PageView e demais agregados ficam apenas no dashboard.
        </p>
      </div>
      <BaseButton variant="outline" size="sm" :loading="loading" @click="refresh">
        Atualizar
      </BaseButton>
    </div>

    <EventsFilters v-model="filters" />

    <EventsTable
      :events="events"
      :loading="loading"
      @select="onSelect"
    />

    <div v-if="total > 0" class="pagination">
      <span class="pag-info">
        Mostrando {{ offsetFrom }}-{{ offsetTo }} de {{ formatNumber(total) }}
      </span>
      <div class="pag-buttons">
        <BaseButton
          variant="ghost"
          size="sm"
          :disabled="offset === 0"
          @click="prevPage"
        >
          ← Anterior
        </BaseButton>
        <BaseButton
          variant="ghost"
          size="sm"
          :disabled="offsetTo >= total"
          @click="nextPage"
        >
          Próximo →
        </BaseButton>
      </div>
    </div>

    <EventDetailsModal v-model="modalOpen" :event="selected" />
  </div>
</template>

<script setup lang="ts">
import EventsFilters from '~/components/events/EventsFilters.vue'
import EventsTable from '~/components/events/EventsTable.vue'
import EventDetailsModal from '~/components/events/EventDetailsModal.vue'
import type { EventFilters, SentEvent } from '#shared/types'

definePageMeta({
  layout: 'app',
  middleware: 'auth',
})

useHead({ title: 'Eventos - TrackFind' })

const eventsApi = useEvents()
const toast = useToast()

const PAGE_SIZE = 50

const filters = ref<EventFilters>({})
const events = ref<SentEvent[]>([])
const total = ref(0)
const offset = ref(0)
const loading = ref(false)

const modalOpen = ref(false)
const selected = ref<SentEvent | null>(null)

const offsetFrom = computed(() => (events.value.length ? offset.value + 1 : 0))
const offsetTo = computed(() => offset.value + events.value.length)

const fetchEvents = async () => {
  loading.value = true
  try {
    const response = await eventsApi.list({
      ...filters.value,
      limit: PAGE_SIZE,
      offset: offset.value,
    })
    events.value = response.data || []
    total.value = response.total || 0
  } catch (error: any) {
    toast.error('Erro', error.message || 'Não foi possível carregar os eventos')
  } finally {
    loading.value = false
  }
}

const refresh = () => {
  offset.value = 0
  fetchEvents()
}

const prevPage = () => {
  offset.value = Math.max(0, offset.value - PAGE_SIZE)
  fetchEvents()
}

const nextPage = () => {
  offset.value += PAGE_SIZE
  fetchEvents()
}

const onSelect = (ev: SentEvent) => {
  selected.value = ev
  modalOpen.value = true
}

// Filtros mudaram → reseta paginação e refetch
watch(filters, () => {
  offset.value = 0
  fetchEvents()
}, { deep: true })

const formatNumber = (n: number): string => new Intl.NumberFormat('pt-BR').format(n)

onMounted(fetchEvents)
</script>

<style scoped>
.events-page {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 1rem;
  flex-wrap: wrap;
}

.page-header > div:first-child {
  flex: 1;
  min-width: 280px;
}

.page-title {
  font-family: 'Instrument Sans', sans-serif;
  font-size: 1.875rem;
  font-weight: 700;
  color: var(--text-strong);
  margin-bottom: 0.25rem;
}

.page-subtitle {
  color: var(--text-2);
  font-size: 0.9375rem;
  max-width: 640px;
}

.pagination {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 1rem;
  padding: 0.75rem 0.25rem;
}

.pag-info {
  font-size: 0.8125rem;
  color: var(--text-3);
}

.pag-buttons {
  display: flex;
  gap: 0.375rem;
}
</style>
