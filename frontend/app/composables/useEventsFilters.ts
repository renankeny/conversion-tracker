import { ref, computed, watch, nextTick, onMounted, onUnmounted } from 'vue'
import type { EventFilters, GroupedEvent } from '#shared/types'
import { useLiveEventsStore } from '~/stores/useLiveEventsStore'
import { useTenantStore } from '~/stores/useTenantStore'
import { errorMessage } from '~/utils/errorMessage'

const PAGE_SIZE = 50

export const useEventsFilters = () => {
  const eventsApi = useEvents()
  const accountsApi = useAccounts()
  const toast = useToast()
  const liveStore = useLiveEventsStore()
  const tenantStore = useTenantStore()

  const filters = ref<EventFilters>({})
  const eventsFilterMap = ref<Record<string, string[]>>({})
  let eventsFilterLoaded = false
  let isRestoringEventFilter = false
  let lastPixel = ''
  let lastEventCSV = ''
  let requestId = 0

  const events = ref<GroupedEvent[]>([])
  const total = ref(0)
  const offset = ref(0)
  const loading = ref(false)
  const modalOpen = ref(false)
  const selected = ref<GroupedEvent | null>(null)
  const selectedIds = ref<string[]>([])

  const offsetFrom = computed(() => (events.value.length ? offset.value + 1 : 0))
  const offsetTo = computed(() => offset.value + events.value.length)

  const fetchEvents = async (silent = false) => {
    const reqId = ++requestId
    if (!silent) loading.value = true
    try {
      const response = await eventsApi.listGrouped({
        ...filters.value,
        limit: PAGE_SIZE,
        offset: offset.value,
        include_user_context: true,
      })
      if (reqId !== requestId) return
      events.value = response.data || []
      total.value = response.total || 0
    } catch (err: unknown) {
      if (reqId !== requestId) return
      if (!silent) toast.error('Erro', errorMessage(err, 'Não foi possível carregar os eventos'))
    } finally {
      if (reqId === requestId && !silent) loading.value = false
    }
  }

  const applyRestoredFilters = async (next: EventFilters) => {
    isRestoringEventFilter = true
    filters.value = next
    lastPixel = next.pixel || ''
    lastEventCSV = next.event || ''
    await nextTick()
    isRestoringEventFilter = false
  }

  watch(filters, async (next) => {
    if (isRestoringEventFilter) return
    const nextPixel = next.pixel || ''
    const nextEvent = next.event || ''

    if (eventsFilterLoaded && nextPixel !== lastPixel) {
      lastPixel = nextPixel
      const saved = (eventsFilterMap.value[nextPixel] || []).join(',')
      if (nextEvent !== saved) {
        await applyRestoredFilters({ ...next, event: saved || undefined })
      }
      offset.value = 0
      void fetchEvents()
      return
    }

    if (eventsFilterLoaded && nextEvent !== lastEventCSV) {
      lastEventCSV = nextEvent
      const arr = nextEvent.split(',').map((s) => s.trim()).filter(Boolean)
      eventsFilterMap.value = { ...eventsFilterMap.value, [nextPixel]: arr }
      accountsApi.updateEventsFilter(nextPixel, arr).catch(() => {})
    }

    offset.value = 0
    void fetchEvents()
  }, { deep: true })

  const loadAccountFilterAndEvents = async () => {
    eventsFilterLoaded = false
    try {
      const res = await accountsApi.getEventsFilter()
      eventsFilterMap.value = res.events_filter || {}
    } catch {
      eventsFilterMap.value = {}
    }
    eventsFilterLoaded = true
    const key = filters.value.pixel || ''
    const saved = (eventsFilterMap.value[key] || []).join(',')
    await applyRestoredFilters({ ...filters.value, event: saved || undefined })
    void fetchEvents()
  }

  let liveRefreshTimer: ReturnType<typeof setTimeout> | null = null
  watch(() => liveStore.lastEventSeq, () => {
    if (offset.value > 0 || selectedIds.value.length > 0 || modalOpen.value) return
    if (liveRefreshTimer) clearTimeout(liveRefreshTimer)
    liveRefreshTimer = setTimeout(() => void fetchEvents(true), 500)
  })

  watch(() => tenantStore.switchVersion, async () => {
    modalOpen.value = false
    selected.value = null
    selectedIds.value = []
    offset.value = 0
    await applyRestoredFilters({})
    void loadAccountFilterAndEvents()
  })

  onMounted(() => void loadAccountFilterAndEvents())
  onUnmounted(() => { if (liveRefreshTimer) clearTimeout(liveRefreshTimer) })

  const refresh = () => { offset.value = 0; void fetchEvents() }
  const prevPage = () => { offset.value = Math.max(0, offset.value - PAGE_SIZE); void fetchEvents() }
  const nextPage = () => { offset.value += PAGE_SIZE; void fetchEvents() }

  return {
    filters,
    events,
    total,
    offset,
    loading,
    modalOpen,
    selected,
    selectedIds,
    offsetFrom,
    offsetTo,
    fetchEvents,
    refresh,
    prevPage,
    nextPage,
  }
}

