<template>
  <div class="msg-row" :class="{ 'from-me': message.is_from_me }">
    <div class="msg-bubble" :class="{ 'bubble-me': message.is_from_me }">
      <!-- Mídia ativa no MinIO / Cloud Storage -->
      <template v-if="message.has_media">
        <div v-if="message.media_status === 'ready' && message.media_stream_url" class="media-ready">
          <WhatsAppAudioPlayer
            v-if="message.message_type === 'audio'"
            :src="message.media_stream_url"
            :initial-duration="message.media_duration_sec"
          />

          <a
            v-else-if="message.message_type === 'image' || message.message_type === 'sticker'"
            :href="message.media_stream_url"
            target="_blank"
            rel="noopener"
            class="img-link"
          >
            <img :src="message.media_stream_url" alt="Imagem WhatsApp" class="chat-img" loading="lazy">
          </a>

          <a
            v-else
            :href="message.media_stream_url"
            target="_blank"
            rel="noopener"
            class="doc-pill"
          >
            <FileText :size="15" />
            <span class="doc-name">{{ message.media_filename || mediaTypeLabel }}</span>
          </a>
        </div>

        <!-- Mídia em processamento assíncrono -->
        <div v-else-if="message.media_status === 'pending'" class="media-notice">
          <Clock :size="13" />
          <span>Sincronizando {{ mediaTypeLabel.toLowerCase() }}...</span>
        </div>

        <!-- Mídia expirada após 14 dias -->
        <div v-else-if="message.media_status === 'expired'" class="media-notice expired">
          <Archive :size="13" />
          <span>{{ expiredLabel }}</span>
        </div>

        <!-- Vídeo pesado ignorado -->
        <div v-else-if="message.media_status === 'skipped_size'" class="media-notice">
          <Film :size="13" />
          <span>Mídia acima do limite (&gt;15 MB)</span>
        </div>

        <div v-else class="media-notice">
          <AlertCircle :size="13" />
          <span>{{ mediaTypeLabel }} indisponível</span>
        </div>
      </template>

      <!-- Texto da mensagem ou legenda da mídia -->
      <p v-if="message.content" class="msg-text">{{ message.content }}</p>

      <!-- Rodapé WhatsApp: horário + confirmação de leitura -->
      <div class="msg-footer">
        <span class="msg-time">{{ formattedTime }}</span>
        <CheckCheck v-if="message.is_from_me" :size="14" class="msg-check" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { FileText, Clock, Archive, Film, AlertCircle, CheckCheck } from 'lucide-vue-next'
import type { WhatsAppMessage } from '#shared/types'
import WhatsAppAudioPlayer from '~/components/whatsapp/WhatsAppAudioPlayer.vue'

const props = defineProps<{
  message: WhatsAppMessage
}>()

const mediaTypeLabel = computed(() => {
  switch (props.message.message_type) {
    case 'audio': return 'Áudio'
    case 'image': return 'Imagem'
    case 'video': return 'Vídeo'
    case 'document': return 'Documento'
    case 'sticker': return 'Figurinha'
    default: return 'Mídia'
  }
})

const expiredLabel = computed(() => {
  if (props.message.message_type === 'audio' && props.message.media_duration_sec > 0) {
    const m = Math.floor(props.message.media_duration_sec / 60)
    const s = props.message.media_duration_sec % 60
    return `Áudio (${m}:${String(s).padStart(2, '0')}) — expirado`
  }
  return `${props.message.media_filename || mediaTypeLabel.value} — expirado`
})

const formattedTime = computed(() => {
  if (!props.message.timestamp) return ''
  try {
    return new Intl.DateTimeFormat('pt-BR', {
      hour: '2-digit',
      minute: '2-digit',
      timeZone: 'America/Sao_Paulo',
    }).format(new Date(props.message.timestamp))
  } catch {
    return ''
  }
})
</script>

<style scoped>
.msg-row {
  display: flex;
  justify-content: flex-start;
  width: 100%;
  margin-bottom: 0.25rem;
}
.msg-row.from-me {
  justify-content: flex-end;
}
.msg-bubble {
  max-width: 82%;
  min-width: 80px;
  padding: 0.5rem 0.65rem 0.35rem 0.75rem;
  border-radius: 8px 8px 8px 2px;
  background: var(--surface-2);
  border: 1px solid var(--border-soft);
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.15);
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  position: relative;
}
.msg-bubble.bubble-me {
  border-radius: 8px 8px 2px 8px;
  background: var(--brand-soft);
  border-color: var(--brand-border);
}
.msg-text {
  font-size: var(--fs-md);
  color: var(--text-1);
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
  line-height: 1.45;
}
.msg-footer {
  display: inline-flex;
  align-items: center;
  align-self: flex-end;
  gap: 0.25rem;
  margin-top: 0.1rem;
}
.msg-time {
  font-size: var(--fs-2xs);
  color: var(--text-3);
  font-family: var(--font-mono);
  user-select: none;
}
.msg-check {
  color: var(--brand);
  display: block;
}
.chat-img {
  max-width: 240px;
  max-height: 240px;
  border-radius: 6px;
  display: block;
  object-fit: cover;
}
.doc-pill {
  display: inline-flex;
  align-items: center;
  gap: 0.45rem;
  padding: 0.45rem 0.65rem;
  border-radius: 6px;
  background: var(--surface-deep);
  border: 1px solid var(--border);
  color: var(--brand);
  text-decoration: none;
  font-size: var(--fs-sm);
  font-weight: var(--fw-medium);
}
.doc-name {
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.media-notice {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.35rem 0.55rem;
  border-radius: 6px;
  background: var(--surface-deep);
  border: 1px dashed var(--border);
  color: var(--text-3);
  font-size: var(--fs-xs);
}
.media-notice.expired {
  color: var(--text-2);
}
</style>
