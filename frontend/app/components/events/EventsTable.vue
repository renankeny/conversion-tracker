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
            <th>Hora</th>
            <th>Pixel</th>
            <th>Evento</th>
            <th>External ID</th>
            <th>Status</th>
            <th>HTTP</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="ev in events"
            :key="ev.id"
            class="row"
            @click="$emit('select', ev)"
          >
            <td class="time">{{ formatTime(ev.created_at) }}</td>
            <td class="pixel">{{ ev.pixel_name }}</td>
            <td>
              <span class="event-name" :class="eventClass(ev.event_name)">
                {{ ev.event_name }}
              </span>
            </td>
            <td>
              <code class="ext-id">{{ truncate(ev.external_id, 14) }}</code>
            </td>
            <td>
              <BaseBadge :variant="badgeVariant(ev.status)">
                {{ statusLabel(ev.status) }}
              </BaseBadge>
            </td>
            <td>
              <span class="http-code" :class="httpClass(ev.response_code)">
                {{ ev.response_code || '—' }}
              </span>
            </td>
            <td class="action-cell">
              <button class="row-arrow" aria-label="Ver detalhes">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
                  <path fill-rule="evenodd" d="M7.293 14.707a1 1 0 010-1.414L10.586 10 7.293 6.707a1 1 0 011.414-1.414l4 4a1 1 0 010 1.414l-4 4a1 1 0 01-1.414 0z" clip-rule="evenodd" />
                </svg>
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { EventStatus, SentEvent } from '#shared/types'

defineProps<{
  events: SentEvent[]
  loading: boolean
}>()

defineEmits<{
  select: [event: SentEvent]
}>()

const formatTime = (iso: string): string => {
  if (!iso) return ''
  const d = new Date(iso)
  return d.toLocaleString('pt-BR', {
    day: '2-digit',
    month: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}

const truncate = (s: string, n: number): string =>
  s && s.length > n ? s.slice(0, n) + '…' : s || ''

const badgeVariant = (status: EventStatus): 'success' | 'warning' | 'danger' => {
  if (status === 'sent') return 'success'
  if (status === 'pending') return 'warning'
  return 'danger'
}

const statusLabel = (status: EventStatus): string => {
  if (status === 'sent') return 'Enviado'
  if (status === 'pending') return 'Pendente'
  return 'Falhou'
}

const httpClass = (code: number): string => {
  if (!code) return ''
  if (code >= 200 && code < 300) return 'http-ok'
  if (code >= 400 && code < 500) return 'http-warn'
  if (code >= 500) return 'http-err'
  return ''
}

const importantEvents = new Set([
  'Lead',
  'InitiateCheckout',
  'Purchase',
  'AddPaymentInfo',
  'CompleteRegistration',
  'AddToCart',
  'Contact',
])

const eventClass = (name: string): string => {
  if (importantEvents.has(name)) return 'event-important'
  return 'event-default'
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

.table-wrap {
  overflow-x: auto;
}

.table {
  width: 100%;
  border-collapse: collapse;
  min-width: 720px;
}

.table th {
  background: var(--surface-soft-2);
  color: var(--text-3);
  font-size: 0.6875rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  text-align: left;
  padding: 0.625rem 1rem;
  border-bottom: 1px solid var(--border);
}

.table td {
  padding: 0.75rem 1rem;
  font-size: 0.875rem;
  color: var(--text-1);
  border-bottom: 1px solid var(--border-soft);
  vertical-align: middle;
}

.row {
  cursor: pointer;
  transition: background 0.12s;
}

.row:hover {
  background: var(--surface-soft);
}

.row:last-child td {
  border-bottom: none;
}

.time {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.75rem;
  color: var(--text-2);
  white-space: nowrap;
}

.pixel {
  font-weight: 600;
}

.event-name {
  font-size: 0.8125rem;
  font-weight: 600;
}

.event-important {
  color: var(--success);
}

.event-default {
  color: var(--text-2);
}

.ext-id {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.75rem;
  background: var(--surface-soft);
  padding: 0.0625rem 0.375rem;
  border-radius: 4px;
  color: var(--text-2);
}

.http-code {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.8125rem;
  font-weight: 600;
}

.http-ok {
  color: var(--success);
}

.http-warn {
  color: var(--warning);
}

.http-err {
  color: var(--danger);
}

.action-cell {
  text-align: right;
}

.row-arrow {
  background: transparent;
  border: none;
  color: var(--text-3);
  padding: 0.25rem;
  cursor: pointer;
  border-radius: 4px;
  transition: all 0.15s;
}

.row:hover .row-arrow {
  color: var(--brand);
  background: var(--brand-soft);
}

.state-row {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 3rem 1rem;
  gap: 0.75rem;
  color: var(--text-3);
}

.empty-msg {
  font-style: italic;
  font-size: 0.875rem;
}

.spinner {
  width: 1.5rem;
  height: 1.5rem;
  border: 3px solid var(--border);
  border-top-color: var(--brand);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
