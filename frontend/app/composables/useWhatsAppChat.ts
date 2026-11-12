import type { WhatsAppChat, WhatsAppMessage } from '#shared/types'

export interface ChatFeedItem {
  key: string
  type: 'date' | 'message'
  date?: string
  message?: WhatsAppMessage
}

export const useWhatsAppChat = () => {
  const wa = useWhatsApp()

  const loading = ref(false)
  const loadingMore = ref(false)
  const chat = ref<WhatsAppChat | null>(null)
  const messages = ref<WhatsAppMessage[]>([])
  const hasMore = ref(false)
  const nextBeforeId = ref<number | null>(null)

  const scrollBox = ref<HTMLElement | null>(null)
  const showScrollBottom = ref(false)
  const unreadBelow = ref(0)
  const canAutoLoadOlder = ref(false)

  const scrollToBottom = (smooth = false) => {
    nextTick(() => {
      const el = scrollBox.value
      if (!el) return
      el.scrollTop = el.scrollHeight
      if (smooth) {
        el.scrollTo({ top: el.scrollHeight, behavior: 'smooth' })
      }
      unreadBelow.value = 0
      showScrollBottom.value = false
    })
  }

  const loadInitial = async (leadId?: number | null, userId?: number | null) => {
    if (!leadId && !userId) {
      chat.value = null
      messages.value = []
      hasMore.value = false
      nextBeforeId.value = null
      canAutoLoadOlder.value = false
      return
    }
    canAutoLoadOlder.value = false
    loading.value = true
    try {
      const res = leadId
        ? await wa.getChatByLead(leadId, { limit: 20 })
        : await wa.getChatByTrackedUser(userId!, { limit: 20 })

      chat.value = res.chat || null
      messages.value = res.messages || []
      hasMore.value = Boolean(res.has_more)
      nextBeforeId.value = res.next_before_id ?? null
    } catch {
      chat.value = null
      messages.value = []
      hasMore.value = false
      nextBeforeId.value = null
    } finally {
      loading.value = false
      await nextTick()
      scrollToBottom(false)
      requestAnimationFrame(() => scrollToBottom(false))
      setTimeout(() => scrollToBottom(false), 80)
      setTimeout(() => {
        scrollToBottom(false)
        canAutoLoadOlder.value = true
      }, 300)
    }
  }

  const loadOlder = async (leadId?: number | null, userId?: number | null) => {
    if (!hasMore.value || loadingMore.value || !nextBeforeId.value) return
    const el = scrollBox.value
    const prevScrollHeight = el ? el.scrollHeight : 0
    const prevScrollTop = el ? el.scrollTop : 0

    loadingMore.value = true
    try {
      const res = leadId
        ? await wa.getChatByLead(leadId, { before_id: nextBeforeId.value, limit: 20 })
        : await wa.getChatByTrackedUser(userId!, { before_id: nextBeforeId.value, limit: 20 })

      const older = res.messages || []
      const existingIds = new Set(messages.value.map((m) => m.id))
      const fresh = older.filter((m) => !existingIds.has(m.id))

      messages.value = [...fresh, ...messages.value]
      hasMore.value = Boolean(res.has_more)
      nextBeforeId.value = res.next_before_id ?? null

      await nextTick()
      if (el) {
        el.scrollTop = el.scrollHeight - prevScrollHeight + prevScrollTop
      }
    } finally {
      loadingMore.value = false
    }
  }

  const onScroll = () => {
    const el = scrollBox.value
    if (!el) return
    const distanceToBottom = el.scrollHeight - el.scrollTop - el.clientHeight
    if (distanceToBottom > 150) {
      showScrollBottom.value = true
    } else {
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
    if (distanceToBottom <= 180) {
      scrollToBottom(true)
    } else {
      unreadBelow.value++
      showScrollBottom.value = true
    }
  }

  const updateMessageMedia = (messageId: number, mediaData: Partial<WhatsAppMessage>) => {
    const idx = messages.value.findIndex((m) => m.id === messageId)
    if (idx >= 0) {
      messages.value[idx] = { ...messages.value[idx]!, ...mediaData }
    }
  }


  const feedItems = computed<ChatFeedItem[]>(() => {
    const items: ChatFeedItem[] = []
    let lastDateStr = ''

    for (const msg of messages.value) {
      if (msg.timestamp) {
        const d = new Date(msg.timestamp)
        const dateStr = `${d.getFullYear()}-${d.getMonth() + 1}-${d.getDate()}`
        if (dateStr !== lastDateStr) {
          items.push({
            key: `date-${dateStr}`,
            type: 'date',
            date: msg.timestamp,
          })
          lastDateStr = dateStr
        }
      }
      items.push({
        key: `msg-${msg.id}`,
        type: 'message',
        message: msg,
      })
    }
    return items
  })

  return {
    loading,
    loadingMore,
    canAutoLoadOlder,
    chat,
    messages,
    hasMore,
    scrollBox,
    showScrollBottom,
    unreadBelow,
    feedItems,
    loadInitial,
    loadOlder,
    scrollToBottom,
    onScroll,
    appendLiveMessage,
    updateMessageMedia,
  }
}
