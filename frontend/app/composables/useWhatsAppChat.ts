import type { WhatsAppAIEvent, WhatsAppChat, WhatsAppChatWithMessages, WhatsAppMessage } from '#shared/types'
import { dateKeySP } from '~/utils/formatDate'

export interface ChatFeedItem {
  key: string
  type: 'date' | 'message' | 'ai'
  date?: string
  message?: WhatsAppMessage
  aiEvent?: WhatsAppAIEvent
}

export interface ChatTargetRef {
  chatId?: number | null
  leadId?: number | null
  userId?: number | null
  publicToken?: string | null
}

export const useWhatsAppChat = () => {
  const wa = useWhatsApp()
  const api = useAPI()

  const loading = ref(false)
  const loadingMore = ref(false)
  const chat = ref<WhatsAppChat | null>(null)
  const messages = ref<WhatsAppMessage[]>([])
  const aiEvents = ref<WhatsAppAIEvent[]>([])
  const hasMore = ref(false)
  const nextBeforeId = ref<number | null>(null)

  const scrollBox = ref<HTMLElement | null>(null)
  const showScrollBottom = ref(false)
  const unreadBelow = ref(0)
  const canAutoLoadOlder = ref(false)

  const fetchChatPage = (target: ChatTargetRef, params: { before_id?: number | null; limit: number }): Promise<WhatsAppChatWithMessages> => {
    const q = new URLSearchParams()
    if (params.before_id) q.set('before_id', String(params.before_id))
    q.set('limit', String(params.limit))
    const qs = `?${q.toString()}`

    if (target.publicToken) {
      const base = `/public/dashboard/${encodeURIComponent(target.publicToken)}`
      if (target.chatId) {
        return api.publicGet<WhatsAppChatWithMessages>(`${base}/chats/${target.chatId}/messages${qs}`)
      }
      return api.publicGet<WhatsAppChatWithMessages>(`${base}/crm/leads/${target.leadId}/chat${qs}`)
    }
    if (target.chatId) return wa.getChatMessages(target.chatId, params)
    if (target.leadId) return wa.getChatByLead(target.leadId, params)
    return wa.getChatByTrackedUser(target.userId!, params)
  }

  const scrollToBottom = (smooth = false) => {
    nextTick(() => {
      const el = scrollBox.value
      if (!el) return
      el.scrollTop = el.scrollHeight
      if (smooth) el.scrollTo({ top: el.scrollHeight, behavior: 'smooth' })
      unreadBelow.value = 0
      showScrollBottom.value = false
    })
  }

  let loadSeq = 0

  const loadInitial = async (leadId?: number | null, userId?: number | null, chatId?: number | null, publicToken?: string | null) => {
    const seq = ++loadSeq
    const hasTarget = Boolean(chatId || leadId || (!publicToken && userId))
    if (!hasTarget) {
      chat.value = null
      messages.value = []
      aiEvents.value = []
      hasMore.value = false
      nextBeforeId.value = null
      canAutoLoadOlder.value = false
      loading.value = false
      return
    }
    canAutoLoadOlder.value = false
    loading.value = true
    try {
      const res = await fetchChatPage({ chatId, leadId, userId, publicToken }, { limit: 20 })
      if (seq !== loadSeq) return
      chat.value = res.chat || null
      messages.value = res.messages || []
      aiEvents.value = res.ai_events || []
      hasMore.value = Boolean(res.has_more)
      nextBeforeId.value = res.next_before_id ?? null
    } catch {
      if (seq !== loadSeq) return
      chat.value = null
      messages.value = []
      aiEvents.value = []
      hasMore.value = false
      nextBeforeId.value = null
    } finally {
      if (seq === loadSeq) {
        loading.value = false
        await nextTick()
        scrollToBottom(false)
        requestAnimationFrame(() => { if (seq === loadSeq) scrollToBottom(false) })
        setTimeout(() => { if (seq === loadSeq) scrollToBottom(false) }, 80)
        setTimeout(() => {
          if (seq === loadSeq) {
            scrollToBottom(false)
            canAutoLoadOlder.value = true
          }
        }, 300)
      }
    }
  }

  const loadOlder = async (leadId?: number | null, userId?: number | null, chatId?: number | null, publicToken?: string | null) => {
    if (!hasMore.value || loadingMore.value || !nextBeforeId.value) return
    const seq = loadSeq
    const el = scrollBox.value
    const prevScrollHeight = el ? el.scrollHeight : 0
    const prevScrollTop = el ? el.scrollTop : 0

    loadingMore.value = true
    try {
      const res = await fetchChatPage({ chatId, leadId, userId, publicToken }, { before_id: nextBeforeId.value, limit: 20 })
      if (seq !== loadSeq) return
      const older = res.messages || []
      const existingIds = new Set(messages.value.map((m) => m.id))
      const fresh = older.filter((m) => !existingIds.has(m.id))

      messages.value = [...fresh, ...messages.value]
      if (res.ai_events) aiEvents.value = res.ai_events
      hasMore.value = Boolean(res.has_more)
      nextBeforeId.value = res.next_before_id ?? null

      await nextTick()
      if (el) el.scrollTop = el.scrollHeight - prevScrollHeight + prevScrollTop
    } catch {
      // Falha de rede silenciosa para permitir nova tentativa no scroll
    } finally {
      if (seq === loadSeq) loadingMore.value = false
    }
  }

  const onScroll = () => {
    const el = scrollBox.value
    if (!el) return
    const distanceToBottom = el.scrollHeight - el.scrollTop - el.clientHeight
    if (distanceToBottom > 150) showScrollBottom.value = true
    else {
      showScrollBottom.value = false
      unreadBelow.value = 0
    }
  }

  const appendLiveMessage = (msg: WhatsAppMessage) => {
    if (messages.value.some((m) => m.id === msg.id)) return
    messages.value.push(msg)
    const el = scrollBox.value
    if (!el) {
      scrollToBottom(false)
      return
    }
    const distanceToBottom = el.scrollHeight - el.scrollTop - el.clientHeight
    if (distanceToBottom <= 180) scrollToBottom(true)
    else {
      unreadBelow.value++
      showScrollBottom.value = true
    }
  }

  const updateMessageMedia = (messageId: number, mediaData: Partial<WhatsAppMessage>) => {
    const idx = messages.value.findIndex((m) => m.id === messageId)
    if (idx >= 0) messages.value[idx] = { ...messages.value[idx]!, ...mediaData }
  }

  const upsertAIEvent = (ev: WhatsAppAIEvent) => {
    const idx = aiEvents.value.findIndex((e) => e.id === ev.id)
    if (idx >= 0) aiEvents.value[idx] = ev
    else {
      aiEvents.value.push(ev)
      scrollToBottom(true)
    }
  }

  const feedItems = computed<ChatFeedItem[]>(() => {
    const items: ChatFeedItem[] = []
    let lastDateStr = ''
    const byMsg = new Map<number, WhatsAppAIEvent[]>()
    const placedIds = new Set<number>()

    for (const ev of aiEvents.value) {
      if (ev.evidence_message_id) {
        const arr = byMsg.get(ev.evidence_message_id) || []
        arr.push(ev)
        byMsg.set(ev.evidence_message_id, arr)
      }
    }

    for (const msg of messages.value) {
      if (msg.timestamp) {
        const dateStr = dateKeySP(msg.timestamp)
        if (dateStr && dateStr !== lastDateStr) {
          items.push({ key: `date-${dateStr}`, type: 'date', date: msg.timestamp })
          lastDateStr = dateStr
        }
      }
      items.push({ key: `msg-${msg.id}`, type: 'message', message: msg })
      const attached = byMsg.get(msg.id)
      if (attached) {
        for (const ev of attached) {
          placedIds.add(ev.id)
          items.push({ key: `ai-${ev.id}`, type: 'ai', aiEvent: ev })
        }
      }
    }

    for (const ev of aiEvents.value) {
      if (!placedIds.has(ev.id)) items.push({ key: `ai-${ev.id}`, type: 'ai', aiEvent: ev })
    }
    return items
  })

  return {
    loading, loadingMore, canAutoLoadOlder, chat, messages, aiEvents, hasMore,
    scrollBox, showScrollBottom, unreadBelow, feedItems,
    loadInitial, loadOlder, scrollToBottom, onScroll,
    appendLiveMessage, updateMessageMedia, upsertAIEvent,
  }
}
