<template>
  <tr class="audit-row" @click="$emit('inspect', row)">
    <td class="date-cell font-mono">{{ formatAuditDateTime(row.last_message_time) }}</td>

    <td>
      <div class="lead-cell">
        <div class="lead-avatar">{{ initials }}</div>
        <div class="lead-texts">
          <div class="lead-name-row">
            <span class="lead-name">{{ row.lead_name }}</span>
            <span v-if="row.ai_qualified" class="ai-pill" title="Qualificado pela IA">IA</span>
          </div>
          <span class="lead-phone font-mono">{{ row.phone || 'Sem número' }}</span>
        </div>
      </div>
    </td>

    <td>
      <div class="origin-cell">
        <span class="origin-badge" :class="originMeta.badgeClass">
          <MetaAdsIcon v-if="row.origin_category === 'meta_ads'" :size="11" />
          <GoogleAdsIcon v-else-if="row.origin_category === 'google_ads'" :size="11" />
          <span>{{ originMeta.label }}</span>
        </span>
        <span v-if="originSubtitle" class="origin-sub" :title="originSubtitle">
          {{ originSubtitle }}
        </span>
      </div>
    </td>

    <td>
      <span v-if="row.stage_name" class="stage-pill">
        <span class="stage-dot" :style="{ backgroundColor: row.stage_color || 'var(--brand)' }" />
        <span>{{ row.stage_name }}</span>
      </span>
      <span v-else class="muted-dash">-</span>
    </td>

    <td class="text-right">
      <span v-if="row.lead_value > 0" class="sale-val" :class="{ 'is-won': row.is_won, blurred }">
        {{ formatAuditMoney(row.lead_value, row.currency) }}
      </span>
      <span v-else class="muted-dash">-</span>
    </td>

    <td>
      <span class="sla-badge" :class="slaMeta.statusClass">
        {{ slaMeta.label }}
      </span>
    </td>

    <td class="text-center font-mono" :class="responseToneClass(row.first_response_sec)">
      {{ formatDurationSLA(row.first_response_sec) }}
    </td>

    <td class="text-center font-mono" :class="inboundWaitToneClass(row)">
      {{ formatRelativeElapsed(row.last_inbound_at) }}
    </td>

    <td class="text-center font-mono tone-neutral">
      {{ formatRelativeElapsed(row.last_outbound_at) }}
    </td>

    <td class="text-center font-mono" :class="responseToneClass(row.avg_response_sec)">
      {{ formatDurationSLA(row.avg_response_sec) }}
    </td>

    <td class="text-center font-mono msgs-col">{{ row.total_messages }}</td>

    <td class="action-col" @click.stop="$emit('inspect', row)">
      <button type="button" class="inspect-btn" title="Ver histórico da conversa">
        <Eye :size="15" />
      </button>
    </td>
  </tr>
</template>

<script setup lang="ts">
import { Eye } from 'lucide-vue-next'
import { MetaAdsIcon, GoogleAdsIcon } from '~/utils/adIcons'
import type { ConversationAuditItem } from '#shared/types'
import {
  formatAuditDateTime, formatDurationSLA, formatAuditMoney, formatRelativeElapsed,
  inboundWaitToneClass, originMetaOf, responseToneClass, slaStatusMetaOf,
} from '~/utils/conversationFormatters'

const props = defineProps<{
  row: ConversationAuditItem
  blurred?: boolean
}>()

defineEmits<{
  inspect: [item: ConversationAuditItem]
}>()

const initials = computed(() => (props.row.lead_name || 'L').trim().slice(0, 2).toUpperCase())
const originMeta = computed(() => originMetaOf(props.row.origin_category))
const slaMeta = computed(() => slaStatusMetaOf(props.row.sla_status))
const originSubtitle = computed(() => props.row.campaign_name || props.row.trackable_link_name || '')
</script>

<style scoped>
.audit-row { border-bottom: 1px solid var(--border-soft); cursor: pointer; transition: background 0.12s ease; }
.audit-row:last-child { border-bottom: none; }
.audit-row:hover { background: var(--surface-soft); }
.audit-row td { padding: 0.72rem 0.85rem; font-size: var(--fs-base); color: var(--text-2); vertical-align: middle; }
.text-right { text-align: right; }
.text-center { text-align: center; }
.date-cell { font-size: var(--fs-sm); color: var(--text-2); white-space: nowrap; }
.lead-cell { display: flex; align-items: center; gap: 0.6rem; min-width: 170px; }
.lead-avatar {
  width: 30px; height: 30px; border-radius: 8px; background: var(--surface-deep); border: 1px solid var(--border);
  color: var(--brand); font-size: var(--fs-xs); font-weight: var(--fw-bold); display: flex; align-items: center; justify-content: center; flex-shrink: 0;
}
.lead-texts { display: flex; flex-direction: column; min-width: 0; }
.lead-name-row { display: flex; align-items: center; gap: 0.35rem; }
.lead-name { font-size: var(--fs-md); font-weight: var(--fw-semibold); color: var(--text-1); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; max-width: 180px; }
.lead-phone { font-size: var(--fs-xs); color: var(--text-3); }
.ai-pill { padding: 0.05rem 0.35rem; border-radius: 999px; background: var(--brand-soft); border: 1px solid var(--brand-border); color: var(--brand); font-size: var(--fs-2xs); font-weight: var(--fw-bold); }
.origin-cell { display: flex; flex-direction: column; align-items: flex-start; gap: 0.18rem; max-width: 180px; }
.origin-badge { display: inline-flex; align-items: center; gap: 0.3rem; padding: 0.18rem 0.5rem; border-radius: 6px; font-size: var(--fs-xs); font-weight: var(--fw-semibold); border: 1px solid var(--border-soft); white-space: nowrap; }
.origin-meta { background: var(--accent-sky-soft); color: var(--accent-sky); border-color: var(--accent-sky-border); }
.origin-google { background: var(--warning-soft); color: var(--warning); border-color: var(--warning-border); }
.origin-tiktok { background: var(--accent-pink-soft); color: var(--accent-pink); border-color: var(--accent-pink-border); }
.origin-bio { background: var(--accent-purple-soft); color: var(--accent-purple); border-color: var(--accent-purple-border); }
.origin-site { background: var(--brand-soft); color: var(--brand); border-color: var(--brand-border); }
.origin-sub { font-size: var(--fs-2xs); color: var(--text-3); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; max-width: 170px; }
.stage-pill { display: inline-flex; align-items: center; gap: 0.35rem; padding: 0.2rem 0.55rem; border-radius: 6px; background: var(--surface-deep); border: 1px solid var(--border); font-size: var(--fs-xs); font-weight: var(--fw-semibold); color: var(--text-1); white-space: nowrap; }
.stage-dot { width: 7px; height: 7px; border-radius: 50%; flex-shrink: 0; }
.sale-val { font-weight: var(--fw-semibold); color: var(--text-1); white-space: nowrap; }
.sale-val.is-won { color: var(--status-ok); }
.sale-val.blurred { filter: blur(6px); user-select: none; }
.sla-badge { display: inline-flex; padding: 0.18rem 0.5rem; border-radius: 999px; font-size: var(--fs-xs); font-weight: var(--fw-semibold); white-space: nowrap; }
.sla-replied { background: var(--brand-soft); color: var(--brand); }
.sla-waiting { background: var(--warning-soft); color: var(--warning); }
.sla-unanswered { background: var(--danger-soft); color: var(--danger); }
.tone-good { color: var(--status-ok) !important; font-weight: var(--fw-semibold); }
.tone-warn { color: var(--warning) !important; font-weight: var(--fw-semibold); }
.tone-bad { color: var(--danger) !important; font-weight: var(--fw-semibold); }
.tone-neutral { color: var(--text-2); }
.tone-muted, .muted-dash { color: var(--text-3); }
.msgs-col { color: var(--text-1); font-weight: var(--fw-semibold); }
.action-col { width: 44px; text-align: center; }
.inspect-btn {
  width: 30px; height: 30px; border-radius: 7px; border: 1px solid var(--border-soft);
  background: var(--surface-deep); color: var(--text-2); display: inline-flex; align-items: center; justify-content: center; cursor: pointer;
}
.inspect-btn:hover { color: var(--brand); border-color: var(--brand-border); background: var(--brand-soft); }
</style>
