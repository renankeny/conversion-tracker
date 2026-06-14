<template>
  <div class="users-table">
    <div v-if="loading && !users.length" class="state-row">
      <div class="spinner" />
      <p>Carregando usuários...</p>
    </div>

    <div v-else-if="!users.length" class="state-row">
      <p class="empty-msg">Nenhum usuário encontrado.</p>
    </div>

    <div v-else class="table-wrap">
      <table class="table">
        <thead>
          <tr>
            <th>External ID</th>
            <th>Pixel</th>
            <th>Identificação</th>
            <th>Geo</th>
            <th>Última UTM</th>
            <th>Atualizado</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="user in users"
            :key="user.id"
            class="row"
            @click="$emit('select', user)"
          >
            <td>
              <code class="ext-id">{{ truncate(user.external_id, 16) }}</code>
            </td>
            <td class="pixel">{{ user.pixel_name || '—' }}</td>
            <td>
              <div class="identity">
                <span v-if="userName(user)" class="identity-name">{{ userName(user) }}</span>
                <span v-if="user.email" class="identity-line">{{ user.email }}</span>
                <span v-if="user.phone" class="identity-line">{{ user.phone }}</span>
                <span v-if="!userName(user) && !user.email && !user.phone" class="muted">Anônimo</span>
              </div>
            </td>
            <td>
              <span v-if="hasGeo(user)" class="geo">
                <span class="flag">{{ countryFlag(user.country) }}</span>
                {{ formatGeo(user) }}
              </span>
              <span v-else class="muted">—</span>
            </td>
            <td>
              <span v-if="user.last_utm_source || user.last_utm_campaign" class="utm">
                <code v-if="user.last_utm_source">{{ truncate(user.last_utm_source, 18) }}</code>
                <span v-if="user.last_utm_campaign" class="utm-campaign">{{ truncate(user.last_utm_campaign, 16) }}</span>
              </span>
              <span v-else class="muted">—</span>
            </td>
            <td class="time">{{ formatTime(user.updated_at) }}</td>
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
import type { TrackedUser } from '#shared/types'

defineProps<{
  users: TrackedUser[]
  loading: boolean
}>()

defineEmits<{
  select: [user: TrackedUser]
}>()

const truncate = (s: string, n: number): string =>
  s && s.length > n ? s.slice(0, n) + '…' : s || ''

const userName = (u: TrackedUser): string => {
  const parts = [u.first_name, u.last_name].filter(Boolean)
  return parts.join(' ').trim()
}

const hasGeo = (u: TrackedUser): boolean =>
  Boolean(u.country || u.state || u.city)

const formatGeo = (u: TrackedUser): string => {
  const parts = [u.city, u.state, u.country].filter(Boolean)
  return parts.join(', ')
}

const countryFlag = (country: string): string => {
  if (!country || country.length !== 2) return '🌐'
  const offset = 0x1f1a5
  const upper = country.toUpperCase()
  return String.fromCodePoint(
    upper.charCodeAt(0) + offset,
    upper.charCodeAt(1) + offset
  )
}

const formatTime = (iso: string): string => {
  if (!iso) return ''
  return new Date(iso).toLocaleString('pt-BR', {
    day: '2-digit',
    month: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}
</script>

<style scoped>
.users-table {
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
  min-width: 760px;
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

.ext-id {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.75rem;
  background: var(--surface-soft);
  padding: 0.125rem 0.5rem;
  border-radius: 6px;
  color: var(--text-2);
}

.pixel {
  font-weight: 600;
}

.identity {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
}

.identity-name {
  font-weight: 600;
  color: var(--text-1);
}

.identity-line {
  font-size: 0.75rem;
  color: var(--text-2);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
}

.geo {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  font-size: 0.8125rem;
}

.flag {
  font-size: 1rem;
}

.utm {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
}

.utm code {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.7rem;
  color: var(--text-2);
}

.utm-campaign {
  font-size: 0.7rem;
  color: var(--text-3);
}

.time {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.75rem;
  color: var(--text-2);
  white-space: nowrap;
}

.muted {
  color: var(--text-3);
  font-size: 0.8125rem;
  font-style: italic;
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
