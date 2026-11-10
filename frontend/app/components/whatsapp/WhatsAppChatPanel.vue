<template>
  <div class="chat-panel">
    <div v-if="loading" class="chat-empty">
      <div class="spinner" />
      <span>Carregando conversa...</span>
    </div>

    <div v-else-if="!chat" class="chat-empty">
      <MessageCircle :size="24" class="empty-icon" />
      <p class="empty-title">Sem conversa WhatsApp vinculada</p>
      <p class="empty-sub">
        Quando este lead iniciar contato por anúncio CTWA ou por um link com protocolo do site, o histórico aparecerá aqui automaticamente.
      </p>
    </div>

    <template v-else>
      <WhatsAppOriginBanner :chat="chat" />

      <div ref="scrollBox" class="messages-viewport">
        <div v-if="!messages.length" class="no-msgs">
          Conversa vinculada — aguardando mensagens.
        </div>

        <WhatsAppMessageBubble
          v-for="msg in messages"
          :key="msg.id"
          :message="msg"
        />
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { MessageCircle } from 'lucide-vue-next'
import type { WhatsAppChat, WhatsAppMessage } from '#shared/types'
import WhatsAppOriginBanner from '~/components/whatsapp/WhatsAppOriginBanner.vue'
import WhatsAppMessageBubble from '~/components/whatsapp/WhatsAppMessageBubble.vue'
import { useLiveEventsStore } from '~/stores/useLiveEventsStore'

const props = defineProps<{
  crmLeadId?: number | null
  trackedUserId?: number | null
}>()

const wa = useWhatsApp()
const liveStore = useLiveEventsStore()

const loading = ref(false)
const chat = ref<WhatsAppChat | null>(null)
const messages = ref<WhatsAppMessage[]>([])
const scrollBox = ref<HTMLElement | null>(null)

const scrollToBottom = () => {
  nextTick(() => {
    if (scrollBox.value) {
      scrollBox.value.scrollTop = scrollBox.value.scrollHeight
    }
  })
}

const loadConversation = async () => {
  if (!props.crmLeadId && !props.trackedUserId) {
    chat.value = null
    messages.value = []
    return
  }
  loading.value = true
  try {
    const res = props.crmLeadId
      ? await wa.getChatByLead(props.crmLeadId)
      : await wa.getChatByTrackedUser(props.trackedUserId!)
    chat.value = res.chat || null
    messages.value = res.messages || []
    scrollToBottom()
  } catch {
    chat.value = null
    messages.value = []
  } finally {
    loading.value = false
  }
}

watch(
  () => [props.crmLeadId, props.trackedUserId] as const,
  () => { void loadConversation() },
  { immediate: true },
)

let offMsg: (() => void) | null = null
let offMedia: (() => void) | null = null

onMounted(() => {
  offMsg = liveStore.onCustomEvent('whatsapp_message', (d: any) => {
    const matchesChat = chat.value && d?.chat_id === chat.value.id
    const matchesLead = props.crmLeadId && d?.crm_lead_id === props.crmLeadId
    const matchesUser = props.trackedUserId && d?.tracked_user_id === props.trackedUserId
    if (!matchesChat && !matchesLead && !matchesUser) return

    if (!chat.value) {
      void loadConversation()
      return
    }
    if (d?.message && !messages.value.some((m) => m.id === d.message.id)) {
      messages.value.push(d.message)
      scrollToBottom()
    }
  })

  offMedia = liveStore.onCustomEvent('whatsapp_media_ready', (d: any) => {
    if (!chat.value || d?.chat_id !== chat.value.id) return
    const idx = messages.value.findIndex((m) => m.id === d.message_id)
    if (idx >= 0) {
      messages.value[idx] = {
        ...messages.value[idx]!,
        media_status: d.media_status,
        media_mime_type: d.media_mime_type,
        media_size_bytes: d.media_size_bytes,
        media_stream_url: d.media_stream_url,
      }
    }
  })
})

onUnmounted(() => {
  offMsg?.()
  offMedia?.()
})
</script>

<style scoped>
.chat-panel {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  height: 100%;
  min-height: 320px;
}
.messages-viewport {
  flex: 1;
  min-height: 260px;
  max-height: 430px;
  overflow-y: auto;
  padding: 0.75rem;
  background: var(--surface-deep);
  border: 1px solid var(--border-soft);
  border-radius: 10px;
  display: flex;
  flex-direction: column;
  gap: 0.625rem;
}
.chat-empty {
  flex: 1;
  min-height: 280px;
  padding: 2rem 1.25rem;
  background: var(--surface-deep);
  border: 1px dashed var(--border);
  border-radius: 10px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  text-align: center;
  color: var(--text-3);
  font-size: var(--fs-sm);
}
.empty-icon { color: var(--text-3); opacity: 0.7; }
.empty-title { font-size: var(--fs-md); font-weight: var(--fw-semibold); color: var(--text-1); margin: 0; }
.empty-sub { font-size: var(--fs-sm); color: var(--text-3); max-width: 320px; line-height: 1.45; margin: 0; }
.no-msgs { text-align: center; color: var(--text-3); font-size: var(--fs-sm); padding: 2rem 0; }
.spinner { width: 1.5rem; height: 1.5rem; border: 2px solid var(--border); border-top-color: var(--brand); border-radius: 50%; animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
</style>

