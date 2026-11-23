<template>
  <div class="table-card">
    <div v-if="loading && !items.length" class="state-empty">
      <Loader2 :size="26" class="spin icon-brand" />
      <span>Carregando auditoria de conversas...</span>
    </div>

    <div v-else-if="!items.length" class="state-empty">
      <MessageSquare :size="26" class="icon-muted" />
      <p class="empty-title">Nenhuma conversa encontrada neste período</p>
      <p class="empty-sub">
        Assim que visitantes de anúncios, site ou links rastreáveis iniciarem contato no WhatsApp, o histórico e o SLA aparecerão aqui.
      </p>
    </div>

    <template v-else>
      <div class="table-scroll">
        <table class="audit-table">
          <thead>
            <tr>
              <th class="sortable" @click="$emit('sort', 'last_message_time')">
                <span>DATA</span>
                <ArrowUpDown :size="11" class="sort-icon" :class="{ active: sortBy === 'last_message_time' }" />
              </th>
              <th>LEAD</th>
              <th>ORIGEM</th>
              <th>ETAPA</th>
              <th class="sortable text-right" @click="$emit('sort', 'lead_value')">
                <span>VENDA</span>
                <ArrowUpDown :size="11" class="sort-icon" :class="{ active: sortBy === 'lead_value' }" />
              </th>
              <th>STATUS</th>
              <th class="sortable text-center" @click="$emit('sort', 'first_response_sec')">
                <span>1ª RESP.</span>
                <ArrowUpDown :size="11" class="sort-icon" :class="{ active: sortBy === 'first_response_sec' }" />
              </th>
              <th class="sortable text-center" @click="$emit('sort', 'last_inbound_at')">
                <span>ÚLT. LEAD</span>
                <ArrowUpDown :size="11" class="sort-icon" :class="{ active: sortBy === 'last_inbound_at' }" />
              </th>
              <th class="sortable text-center" @click="$emit('sort', 'last_outbound_at')">
                <span>ÚLT. ATEND.</span>
                <ArrowUpDown :size="11" class="sort-icon" :class="{ active: sortBy === 'last_outbound_at' }" />
              </th>
              <th class="sortable text-center" @click="$emit('sort', 'avg_response_sec')">
                <span>T. RESP.</span>
                <ArrowUpDown :size="11" class="sort-icon" :class="{ active: sortBy === 'avg_response_sec' }" />
              </th>
              <th class="sortable text-center" @click="$emit('sort', 'total_messages')">
                <span>MSGS</span>
                <ArrowUpDown :size="11" class="sort-icon" :class="{ active: sortBy === 'total_messages' }" />
              </th>
              <th class="action-col" />
            </tr>
          </thead>
          <tbody>
            <ConversationAuditRow
              v-for="row in items"
              :key="row.chat_id"
              :row="row"
              :blurred="blurred"
              @inspect="$emit('inspect', $event)"
            />
          </tbody>
        </table>
      </div>

      <div v-if="totalPages > 1" class="pagination-bar">
        <span class="page-info">
          Mostrando {{ (page - 1) * pageSize + 1 }} a {{ Math.min(page * pageSize, total) }} de {{ total }}
        </span>
        <div class="page-btns">
          <BaseButton variant="outline" size="sm" :disabled="page <= 1" @click="$emit('update:page', page - 1)">
            Anterior
          </BaseButton>
          <BaseButton variant="outline" size="sm" :disabled="page >= totalPages" @click="$emit('update:page', page + 1)">
            Próxima
          </BaseButton>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { Loader2, MessageSquare, ArrowUpDown } from 'lucide-vue-next'
import ConversationAuditRow from '~/components/conversations/ConversationAuditRow.vue'
import type { ConversationAuditItem } from '#shared/types'

const props = defineProps<{
  items: ConversationAuditItem[]
  total: number
  loading: boolean
  sortBy: string
  sortDir: 'asc' | 'desc'
  page: number
  pageSize: number
  blurred?: boolean
}>()

defineEmits<{
  inspect: [item: ConversationAuditItem]
  sort: [col: string]
  'update:page': [page: number]
}>()

const totalPages = computed(() => Math.max(1, Math.ceil(props.total / props.pageSize)))
</script>

<style scoped>
.table-card { background: var(--surface-2); border: 1px solid var(--border-soft); border-radius: 12px; overflow: hidden; }
.table-scroll { overflow-x: auto; }
.audit-table { width: 100%; border-collapse: collapse; min-width: 980px; }
.audit-table th {
  padding: 0.7rem 0.85rem; text-align: left; font-size: var(--fs-2xs); font-weight: var(--fw-bold);
  color: var(--text-3); letter-spacing: 0.05em; border-bottom: 1px solid var(--border-soft);
  background: var(--surface-deep); white-space: nowrap; user-select: none;
}
.audit-table th.sortable { cursor: pointer; }
.audit-table th.sortable:hover { color: var(--text-1); }
.sort-icon { display: inline-block; vertical-align: middle; margin-left: 3px; opacity: 0.4; }
.sort-icon.active { opacity: 1; color: var(--brand); }
.text-right { text-align: right !important; }
.text-center { text-align: center !important; }
.action-col { width: 44px; text-align: center !important; }
.state-empty { padding: 3.5rem 1.25rem; display: flex; flex-direction: column; align-items: center; gap: 0.5rem; text-align: center; color: var(--text-3); }
.empty-title { font-size: var(--fs-lg); font-weight: var(--fw-semibold); color: var(--text-1); margin: 0; }
.empty-sub { font-size: var(--fs-base); color: var(--text-3); max-width: 420px; margin: 0; }
.icon-brand { color: var(--brand); }
.icon-muted { color: var(--text-3); opacity: 0.6; }
.spin { animation: spin 1s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
.pagination-bar { display: flex; align-items: center; justify-content: space-between; padding: 0.75rem 1rem; border-top: 1px solid var(--border-soft); background: var(--surface-deep); gap: 0.75rem; flex-wrap: wrap; }
.page-info { font-size: var(--fs-sm); color: var(--text-3); }
.page-btns { display: flex; gap: 0.5rem; }
</style>
