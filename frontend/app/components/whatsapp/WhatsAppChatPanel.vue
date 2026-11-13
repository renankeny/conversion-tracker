<template>
  <div ref="panelRef" class="chat-panel">
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

      <div class="chat-viewport-wrapper">
        <div
          ref="scrollBox"
          class="messages-viewport"
          @scroll="onScroll"
        >
          <WhatsAppLoadOlderTrigger
            :has-more="hasMore"
            :loading="loadingMore"
            :total-loaded="messages.length"
            :can-auto-trigger="canAutoLoadOlder"
            @load-more="loadOlder(crmLeadId, trackedUserId)"
          />

          <div v-if="!messages.length && !aiEvents.length" class="no-msgs">
            Conversa vinculada — aguardando mensagens.
          </div>

          <template v-for="item in feedItems" :key="item.key">
            <WhatsAppDateDivider
              v-if="item.type === 'date'"
              :date="item.date!"
            />
            <WhatsAppMessageBubble
              v-else-if="item.type === 'message' && item.message"
              :message="item.message"
            />
            <WhatsAppAiEventCard
              v-else-if="item.type === 'ai' && item.aiEvent"
              :event="item.aiEvent"
              :can-contest="canContestAI"
              @contest="openContestModal"
            />
          </template>
        </div>

        <Transition name="fade-pop">
          <WhatsAppScrollBottomBtn
            v-if="showScrollBottom"
            :unread-count="unreadBelow"
            @click="scrollToBottom(true)"
          />
        </Transition>
      </div>

      <WhatsAppAiContestModal
        v-model="showContestModal"
        :event="contestingEvent"
        @contested="upsertAIEvent"
      />
    </template>
  </div>
</template>

<script setup lang="ts">
import { MessageCircle } from 'lucide-vue-next'
import type { WhatsAppAIEvent, WhatsAppMediaStatus, WhatsAppMessage } from '#shared/types'
import WhatsAppOriginBanner from '~/components/whatsapp/WhatsAppOriginBanner.vue'
import WhatsAppDateDivider from '~/components/whatsapp/WhatsAppDateDivider.vue'
import WhatsAppMessageBubble from '~/components/whatsapp/WhatsAppMessageBubble.vue'
import WhatsAppAiEventCard from '~/components/whatsapp/WhatsAppAiEventCard.vue'
import WhatsAppAiContestModal from '~/components/whatsapp/WhatsAppAiContestModal.vue'
import WhatsAppLoadOlderTrigger from '~/components/whatsapp/WhatsAppLoadOlderTrigger.vue'
import WhatsAppScrollBottomBtn from '~/components/whatsapp/WhatsAppScrollBottomBtn.vue'
import { useWhatsAppChat } from '~/composables/useWhatsAppChat'
import { useLiveEventsStore } from '~/stores/useLiveEventsStore'

const props = defineProps<{
  crmLeadId?: number | null
  trackedUserId?: number | null
}>()

const { user } = useAuth()
const liveStore = useLiveEventsStore()
const panelRef = ref<HTMLElement | null>(null)
const showContestModal = ref(false)
const contestingEvent = ref<WhatsAppAIEvent | null>(null)

const canContestAI = computed(() => user.value?.role !== 'employee')

const {
  loading, loadingMore, canAutoLoadOlder, chat, messages, aiEvents, hasMore,
  scrollBox, showScrollBottom, unreadBelow, feedItems,
  loadInitial, loadOlder, scrollToBottom, onScroll,
  appendLiveMessage, updateMessageMedia, upsertAIEvent,
} = useWhatsAppChat()

const openContestModal = (ev: WhatsAppAIEvent) => {
  contestingEvent.value = ev
  showContestModal.value = true
}

watch(
  () => [props.crmLeadId, props.trackedUserId] as const,
  ([lId, uId]) => { void loadInitial(lId, uId) },
  { immediate: true },
)

let offMsg: (() => void) | null = null
let offMedia: (() => void) | null = null
let offAI: (() => void) | null = null
let panelObserver: IntersectionObserver | null = null

onMounted(() => {
  if (typeof IntersectionObserver !== 'undefined' && panelRef.value) {
    panelObserver = new IntersectionObserver((entries) => {
      if (entries[0]?.isIntersecting) {
        scrollToBottom(false)
        setTimeout(() => scrollToBottom(false), 50)
        setTimeout(() => scrollToBottom(false), 200)
      }
    })
    panelObserver.observe(panelRef.value)
  }

  offMsg = liveStore.onCustomEvent('whatsapp_message', (raw: unknown) => {
    const d = raw as { chat_id?: number; crm_lead_id?: number; tracked_user_id?: number; message?: WhatsAppMessage } | null
    const matchesChat = chat.value && d?.chat_id === chat.value.id
    const matchesLead = props.crmLeadId && d?.crm_lead_id === props.crmLeadId
    const matchesUser = props.trackedUserId && d?.tracked_user_id === props.trackedUserId
    if (!matchesChat && !matchesLead && !matchesUser) return
    if (!chat.value) {
      void loadInitial(props.crmLeadId, props.trackedUserId)
      return
    }
    if (d?.message) appendLiveMessage(d.message)
  })

  offMedia = liveStore.onCustomEvent('whatsapp_media_ready', (raw: unknown) => {
    const d = raw as {
      chat_id?: number; message_id?: number; media_status?: WhatsAppMediaStatus;
      media_mime_type?: string; media_size_bytes?: number; media_stream_url?: string
    } | null
    if (chat.value && d?.chat_id === chat.value.id && d.message_id) {
      updateMessageMedia(d.message_id, {
        media_status: d.media_status, media_mime_type: d.media_mime_type,
        media_size_bytes: d.media_size_bytes, media_stream_url: d.media_stream_url,
      })
    }
  })

  offAI = liveStore.onCustomEvent('whatsapp_ai_event', (raw: unknown) => {
    const d = raw as { chat_id?: number; crm_lead_id?: number; tracked_user_id?: number; ai_event?: WhatsAppAIEvent } | null
    if (!d?.ai_event) return
    if ((chat.value && d.chat_id === chat.value.id) || (props.crmLeadId && d.crm_lead_id === props.crmLeadId) || (props.trackedUserId && d.tracked_user_id === props.trackedUserId)) {
      upsertAIEvent(d.ai_event)
    }
  })
})

onUnmounted(() => {
  offMsg?.()
  offMedia?.()
  offAI?.()
  panelObserver?.disconnect()
  panelObserver = null
})
</script>

<style scoped>
.chat-panel { display: flex; flex-direction: column; gap: 0.75rem; height: 100%; min-height: 380px; min-width: 0; }
.chat-viewport-wrapper { position: relative; flex: 1; min-height: 320px; max-height: 560px; display: flex; flex-direction: column; }
.messages-viewport {
  flex: 1; min-height: 320px; max-height: 560px; min-width: 0; overflow-y: auto; overflow-x: hidden;
  padding: 0.75rem 0.85rem; background-color: var(--surface-deep);
  background-image:
    radial-gradient(circle at 10% 20%, rgba(255, 255, 255, 0.015) 0%, transparent 20%),
    radial-gradient(circle at 90% 80%, rgba(162, 240, 124, 0.02) 0%, transparent 25%);
  border: 1px solid var(--border-soft); border-radius: 12px; display: flex; flex-direction: column;
}
.chat-empty {
  flex: 1; min-height: 280px; padding: 2rem 1.25rem; background: var(--surface-deep);
  border: 1px dashed var(--border); border-radius: 12px; display: flex; flex-direction: column;
  align-items: center; justify-content: center; gap: 0.5rem; text-align: center; color: var(--text-3); font-size: var(--fs-sm);
}
.empty-icon { color: var(--text-3); opacity: 0.7; }
.empty-title { font-size: var(--fs-md); font-weight: var(--fw-semibold); color: var(--text-1); margin: 0; }
.empty-sub { font-size: var(--fs-sm); color: var(--text-3); max-width: 320px; line-height: 1.45; margin: 0; }
.no-msgs { text-align: center; color: var(--text-3); font-size: var(--fs-sm); padding: 2rem 0; }
.spinner { width: 1.5rem; height: 1.5rem; border: 2px solid var(--border); border-top-color: var(--brand); border-radius: 50%; animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
.fade-pop-enter-active, .fade-pop-leave-active { transition: opacity 0.15s ease, transform 0.15s ease; }
.fade-pop-enter-from, .fade-pop-leave-to { opacity: 0; transform: scale(0.85); }
</style>
