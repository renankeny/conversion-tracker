<template>
  <Teleport to="body">
    <Transition name="fade-bg">
      <div v-if="modelValue && item" class="drawer-backdrop" @click="close" />
    </Transition>

    <Transition name="slide-right">
      <aside v-if="modelValue && item" class="inspect-drawer" role="dialog" aria-modal="true">
        <header class="drawer-head">
          <div class="lead-id-row">
            <div class="lead-avatar">{{ initials }}</div>
            <div class="lead-meta">
              <div class="name-line">
                <h2 class="lead-title">{{ item.lead_name }}</h2>
                <span class="sla-pill" :class="slaStatusMetaOf(item.sla_status).statusClass">
                  {{ slaStatusMetaOf(item.sla_status).label }}
                </span>
              </div>
              <div class="sub-line">
                <span class="font-mono">{{ item.phone || 'Sem telefone' }}</span>
                <button v-if="item.phone" type="button" class="copy-btn" title="Copiar telefone" @click="copyPhone">
                  <Check v-if="copied" :size="12" />
                  <Copy v-else :size="12" />
                </button>
                <span v-if="item.instance_name">• {{ item.instance_name }}</span>
              </div>
            </div>
          </div>

          <div class="head-actions">
            <NuxtLink
              v-if="crmPath"
              :to="crmPath"
              class="crm-link-btn"
              @click="close"
            >
              <span>Ver no CRM</span>
              <ArrowUpRight :size="13" />
            </NuxtLink>
            <button type="button" class="close-btn" aria-label="Fechar" @click="close">
              <X :size="18" />
            </button>
          </div>
        </header>

        <div class="sla-strip">
          <div class="sla-cell">
            <span class="sla-lbl">Origem</span>
            <span class="sla-val">{{ originMetaOf(item.origin_category).label }}</span>
          </div>
          <div class="sla-cell">
            <span class="sla-lbl">Etapa CRM</span>
            <span class="sla-val">{{ item.stage_name || 'Sem etapa' }}</span>
          </div>
          <div class="sla-cell">
            <span class="sla-lbl">1ª Resposta</span>
            <span class="sla-val font-mono" :class="responseToneClass(item.first_response_sec)">
              {{ formatDurationSLA(item.first_response_sec) }}
            </span>
          </div>
          <div class="sla-cell">
            <span class="sla-lbl">T. Médio</span>
            <span class="sla-val font-mono" :class="responseToneClass(item.avg_response_sec)">
              {{ formatDurationSLA(item.avg_response_sec) }}
            </span>
          </div>
        </div>

        <div v-if="item.tags?.length" class="tags-row">
          <span
            v-for="tag in item.tags"
            :key="tag.id"
            class="tag-chip"
            :style="{ backgroundColor: tag.color + '18', color: tag.color, borderColor: tag.color + '40' }"
          >
            {{ tag.name }}
          </span>
        </div>

        <div class="chat-body">
          <WhatsAppChatPanel
            :chat-id="item.chat_id"
            :crm-lead-id="item.crm_lead_id"
            :tracked-user-id="item.tracked_user_id"
            :public-token="publicToken"
          />
        </div>
      </aside>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { X, Copy, Check, ArrowUpRight } from 'lucide-vue-next'
import WhatsAppChatPanel from '~/components/whatsapp/WhatsAppChatPanel.vue'
import type { ConversationAuditItem } from '#shared/types'
import { formatDurationSLA, originMetaOf, responseToneClass, slaStatusMetaOf } from '~/utils/conversationFormatters'

const props = defineProps<{
  modelValue: boolean
  item: ConversationAuditItem | null
  publicToken?: string
  showPublicCrmLink?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [v: boolean]
}>()

const copied = ref(false)
const close = () => emit('update:modelValue', false)

const initials = computed(() => (props.item?.lead_name || 'L').trim().slice(0, 2).toUpperCase())

const crmPath = computed(() => {
  if (!props.item?.crm_lead_id) return ''
  if (props.publicToken) {
    return props.showPublicCrmLink ? `/cliente/${encodeURIComponent(props.publicToken)}/crm` : ''
  }
  return '/crm'
})

const copyPhone = async () => {
  if (!props.item?.phone) return
  try {
    await navigator.clipboard.writeText(props.item.phone)
    copied.value = true
    setTimeout(() => { copied.value = false }, 2000)
  } catch {}
}

const onKeydown = (e: KeyboardEvent) => {
  if (e.key === 'Escape' && props.modelValue) close()
}

onMounted(() => { if (import.meta.client) window.addEventListener('keydown', onKeydown) })
onUnmounted(() => { if (import.meta.client) window.removeEventListener('keydown', onKeydown) })
</script>

<style scoped>
.drawer-backdrop { position: fixed; inset: 0; background: rgba(0, 0, 0, 0.55); backdrop-filter: blur(3px); z-index: 70; }
.inspect-drawer {
  position: fixed; top: 0; right: 0; bottom: 0; width: 100%; max-width: 540px; z-index: 75;
  background: var(--surface-2); border-left: 1px solid var(--border);
  display: flex; flex-direction: column; box-shadow: var(--shadow-elevated);
}
.drawer-head {
  padding: 1rem 1.15rem; border-bottom: 1px solid var(--border-soft);
  display: flex; align-items: flex-start; justify-content: space-between; gap: 0.75rem;
}
.lead-id-row { display: flex; align-items: center; gap: 0.75rem; min-width: 0; }
.lead-avatar {
  width: 38px; height: 38px; border-radius: 9px; background: var(--brand-soft); color: var(--brand);
  font-size: var(--fs-md); font-weight: var(--fw-bold); display: flex; align-items: center; justify-content: center; flex-shrink: 0;
}
.lead-meta { min-width: 0; }
.name-line { display: flex; align-items: center; gap: 0.45rem; flex-wrap: wrap; }
.lead-title { font-size: var(--fs-lg); font-weight: var(--fw-bold); color: var(--text-strong); margin: 0; }
.sub-line { display: flex; align-items: center; gap: 0.35rem; font-size: var(--fs-xs); color: var(--text-3); margin-top: 2px; }
.copy-btn { background: none; border: none; color: var(--text-3); cursor: pointer; display: inline-flex; padding: 2px; }
.copy-btn:hover { color: var(--brand); }
.head-actions { display: flex; align-items: center; gap: 0.45rem; flex-shrink: 0; }
.crm-link-btn {
  display: inline-flex; align-items: center; gap: 0.25rem; padding: 0.35rem 0.6rem; border-radius: 7px;
  border: 1px solid var(--brand-border); background: var(--brand-soft); color: var(--brand);
  font-size: var(--fs-xs); font-weight: var(--fw-semibold); text-decoration: none;
}
.close-btn {
  width: 32px; height: 32px; border-radius: 8px; border: 1px solid var(--border-soft);
  background: var(--surface-deep); color: var(--text-2); display: flex; align-items: center; justify-content: center; cursor: pointer;
}
.sla-pill { padding: 0.12rem 0.45rem; border-radius: 999px; font-size: var(--fs-2xs); font-weight: var(--fw-semibold); }
.sla-replied { background: var(--brand-soft); color: var(--brand); }
.sla-waiting { background: var(--warning-soft); color: var(--warning); }
.sla-unanswered { background: var(--danger-soft); color: var(--danger); }
.sla-strip {
  display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 0.5rem;
  padding: 0.65rem 1.15rem; background: var(--surface-deep); border-bottom: 1px solid var(--border-soft);
}
.sla-cell { display: flex; flex-direction: column; gap: 2px; min-width: 0; }
.sla-lbl { font-size: var(--fs-2xs); color: var(--text-3); text-transform: uppercase; letter-spacing: 0.04em; }
.sla-val { font-size: var(--fs-sm); font-weight: var(--fw-semibold); color: var(--text-1); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.tone-good { color: var(--status-ok); }
.tone-warn { color: var(--warning); }
.tone-bad { color: var(--danger); }
.tone-muted { color: var(--text-3); }
.tags-row { display: flex; gap: 0.35rem; flex-wrap: wrap; padding: 0.55rem 1.15rem 0; }
.tag-chip { padding: 0.15rem 0.45rem; border-radius: 5px; font-size: var(--fs-xs); font-weight: var(--fw-semibold); border: 1px solid transparent; }
.chat-body { flex: 1; min-height: 0; padding: 0.85rem 1.15rem 1.15rem; display: flex; flex-direction: column; overflow: hidden; }
.chat-body :deep(.chat-viewport-wrapper), .chat-body :deep(.messages-viewport) { max-height: none; flex: 1; }
.fade-bg-enter-active, .fade-bg-leave-active { transition: opacity 0.2s ease; }
.fade-bg-enter-from, .fade-bg-leave-to { opacity: 0; }
.slide-right-enter-active, .slide-right-leave-active { transition: transform 0.22s cubic-bezier(0.16, 1, 0.3, 1); }
.slide-right-enter-from, .slide-right-leave-to { transform: translateX(100%); }
</style>
