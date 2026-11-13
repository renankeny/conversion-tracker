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
            <th class="th-check">
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
          <tr
            v-for="ev in events"
            :key="`${ev.event_id}-${ev.pixel_name}`"
            class="row"
            :class="{ selected: isSelected(ev) }"
          >
            <td class="td-check" @click.stop>
              <BaseCheckbox
                size="sm"
                :checked="isSelected(ev)"
                @change="toggleOne(ev)"
              />
            </td>
            <td class="time" @click="$emit('select', ev)">{{ formatTime(ev.created_at) }}</td>
            <td class="pixel" @click="$emit('select', ev)">{{ ev.pixel_name }}</td>
            <td @click="$emit('select', ev)">
              <EventRowBadges :event="ev" variant="event" />
            </td>
            <td @click="$emit('select', ev)">
              <code class="ext-id">{{ truncate(ev.external_id, 14) }}</code>
            </td>
            <td class="source-cell" @click="$emit('select', ev)">
              {{ sourceLabel(ev.first_utm_source) }}
            </td>
            <td class="source-cell" @click="$emit('select', ev)">
              {{ sourceLabel(ev.last_utm_source) }}
            </td>
            <td @click="$emit('select', ev)">
              <EventRowBadges :event="ev" variant="channels" />
            </td>
            <td class="action-cell" @click="$emit('select', ev)">
              <div class="row-actions" @click.stop>
                <button
                  v-if="hasFailed(ev)"
                  class="btn-retry"
                  title="Reenviar evento"
                  @click="$emit('retry', [ev.event_id])"
                >
                  <RefreshCw :size="14" />
                </button>
                <button class="row-arrow" aria-label="Ver detalhes" @click="$emit('select', ev)">
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
import { ChevronRight, RefreshCw } from 'lucide-vue-next'
import EventRowBadges from '~/components/events/EventRowBadges.vue'
import type { GroupedEvent } from '#shared/types'
import { formatShortDateTimeSP } from '~/utils/formatDate'

const props = defineProps<{
  events: GroupedEvent[]
  loading: boolean
}>()

const emit = defineEmits<{
  select: [event: GroupedEvent]
  selectionChange: [eventIds: string[]]
  retry: [eventIds: string[]]
}>()

const selected = ref<Set<string>>(new Set())

const isSelected = (ev: GroupedEvent) => selected.value.has(ev.event_id)

const allSelected = computed(() =>
  props.events.length > 0 && props.events.every(ev => selected.value.has(ev.event_id))
)
const someSelected = computed(() =>
  props.events.some(ev => selected.value.has(ev.event_id))
)

const toggleOne = (ev: GroupedEvent) => {
  const next = new Set(selected.value)
  if (next.has(ev.event_id)) next.delete(ev.event_id)
  else next.add(ev.event_id)
  selected.value = next
  emit('selectionChange', [...next])
}

const toggleAll = () => {
  selected.value = allSelected.value
    ? new Set()
    : new Set(props.events.map(ev => ev.event_id))
  emit('selectionChange', [...selected.value])
}

watch(() => props.events, () => {
  selected.value = new Set()
  emit('selectionChange', [])
})

const hasFailed = (ev: GroupedEvent): boolean =>
  ev.channels.some(c => c.has_error || c.status === 'failed')

const formatTime = (iso: string): string => formatShortDateTimeSP(iso, true)

const truncate = (s: string, n: number): string =>
  s && s.length > n ? s.slice(0, n) + '…' : s || ''

const sourceLabel = (source: string | undefined): string => {
  if (!source) return '—'
  const [label] = source.split('|')
  return label || '—'
}
</script>

<style scoped>
.events-table {
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: 12px;
  box-shadow: var(--shadow-card);
  overflow: hidden;
}

.table-wrap { overflow-x: auto; }
.table { width: 100%; border-collapse: collapse; min-width: 1080px; }

.table th {
  background: var(--surface-soft-2);
  color: var(--text-3);
  font-size: var(--fs-xs);
  font-weight: var(--fw-semibold);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  text-align: left;
  padding: 0.625rem 1rem;
  border-bottom: 1px solid var(--border);
}

.th-check { width: 40px; padding: 0.625rem 0.75rem; }

.table td {
  padding: 0.75rem 1rem;
  font-size: var(--fs-md);
  color: var(--text-1);
  border-bottom: 1px solid var(--border-soft);
  vertical-align: middle;
  cursor: pointer;
}

.td-check { width: 40px; padding: 0.75rem 0.75rem; cursor: default; }
.row { transition: background 0.12s; }
.row:hover { background: var(--surface-soft); }
.row.selected { background: var(--brand-soft); }
.row:last-child td { border-bottom: none; }
.time { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: var(--fs-sm); color: var(--text-2); white-space: nowrap; }
.pixel { font-weight: var(--fw-semibold); }

.ext-id {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: var(--fs-sm);
  background: var(--surface-soft);
  padding: 0.0625rem 0.375rem;
  border-radius: 4px;
  color: var(--text-2);
}

.source-cell { font-size: var(--fs-base); color: var(--text-2); white-space: nowrap; }
.action-cell { width: 70px; }
.row-actions { display: flex; align-items: center; gap: 0.25rem; justify-content: flex-end; }

.btn-retry {
  background: var(--warning-soft);
  border: 1px solid var(--warning);
  color: var(--warning);
  cursor: pointer;
  padding: 0.25rem;
  border-radius: 5px;
  display: flex;
  align-items: center;
  transition: all 0.15s;
}

.btn-retry:hover { background: var(--warning); color: var(--text-on-solid); }
.row-arrow { background: none; border: none; color: var(--text-3); cursor: pointer; padding: 0; display: flex; align-items: center; }
.state-row { display: flex; flex-direction: column; align-items: center; justify-content: center; padding: 3rem 1rem; gap: 0.75rem; color: var(--text-3); }
.empty-msg { font-size: var(--fs-lg); }
.spinner { width: 24px; height: 24px; border: 2px solid var(--border); border-top-color: var(--brand); border-radius: 50%; animation: spin 0.7s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
</style>
