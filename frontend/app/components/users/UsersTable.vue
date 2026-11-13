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
                <code v-if="user.last_utm_source">{{ truncate(decodeUTM(user.last_utm_source), 18) }}</code>
                <span v-if="user.last_utm_campaign" class="utm-campaign">{{ truncate(decodeUTM(user.last_utm_campaign), 16) }}</span>
              </span>
              <span v-else class="muted">—</span>
            </td>
            <td class="time">{{ formatTime(user.updated_at) }}</td>
            <td class="action-cell">
              <button class="row-arrow" aria-label="Ver detalhes">
                <ChevronRight :size="16" />
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ChevronRight } from 'lucide-vue-next'
import type { TrackedUser } from '#shared/types'
import { formatShortDateTimeSP } from '~/utils/formatDate'

defineProps<{
  users: TrackedUser[]
  loading: boolean
}>()

defineEmits<{
  select: [user: TrackedUser]
}>()

const decodeUTM = (val: string | null | undefined): string => {
  if (!val) return ''
  try {
    return decodeURIComponent(val.replace(/\+/g, ' '))
  } catch {
    return val.replace(/\+/g, ' ')
  }
}

const truncate = (s: string, n: number): string =>
  s && s.length > n ? s.slice(0, n) + '…' : s || ''

const userName = (u: TrackedUser): string =>
  [u.first_name, u.last_name].filter(Boolean).join(' ').trim()

const hasGeo = (u: TrackedUser): boolean => Boolean(u.country || u.state || u.city)
const formatGeo = (u: TrackedUser): string => [u.city, u.state, u.country].filter(Boolean).join(', ')

const countryFlag = (country: string): string => {
  if (!country || country.length !== 2) return '🌐'
  const offset = 0x1f1a5
  const upper = country.toUpperCase()
  return String.fromCodePoint(upper.charCodeAt(0) + offset, upper.charCodeAt(1) + offset)
}

const formatTime = (iso: string): string => formatShortDateTimeSP(iso)
</script>

<style scoped>
.users-table {
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: 12px;
  box-shadow: var(--shadow-card);
  overflow: hidden;
}

.table-wrap { overflow-x: auto; }
.table { width: 100%; border-collapse: collapse; min-width: 760px; }

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

.table td {
  padding: 0.75rem 1rem;
  font-size: var(--fs-md);
  color: var(--text-1);
  border-bottom: 1px solid var(--border-soft);
  vertical-align: middle;
}

.row { cursor: pointer; transition: background 0.12s; }
.row:hover { background: var(--surface-soft); }
.row:last-child td { border-bottom: none; }

.ext-id {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: var(--fs-sm);
  background: var(--surface-soft);
  padding: 0.125rem 0.5rem;
  border-radius: 6px;
  color: var(--text-2);
}

.pixel { font-weight: var(--fw-semibold); }
.identity { display: flex; flex-direction: column; gap: 0.125rem; }
.identity-name { font-weight: var(--fw-semibold); color: var(--text-1); }
.identity-line { font-size: var(--fs-sm); color: var(--text-2); font-family: ui-monospace, SFMono-Regular, Menlo, monospace; }
.geo { display: inline-flex; align-items: center; gap: 0.375rem; font-size: var(--fs-base); }
.flag { font-size: var(--fs-lg); }
.utm { display: flex; flex-direction: column; gap: 0.125rem; }
.utm code { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: var(--fs-xs); color: var(--text-2); }
.utm-campaign { font-size: var(--fs-xs); color: var(--text-3); }
.time { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: var(--fs-sm); color: var(--text-2); white-space: nowrap; }
.muted { color: var(--text-3); font-size: var(--fs-base); font-style: italic; }
.action-cell { text-align: right; }

.row-arrow {
  background: transparent;
  border: none;
  color: var(--text-3);
  padding: 0.25rem;
  cursor: pointer;
  border-radius: 4px;
}

.row:hover .row-arrow { color: var(--brand); background: var(--brand-soft); }
.state-row { display: flex; flex-direction: column; align-items: center; justify-content: center; padding: 3rem 1rem; gap: 0.75rem; color: var(--text-3); }
.empty-msg { font-style: italic; font-size: var(--fs-md); }
.spinner { width: 1.5rem; height: 1.5rem; border: 3px solid var(--border); border-top-color: var(--brand); border-radius: 50%; animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
</style>
