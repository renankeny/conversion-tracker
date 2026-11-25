<template>
  <div class="msg-row" :class="{ 'from-me': message.is_from_me }">
    <div class="msg-bubble" :class="{ 'bubble-me': message.is_from_me }">
      <template v-if="isMediaMessage">
        <!-- Mídia pronta para reprodução / visualização -->
        <div v-if="message.media_status === 'ready' && message.media_stream_url" class="media-ready">
          <template v-if="message.message_type === 'audio'">
            <div class="media-kind">
              <Mic :size="11" />
              <span>Áudio</span>
            </div>
            <WhatsAppAudioPlayer
              :src="message.media_stream_url"
              :initial-duration="message.media_duration_sec"
            />
          </template>

          <template v-else-if="message.message_type === 'video'">
            <div class="media-kind">
              <Video :size="11" />
              <span>Vídeo{{ formattedDuration ? ` · ${formattedDuration}` : '' }}</span>
            </div>
            <video
              :src="message.media_stream_url"
              controls
              preload="metadata"
              playsinline
              class="chat-video"
            />
          </template>

          <template v-else-if="message.message_type === 'image' || message.message_type === 'sticker'">
            <div class="media-kind">
              <component :is="mediaIcon" :size="11" />
              <span>{{ mediaTypeLabel }}</span>
            </div>
            <a
              :href="message.media_stream_url"
              target="_blank"
              rel="noopener"
              class="img-link"
            >
              <img :src="message.media_stream_url" :alt="mediaTypeLabel" class="chat-img" loading="lazy">
            </a>
          </template>

          <a
            v-else
            :href="message.media_stream_url"
            target="_blank"
            rel="noopener"
            class="doc-pill"
          >
            <FileText :size="16" class="doc-icon" />
            <div class="doc-meta">
              <span class="doc-name">{{ message.media_filename || 'Documento' }}</span>
              <span class="doc-sub">Documento{{ formattedSize ? ` · ${formattedSize}` : '' }}</span>
            </div>
          </a>
        </div>

        <!-- Mídia em processamento assíncrono -->
        <div v-else-if="message.media_status === 'pending'" class="media-notice">
          <Clock :size="13" class="media-icon" />
          <span>Sincronizando {{ mediaSummary.toLowerCase() }}...</span>
        </div>

        <!-- Mídia expirada após janela de retenção -->
        <div v-else-if="message.media_status === 'expired'" class="media-notice expired">
          <component :is="mediaIcon" :size="13" class="media-icon" />
          <span>{{ mediaSummary }} — expirado</span>
        </div>

        <!-- Mídia acima do limite de armazenamento -->
        <div v-else-if="message.media_status === 'skipped_size'" class="media-notice">
          <component :is="mediaIcon" :size="13" class="media-icon" />
          <span>{{ mediaSummary }} acima do limite</span>
        </div>

        <!-- Marcador claro do tipo de mídia quando sem storage / indisponível -->
        <div v-else class="media-notice marked">
          <component :is="mediaIcon" :size="13" class="media-icon brand" />
          <span>{{ mediaSummary }}</span>
        </div>
      </template>

      <p v-if="message.content" class="msg-text">{{ message.content }}</p>

      <div class="msg-footer">
        <span class="msg-time">{{ formattedTime }}</span>
        <CheckCheck v-if="message.is_from_me" :size="14" class="msg-check" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import {
  FileText, Clock, CheckCheck, Mic, Video, Image as ImageIcon, Smile,
} from 'lucide-vue-next'
import type { WhatsAppMessage } from '#shared/types'
import WhatsAppAudioPlayer from '~/components/whatsapp/WhatsAppAudioPlayer.vue'

const props = defineProps<{
  message: WhatsAppMessage
}>()

const isMediaMessage = computed(() => (
  props.message.has_media
  || ['audio', 'image', 'video', 'document', 'sticker'].includes(props.message.message_type)
))

const mediaTypeLabel = computed(() => {
  switch (props.message.message_type) {
    case 'audio': return 'Áudio'
    case 'image': return 'Foto'
    case 'video': return 'Vídeo'
    case 'document': return 'Documento'
    case 'sticker': return 'Figurinha'
    default: return 'Mídia'
  }
})

const mediaIcon = computed(() => {
  switch (props.message.message_type) {
    case 'audio': return Mic
    case 'image': return ImageIcon
    case 'video': return Video
    case 'sticker': return Smile
    default: return FileText
  }
})

const formattedDuration = computed(() => {
  const sec = Math.floor(props.message.media_duration_sec || 0)
  if (sec <= 0) return ''
  const m = Math.floor(sec / 60)
  const s = sec % 60
  return `${m}:${String(s).padStart(2, '0')}`
})

const formattedSize = computed(() => {
  const b = props.message.media_size_bytes || 0
  if (b <= 0) return ''
  if (b >= 1024 * 1024) return `${(b / (1024 * 1024)).toFixed(1)} MB`
  return `${Math.max(1, Math.round(b / 1024))} KB`
})

const mediaSummary = computed(() => {
  if (props.message.message_type === 'document' && props.message.media_filename) {
    return `Documento · ${props.message.media_filename}`
  }
  if (formattedDuration.value && (props.message.message_type === 'audio' || props.message.message_type === 'video')) {
    return `${mediaTypeLabel.value} (${formattedDuration.value})`
  }
  return mediaTypeLabel.value
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
.msg-row { display: flex; justify-content: flex-start; width: 100%; margin-bottom: 0.25rem; }
.msg-row.from-me { justify-content: flex-end; }
.msg-bubble {
  max-width: 82%; min-width: 84px; padding: 0.5rem 0.65rem 0.35rem 0.75rem;
  border-radius: 8px 8px 8px 2px; background: var(--surface-2); border: 1px solid var(--border-soft);
  display: flex; flex-direction: column; gap: 0.28rem; position: relative;
}
.msg-bubble.bubble-me {
  border-radius: 8px 8px 2px 8px; background: var(--brand-soft); border-color: var(--brand-border);
}
.media-ready { display: flex; flex-direction: column; gap: 0.25rem; }
.media-kind {
  display: inline-flex; align-items: center; gap: 0.3rem;
  color: var(--text-3); font-size: var(--fs-2xs); font-weight: var(--fw-semibold);
}
.msg-text {
  font-size: var(--fs-md); color: var(--text-1); margin: 0;
  white-space: pre-wrap; word-break: break-word; line-height: 1.45;
}
.msg-footer { display: inline-flex; align-items: center; align-self: flex-end; gap: 0.25rem; margin-top: 0.1rem; }
.msg-time { font-size: var(--fs-2xs); color: var(--text-3); font-family: var(--font-mono); user-select: none; }
.msg-check { color: var(--brand); display: block; }
.chat-img { max-width: 240px; max-height: 240px; border-radius: 6px; display: block; object-fit: cover; }
.chat-video { max-width: 260px; max-height: 260px; border-radius: 6px; display: block; background: var(--surface-deep); }
.doc-pill {
  display: inline-flex; align-items: center; gap: 0.5rem; padding: 0.45rem 0.65rem;
  border-radius: 6px; background: var(--surface-deep); border: 1px solid var(--border);
  color: var(--text-1); text-decoration: none;
}
.doc-icon { color: var(--brand); flex-shrink: 0; }
.doc-meta { display: flex; flex-direction: column; min-width: 0; }
.doc-name {
  max-width: 200px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
  font-size: var(--fs-sm); font-weight: var(--fw-medium); color: var(--brand);
}
.doc-sub { font-size: var(--fs-2xs); color: var(--text-3); }
.media-notice {
  display: inline-flex; align-items: center; gap: 0.4rem; padding: 0.35rem 0.55rem;
  border-radius: 6px; background: var(--surface-deep); border: 1px dashed var(--border);
  color: var(--text-2); font-size: var(--fs-xs); font-weight: var(--fw-medium);
}
.media-notice.marked { border-style: solid; border-color: var(--border-soft); color: var(--text-1); }
.media-notice.expired { color: var(--text-3); }
.media-icon { flex-shrink: 0; color: var(--text-3); }
.media-icon.brand { color: var(--brand); }
</style>
