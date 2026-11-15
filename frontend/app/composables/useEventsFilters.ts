import { ref, computed, watch, nextTick, onMounted, onUnmounted } from 'vue'
import type { EventFilters, EventKeyRef, GroupedEvent } from '#shared/types'
import { useAuthStore } from '~/stores/useAuthStore'
import { useLiveEventsStore } from '~/stores/useLiveEventsStore'
import { usePixelStore } from '~/stores/usePixelStore'
import { useTenantStore } from '~/stores/useTenantStore'
import { errorMessage } from '~/utils/errorMessage'

const PAGE_SIZE = 50

export const useEventsFilters = () => {
  const eventsApi = useEvents()
  const accountsApi = useAccounts()
  const toast = useToast()
  const authStore = useAuthStore()
  const liveStore = useLiveEventsStore()
  const pixelStore = usePixelStore()
  const tenantStore = useTenantStore()

  const canManageUsers = computed(() => authStore.canManageUsers)

  const filters = ref<EventFilters>({})
  const eventsFilterMap = ref<Record<string, string[]>>({})
  let eventsFilterLoaded = false
  let isRestoringEventFilter = false
  let lastPixel = ''
  let lastEventCSV = ''
  let requestId = 0
  let filterSeq = 0

  const events = ref<GroupedEvent[]>([])
  const total = ref(0)
  const offset = ref(0)
  const loading = ref(false)
  const modalOpen = ref(false)
  const selected = ref<GroupedEvent | null>(null)
  const selectedItems = ref<EventKeyRef[]>([])
  const selectedIds = computed(() => selectedItems.value.map((i) => i.event_id))

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
      if (reqId === requestId) loading.value = false
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

  const clearFilters = async () => {
    isRestoringEventFilter = true
    filters.value = {}
    lastPixel = ''
    lastEventCSV = ''
    offset.value = 0
    selectedItems.value = []
    await nextTick()
    isRestoringEventFilter = false
    void fetchEvents()
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
      if (nextPixel !== '') {
        eventsFilterMap.value = { ...eventsFilterMap.value, [nextPixel]: arr }
        if (canManageUsers.value) {
          accountsApi.updateEventsFilter(nextPixel, arr).catch(() => {})
        }
      }
    }

    offset.value = 0
    void fetchEvents()
  }, { deep: true })

  const loadAccountFilterAndEvents = async () => {
    const mySeq = ++filterSeq
    eventsFilterLoaded = false
    let nextMap: Record<string, string[]> = {}
    try {
      const res = await accountsApi.getEventsFilter()
      nextMap = res.events_filter || {}
    } catch {
      nextMap = {}
    }
    if (mySeq !== filterSeq) return
    eventsFilterMap.value = nextMap
    eventsFilterLoaded = true
    const key = filters.value.pixel || ''
    const saved = (eventsFilterMap.value[key] || []).join(',')
    await applyRestoredFilters({ ...filters.value, event: saved || undefined })
    if (mySeq !== filterSeq) return
    void fetchEvents()
  }

  let liveRefreshTimer: ReturnType<typeof setTimeout> | null = null
  let unsubUpdated: (() => void) | null = null
  let unsubWhatsappAi: (() => void) | null = null

  const scheduleLiveRefresh = (delayMs = 600) => {
    if (offset.value > 0 || selectedItems.value.length > 0 || modalOpen.value) return
    if (liveRefreshTimer) clearTimeout(liveRefreshTimer)
    liveRefreshTimer = setTimeout(() => void fetchEvents(true), delayMs)
  }

  watch(() => liveStore.lastConversionSeq, () => {
    if (offset.value > 0 || selectedItems.value.length > 0 || modalOpen.value) return
    const ev = liveStore.lastConversionEvent
    if (!ev) return
    if (filters.value.to && new Date(ev.ts) > new Date(filters.value.to)) return
    if (filters.value.pixel && ev.pixel_name !== filters.value.pixel) return
    if (filters.value.event && !filters.value.event.split(',').map((s) => s.trim()).includes(ev.event_name)) return
    if (filters.value.channel && !(ev.channels || '').split(',').map((s) => s.trim()).includes(filters.value.channel)) return
    if (filters.value.status === 'failed' && ev.status !== 'failed') return
    scheduleLiveRefresh(600)
  })

  watch(() => tenantStore.switchVersion, async () => {
    modalOpen.value = false
    selected.value = null
    selectedItems.value = []
    offset.value = 0
    pixelStore.fetch().catch(() => {})
    await applyRestoredFilters({})
    void loadAccountFilterAndEvents()
  })

  onMounted(() => {
    void loadAccountFilterAndEvents()
    unsubUpdated = liveStore.onCustomEvent('events_updated', () => scheduleLiveRefresh(400))
    unsubWhatsappAi = liveStore.onCustomEvent('whatsapp_ai_event', () => scheduleLiveRefresh(600))
  })

  onUnmounted(() => {
    if (liveRefreshTimer) clearTimeout(liveRefreshTimer)
    unsubUpdated?.()
    unsubWhatsappAi?.()
  })

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
    selectedItems,
    selectedIds,
    offsetFrom,
    offsetTo,
    fetchEvents,
    clearFilters,
    refresh,
    prevPage,
    nextPage,
  }
}
