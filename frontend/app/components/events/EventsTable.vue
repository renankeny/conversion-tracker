<template>
  <div class="events-table">
    <div v-if="loading && !events.length" class="state-row">
      <div class="spinner" />
      <p>Carregando eventos...</p>
    </div>

    <div v-else-if="!events.length" class="state-row">
      <p class="empty-msg">Nenhum evento encontrado com esses filtros.</p>
    </div>

    <div v-else class="table-wrap">
      <table class="table">
        <thead>
          <tr>
            <th v-if="canManage" class="th-check">
              <BaseCheckbox
                size="sm"
                :checked="allSelected"
                :indeterminate="someSelected && !allSelected"
                @change="toggleAll"
              />
            </th>
            <th>Hora</th>
            <th>Pixel</th>
            <th>Evento</th>
            <th>Id do Usuário</th>
            <th>Fonte 1º Acesso</th>
            <th>Fonte Último Acesso</th>
            <th>Canais</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="ev in events" :key="rowKey(ev)" class="row" :class="{ selected: isSelected(ev) }">
            <td v-if="canManage" class="td-check" @click.stop>
              <BaseCheckbox size="sm" :checked="isSelected(ev)" @change="toggleOne(ev)" />
            </td>
            <td class="time" @click="$emit('select', ev)">{{ formatTime(ev.created_at) }}</td>
            <td class="pixel" @click="$emit('select', ev)">{{ ev.pixel_name }}</td>
            <td @click="$emit('select', ev)">
              <EventRowBadges :event="ev" variant="event" />
            </td>
            <td @click="$emit('select', ev)">
              <div v-if="ev.external_id" class="ext-id-wrap">
                <code class="ext-id" :title="ev.external_id">{{ truncate(ev.external_id, 14) }}</code>
                <button
                  type="button"
                  class="btn-copy-id"
                  :class="{ copied: copiedKey === rowKey(ev) }"
                  :title="copiedKey === rowKey(ev) ? 'ID copiado' : 'Copiar ID do usuário'"
                  @click.stop="copyExtId(ev)"
                >
                  <Check v-if="copiedKey === rowKey(ev)" :size="12" />
                  <Copy v-else :size="12" />
                </button>
              </div>
              <span v-else class="source-cell">-</span>
            </td>
            <td class="source-cell" @click="$emit('select', ev)">{{ sourceLabel(ev.first_utm_source) }}</td>
            <td class="source-cell" @click="$emit('select', ev)">{{ sourceLabel(ev.last_utm_source) }}</td>
            <td @click="$emit('select', ev)">
              <EventRowBadges :event="ev" variant="channels" />
            </td>
            <td class="action-cell" @click="$emit('select', ev)">
              <div class="row-actions" @click.stop>
                <button
                  v-if="canManage && hasFailed(ev)"
                  type="button"
                  class="btn-retry"
                  title="Reenviar evento"
                  @click="$emit('retry', [toKeyRef(ev)])"
                >
                  <RefreshCw :size="14" />
                </button>
                <button type="button" class="row-arrow" aria-label="Ver detalhes" @click="$emit('select', ev)">
                  <ChevronRight :size="16" />
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { Check, ChevronRight, Copy, RefreshCw } from 'lucide-vue-next'
import EventRowBadges from '~/components/events/EventRowBadges.vue'
import type { EventKeyRef, GroupedEvent } from '#shared/types'
import { formatShortDateTimeSP } from '~/utils/formatDate'

const props = withDefaults(defineProps<{
  events: GroupedEvent[]
  loading: boolean
  canManage?: boolean
  selectedItems?: EventKeyRef[]
}>(), {
  canManage: false,
  selectedItems: () => [],
})

const emit = defineEmits<{
  select: [event: GroupedEvent]
  selectionChange: [items: EventKeyRef[]]
  retry: [items: EventKeyRef[]]
}>()

const rowKey = (ev: Pick<GroupedEvent, 'event_id' | 'pixel_name'>): string => `${ev.pixel_name}::${ev.event_id}`
const refKey = (item: EventKeyRef): string => `${item.pixel_name || ''}::${item.event_id}`
const toKeyRef = (ev: GroupedEvent): EventKeyRef => ({ event_id: ev.event_id, pixel_name: ev.pixel_name })

const selectedMap = computed(() => {
  const map = new Map<string, EventKeyRef>()
  for (const item of props.selectedItems) map.set(refKey(item), item)
  return map
})

const isSelected = (ev: GroupedEvent) => selectedMap.value.has(rowKey(ev))
const allSelected = computed(() => props.events.length > 0 && props.events.every((ev) => selectedMap.value.has(rowKey(ev))))
const someSelected = computed(() => props.events.some((ev) => selectedMap.value.has(rowKey(ev))))

const toggleOne = (ev: GroupedEvent) => {
  const next = new Map(selectedMap.value)
  const key = rowKey(ev)
  if (next.has(key)) next.delete(key)
  else next.set(key, toKeyRef(ev))
  emit('selectionChange', [...next.values()])
}

const toggleAll = () => {
  emit('selectionChange', allSelected.value ? [] : props.events.map(toKeyRef))
}

watch(() => props.events, (nextEvents) => {
  if (!props.selectedItems.length) return
  const validKeys = new Set(nextEvents.map(rowKey))
  const filtered = props.selectedItems.filter((item) => validKeys.has(refKey(item)))
  if (filtered.length !== props.selectedItems.length) emit('selectionChange', filtered)
})

const copiedKey = ref('')
const copyExtId = async (ev: GroupedEvent) => {
  if (!ev.external_id) return
  try {
    await navigator.clipboard.writeText(ev.external_id)
    const key = rowKey(ev)
    copiedKey.value = key
    setTimeout(() => { if (copiedKey.value === key) copiedKey.value = '' }, 1800)
  } catch { /* silent */ }
}

const hasFailed = (ev: GroupedEvent): boolean =>
  ev.channels.some((c) => c.channel !== 'internal' && (c.has_error || c.status === 'failed'))

const formatTime = (iso: string): string => formatShortDateTimeSP(iso, true)
const truncate = (s: string, n: number): string => (s && s.length > n ? s.slice(0, n) + '…' : s || '')

const sourceLabel = (source: string | undefined): string => {
  if (!source) return '-'
  const val = (source.split('|')[0] || '').trim()
  if (!val) return '-'
  try {
    return decodeURIComponent(val.replace(/\+/g, ' '))
  } catch {
    return val.replace(/\+/g, ' ')
  }
}
</script>

<style scoped>
.events-table { background: var(--surface-2); border: 1px solid var(--border); border-radius: 12px; box-shadow: var(--shadow-card); overflow: hidden; }
.table-wrap { overflow-x: auto; }
.table { width: 100%; border-collapse: collapse; min-width: 1080px; }
.table th { background: var(--surface-soft-2); color: var(--text-3); font-size: var(--fs-xs); font-weight: var(--fw-semibold); text-transform: uppercase; letter-spacing: 0.05em; text-align: left; padding: 0.625rem 1rem; border-bottom: 1px solid var(--border); }
.th-check { width: 40px; padding: 0.625rem 0.75rem; }
.table td { padding: 0.75rem 1rem; font-size: var(--fs-md); color: var(--text-1); border-bottom: 1px solid var(--border-soft); vertical-align: middle; cursor: pointer; }
.td-check { width: 40px; padding: 0.75rem 0.75rem; cursor: default; }
.row { transition: background 0.12s; }
.row:hover { background: var(--surface-soft); }
.row.selected { background: var(--brand-soft); }
.row:last-child td { border-bottom: none; }
.time { font-family: var(--font-mono); font-size: var(--fs-sm); color: var(--text-2); white-space: nowrap; }
.pixel { font-weight: var(--fw-semibold); }
.ext-id-wrap { display: inline-flex; align-items: center; gap: 0.3rem; }
.ext-id { font-family: var(--font-mono); font-size: var(--fs-sm); background: var(--surface-soft); padding: 0.0625rem 0.375rem; border-radius: 4px; color: var(--text-2); }
.btn-copy-id { display: inline-flex; align-items: center; justify-content: center; padding: 0.1875rem; border-radius: 4px; border: 1px solid transparent; background: transparent; color: var(--text-3); cursor: pointer; opacity: 0.65; transition: all 0.15s; }
.row:hover .btn-copy-id { opacity: 1; }
.btn-copy-id:hover { color: var(--brand); background: var(--surface-soft); border-color: var(--border-soft); }
.btn-copy-id.copied { opacity: 1; color: var(--success); }
.source-cell { font-size: var(--fs-base); color: var(--text-2); white-space: nowrap; }
.action-cell { width: 70px; }
.row-actions { display: flex; align-items: center; gap: 0.25rem; justify-content: flex-end; }
.btn-retry { background: var(--warning-soft); border: 1px solid var(--warning); color: var(--warning); cursor: pointer; padding: 0.25rem; border-radius: 5px; display: flex; align-items: center; transition: all 0.15s; }
.btn-retry:hover { background: var(--warning); color: var(--text-on-solid); }
.row-arrow { background: none; border: none; color: var(--text-3); cursor: pointer; padding: 0; display: flex; align-items: center; }
.state-row { display: flex; flex-direction: column; align-items: center; justify-content: center; padding: 3rem 1rem; gap: 0.75rem; color: var(--text-3); }
.empty-msg { font-size: var(--fs-lg); }
.spinner { width: 24px; height: 24px; border: 2px solid var(--border); border-top-color: var(--brand); border-radius: 50%; animation: spin 0.7s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
</style>
