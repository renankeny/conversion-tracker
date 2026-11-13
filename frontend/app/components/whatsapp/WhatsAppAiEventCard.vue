<template>
  <div class="ai-event-wrap">
    <div class="ai-event-card" :class="[`kind-${event.event_type}`, { contested: isContested }]">
      <div class="ai-head">
        <div class="ai-title-group">
          <Sparkles :size="14" class="ai-icon" />
          <span class="ai-title">{{ titleText }}</span>
          <span v-if="formattedAmount" class="ai-amount">{{ formattedAmount }}</span>
        </div>

        <div class="ai-badges">
          <span v-if="isContested" class="status-tag tag-contested">Contestado</span>
          <span v-else-if="event.deduplicated" class="status-tag tag-dedup" title="A conversão já havia sido enviada por gatilho ou webhook, evitando duplicidade.">
            Sem duplicar
          </span>
          <span v-else-if="event.dispatched_event_id" class="status-tag tag-sent">
            Evento enviado
          </span>
        </div>
      </div>

      <p v-if="event.ai_reason" class="ai-reason">{{ event.ai_reason }}</p>

      <p v-if="isContested && event.contest_reason" class="contest-note">
        <strong>Motivo da contestação:</strong> {{ event.contest_reason }}
      </p>

      <div class="ai-foot">
        <span class="ai-time">{{ timeLabel }}</span>
        <button
          v-if="!isContested && canContest"
          type="button"
          class="contest-btn"
          @click="$emit('contest', event)"
        >
          <Flag :size="12" />
          <span>Contestar</span>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Sparkles, Flag } from 'lucide-vue-next'
import type { WhatsAppAIEvent } from '#shared/types'
import { formatDateTimeSP } from '~/utils/formatDate'

const props = defineProps<{
  event: WhatsAppAIEvent
  canContest?: boolean
}>()

defineEmits<{
  contest: [ev: WhatsAppAIEvent]
}>()

const isContested = computed(() => props.event.status === 'contested')

const titleText = computed(() => {
  if (props.event.event_type === 'purchase') {
    return isContested.value ? 'Venda pela IA (revertida)' : 'Venda identificada pela IA'
  }
  return isContested.value ? 'Qualificação pela IA (revertida)' : 'Qualificado pela IA'
})

const formattedAmount = computed(() => {
  if (props.event.event_type !== 'purchase' || !props.event.amount || props.event.amount <= 0) return ''
  try {
    return new Intl.NumberFormat('pt-BR', {
      style: 'currency',
      currency: props.event.currency || 'BRL',
    }).format(props.event.amount)
  } catch {
    return `${props.event.currency || 'BRL'} ${props.event.amount.toFixed(2)}`
  }
})

const timeLabel = computed(() => formatDateTimeSP(props.event.created_at))
</script>

<style scoped>
.ai-event-wrap { display: flex; justify-content: center; margin: 0.45rem 0; width: 100%; }
.ai-event-card { width: 100%; max-width: 460px; padding: 0.65rem 0.85rem; border-radius: 10px; background: var(--surface-2); border: 1px solid var(--brand-border); display: flex; flex-direction: column; gap: 0.35rem; }
.ai-event-card.kind-purchase { border-color: var(--success); }
.ai-event-card.contested { border-color: var(--border); opacity: 0.72; }
.ai-head { display: flex; align-items: center; justify-content: space-between; gap: 0.5rem; flex-wrap: wrap; }
.ai-title-group { display: inline-flex; align-items: center; gap: 0.4rem; min-width: 0; }
.ai-icon { color: var(--brand); flex-shrink: 0; }
.kind-purchase .ai-icon { color: var(--success); }
.contested .ai-icon { color: var(--text-3); }
.ai-title { font-size: var(--fs-sm); font-weight: var(--fw-semibold); color: var(--text-strong); }
.contested .ai-title { text-decoration: line-through; color: var(--text-2); }
.ai-amount { font-family: var(--font-mono); font-size: var(--fs-xs); font-weight: var(--fw-semibold); color: var(--success); padding: 0.1rem 0.4rem; border-radius: 6px; background: var(--surface-deep); }
.ai-badges { display: inline-flex; align-items: center; gap: 0.35rem; }
.status-tag { padding: 0.12rem 0.45rem; border-radius: 999px; font-size: var(--fs-2xs); font-weight: var(--fw-semibold); border: 1px solid var(--border-soft); background: var(--surface-deep); }
.tag-sent { color: var(--success); }
.tag-dedup { color: var(--text-2); }
.tag-contested { color: var(--danger); }
.ai-reason { margin: 0; font-size: var(--fs-xs); color: var(--text-2); line-height: 1.45; }
.contest-note { margin: 0; padding: 0.35rem 0.5rem; border-radius: 6px; background: var(--surface-deep); font-size: var(--fs-xs); color: var(--warning); line-height: 1.4; }
.ai-foot { display: flex; align-items: center; justify-content: space-between; gap: 0.5rem; margin-top: 0.1rem; }
.ai-time { font-size: var(--fs-2xs); color: var(--text-3); }
.contest-btn { display: inline-flex; align-items: center; gap: 0.3rem; background: transparent; border: 1px solid var(--border); color: var(--text-2); font-size: var(--fs-2xs); font-weight: var(--fw-semibold); padding: 0.18rem 0.5rem; border-radius: 6px; cursor: pointer; transition: all 0.15s ease; }
.contest-btn:hover { border-color: var(--danger); color: var(--danger); background: var(--surface-soft); }
</style>

