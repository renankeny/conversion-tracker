import { defineStore } from 'pinia'
import { ref } from 'vue'
import { useAuthStore } from '~/stores/useAuthStore'
import { useTenantStore } from '~/stores/useTenantStore'

export interface LiveEvent {
  account_id: number
  event_id: string
  event_name: string
  pixel_name: string
  external_id: string
  channels: string
  status: 'sent' | 'failed'
  ts: string
  value?: number
  city?: string
  state?: string
  country?: string
}

export const useLiveEventsStore = defineStore('liveEvents', () => {
  const MAX_EVENTS = 100
  const RECONNECT_DELAY = 3000

  const events = ref<LiveEvent[]>([])
  const lastEventSeq = ref(0)
  const connected = ref(false)
  const connecting = ref(false)
  const pixelFilter = ref('')
  const activeAccountId = ref(0)

  let ws: WebSocket | null = null
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null
  let shouldReconnect = false
  let connectGen = 0
  let currentToken = ''
  let currentPixel = ''
  const customListeners = new Map<string, Set<(payload: any) => void>>()

  function onCustomEvent<T = any>(eventType: string, handler: (payload: T) => void): () => void {
    let set = customListeners.get(eventType)
    if (!set) {
      set = new Set()
      customListeners.set(eventType, set)
    }
    set.add(handler)
    return () => {
      set?.delete(handler)
    }
  }

  function clearReconnectTimer(): void {
    if (reconnectTimer !== null) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
  }

  function scheduleReconnect(): void {
    if (!shouldReconnect || !currentToken) return
    clearReconnectTimer()
    reconnectTimer = setTimeout(() => {
      void connect(currentToken, currentPixel)
    }, RECONNECT_DELAY)
  }

  async function connect(token?: string, pixel = ''): Promise<void> {
    if (!import.meta.client) return
    const authStore = useAuthStore()
    const tenantStore = useTenantStore()
    const resolvedToken = token || authStore.token || ''
    if (!resolvedToken) return

    if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) {
      return
    }

    currentToken = resolvedToken
    currentPixel = pixel
    pixelFilter.value = pixel
    shouldReconnect = true
    activeAccountId.value = Number(tenantStore.activeAccountId || authStore.user?.account_id || 0)

    const myGen = ++connectGen
    connecting.value = true

    const api = useAPI()
    const wsBase = api.apiBase.replace(/^https/, 'wss').replace(/^http/, 'ws')
    const params = new URLSearchParams()

    try {
      const res = await api.post<{ ticket: string; expires_in: number }>('/admin/ws-ticket', {})
      if (myGen !== connectGen || !shouldReconnect) {
        connecting.value = false
        return
      }
      params.set('ticket', res.ticket)
    } catch (err: any) {
      if (myGen !== connectGen || !shouldReconnect) {
        connecting.value = false
        return
      }
      const status = err?.statusCode
      if (status === 401 || status === 403 || status === 423) {
        shouldReconnect = false
        connecting.value = false
        return
      }
      if (status === 404) {
        params.set('token', resolvedToken)
      } else {
        connecting.value = false
        scheduleReconnect()
        return
      }
    }

    if (pixel) params.set('pixel', pixel)
    const socket = new WebSocket(`${wsBase}/ws/events?${params.toString()}`)
    ws = socket

    socket.onopen = () => {
      if (myGen !== connectGen) {
        socket.close()
        return
      }
      connected.value = true
      connecting.value = false
      clearReconnectTimer()
    }

    socket.onclose = () => {
      if (ws === socket) {
        ws = null
        connected.value = false
        connecting.value = false
      }
      if (myGen === connectGen) {
        scheduleReconnect()
      }
    }

    socket.onerror = () => {
      socket.close()
    }

    socket.onmessage = (e: MessageEvent) => {
      try {
        const msg = JSON.parse(e.data as string) as { type: string; data: any }
        if (msg.type === 'new_event' && msg.data) {
          if (
            activeAccountId.value > 0 &&
            msg.data.account_id &&
            msg.data.account_id !== activeAccountId.value
          ) {
            return
          }
          events.value.unshift(msg.data as LiveEvent)
          if (events.value.length > MAX_EVENTS) {
            events.value.splice(MAX_EVENTS)
          }
          lastEventSeq.value++
          return
        }
        const handlers = customListeners.get(msg.type)
        if (handlers?.size) {
          for (const cb of handlers) {
            cb(msg.data)
          }
        }
      } catch {
        // Mensagem malformada — ignora silenciosamente
      }
    }
  }

  function disconnect(): void {
    shouldReconnect = false
    connectGen++
    clearReconnectTimer()
    currentToken = ''
    currentPixel = ''
    ws?.close()
    ws = null
    connected.value = false
    connecting.value = false
  }

  function clearEvents(): void {
    events.value = []
  }

  function reconnectForAccount(accountId: number): void {
    activeAccountId.value = accountId
    clearEvents()
    if (!import.meta.client) return
    const authStore = useAuthStore()
    const tok = currentToken || authStore.token || ''
    const pix = currentPixel
    connectGen++
    clearReconnectTimer()
    ws?.close()
    ws = null
    connected.value = false
    connecting.value = false
    if (tok) {
      shouldReconnect = true
      void connect(tok, pix)
    }
  }

  return {
    events,
    lastEventSeq,
    connected,
    connecting,
    pixelFilter,
    activeAccountId,
    connect,
    disconnect,
    clearEvents,
    reconnectForAccount,
    onCustomEvent,
  }
})
