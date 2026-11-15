<template>
  <div class="users-table">
    <div v-if="loading && !users.length" class="state-row">
      <div class="spinner" />
      <p>Carregando usuários...</p>
    </div>

    <div v-else-if="!users.length" class="state-row">
      <p class="empty-msg">Nenhum usuário encontrado com os filtros selecionados.</p>
    </div>

    <div v-else class="table-wrap">
      <table class="table">
        <thead>
          <tr>
            <th>Visitante / Contato</th>
            <th>Tráfego</th>
            <th>LTV</th>
            <th>Pixel</th>
            <th>Localização</th>
            <th>Última Origem</th>
            <th>Atividade</th>
            <th />
          </tr>
        </thead>
        <tbody>
          <tr v-for="user in users" :key="user.id" class="row" @click="$emit('select', user)">
            <td>
              <div class="visitor-cell">
                <div class="visitor-avatar" :class="{ 'is-anon': isAnonymousUser(user) }">
                  <User v-if="isAnonymousUser(user)" :size="15" />
                  <template v-else>{{ userInitials(user) }}</template>
                </div>
                <div class="visitor-info">
                  <span class="visitor-primary" :class="{ 'is-anon': isAnonymousUser(user) }">
                    {{ userPrimaryLabel(user) }}
                  </span>
                  <div class="visitor-secondary">
                    <span v-if="userSecondaryLine(user)" class="visitor-contact">
                      {{ userSecondaryLine(user) }}
                    </span>
                    <button
                      type="button"
                      class="quick-copy-btn"
                      :class="{ copied: copiedKey === `contact-${user.id}` }"
                      :title="copiedKey === `contact-${user.id}` ? 'Copiado!' : userQuickCopyTarget(user).label"
                      @click.stop="copyValue(userQuickCopyTarget(user).value, `contact-${user.id}`)"
                    >
                      <Check v-if="copiedKey === `contact-${user.id}`" :size="11" />
                      <Copy v-else :size="11" />
                    </button>
                    <code
                      class="id-badge"
                      :class="{ copied: copiedKey === `id-${user.id}` }"
                      title="Clique para copiar ID"
                      @click.stop="copyValue(user.external_id, `id-${user.id}`)"
                    >
                      ID: {{ truncateText(user.external_id, 10) }}
                    </code>
                  </div>
                </div>
              </div>
            </td>

            <td>
              <span class="traffic-pill" :class="trafficPillClass(user.traffic_type)">
                {{ trafficPillLabel(user.traffic_type) }}
              </span>
            </td>

            <td>
              <span v-if="(user.ltv || 0) > 0" class="ltv-positive">{{ formatCurrencyBRL(user.ltv) }}</span>
              <span v-else class="muted">-</span>
            </td>

            <td>
              <span v-if="user.pixel_name" class="pixel-text">{{ user.pixel_name }}</span>
              <span v-else class="muted">-</span>
            </td>

            <td>
              <span v-if="hasUserGeo(user)" class="geo">
                <span class="flag">{{ countryFlag(user.country) }}</span>
                <span>{{ formatUserGeo(user) }}</span>
              </span>
              <span v-else class="muted">-</span>
            </td>

            <td>
              <div v-if="!resolveUserOrigin(user).isEmpty" class="origin-cell">
                <span class="origin-source" :class="{ 'is-direct': resolveUserOrigin(user).isDirect }">
                  {{ resolveUserOrigin(user).primary }}
                </span>
                <span v-if="resolveUserOrigin(user).campaign" class="origin-campaign">
                  {{ resolveUserOrigin(user).campaign }}
                </span>
              </div>
              <span v-else class="muted">-</span>
            </td>

            <td class="time">{{ formatShortDateTimeSP(user.updated_at) || '-' }}</td>

            <td class="action-cell">
              <button type="button" class="row-arrow" aria-label="Ver detalhes">
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
import { ref, onBeforeUnmount } from 'vue'
import { ChevronRight, Copy, Check, User } from 'lucide-vue-next'
import {
  countryFlag, formatCurrencyBRL, formatUserGeo, hasUserGeo, isAnonymousUser, resolveUserOrigin,
  trafficPillClass, trafficPillLabel, truncateText, userInitials, userPrimaryLabel, userQuickCopyTarget, userSecondaryLine,
} from '~/components/users/userFormatters'
import type { TrackedUser } from '#shared/types'
import { formatShortDateTimeSP } from '~/utils/formatDate'

defineProps<{ users: TrackedUser[]; loading: boolean }>()
defineEmits<{ select: [user: TrackedUser] }>()

const copiedKey = ref<string | null>(null)
let copyTimer: ReturnType<typeof setTimeout> | null = null

const copyValue = async (text: string, key: string) => {
  if (!text) return
  try {
    await navigator.clipboard.writeText(text)
    copiedKey.value = key
    if (copyTimer) clearTimeout(copyTimer)
    copyTimer = setTimeout(() => {
      if (copiedKey.value === key) copiedKey.value = null
    }, 2000)
  } catch {
    /* silent */
  }
}

onBeforeUnmount(() => {
  if (copyTimer) clearTimeout(copyTimer)
})
</script>

<style scoped>
.users-table { background: var(--surface-2); border: 1px solid var(--border); border-radius: 12px; box-shadow: var(--shadow-card); overflow: hidden; }
.table-wrap { overflow-x: auto; }
.table { width: 100%; border-collapse: collapse; min-width: 880px; }
.table th { background: var(--surface-soft-2); color: var(--text-3); font-size: var(--fs-xs); font-weight: var(--fw-semibold); text-transform: uppercase; letter-spacing: 0.05em; text-align: left; padding: 0.625rem 0.875rem; border-bottom: 1px solid var(--border); white-space: nowrap; }
.table td { padding: 0.6875rem 0.875rem; font-size: var(--fs-base); color: var(--text-1); border-bottom: 1px solid var(--border-soft); vertical-align: middle; }
.row { cursor: pointer; transition: background 0.12s; }
.row:hover { background: var(--surface-soft); }
.row:last-child td { border-bottom: none; }

.visitor-cell { display: flex; align-items: center; gap: 0.6875rem; min-width: 220px; }
.visitor-avatar { width: 34px; height: 34px; border-radius: 8px; flex-shrink: 0; display: flex; align-items: center; justify-content: center; background: var(--brand-soft); border: 1px solid var(--brand-border); color: var(--brand); font-size: var(--fs-sm); font-weight: var(--fw-bold); }
.visitor-avatar.is-anon { background: var(--surface-soft); border-color: var(--border-soft); color: var(--text-3); }
.visitor-info { display: flex; flex-direction: column; gap: 0.125rem; min-width: 0; }
.visitor-primary { font-size: var(--fs-md); font-weight: var(--fw-semibold); color: var(--text-1); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; max-width: 240px; }
.visitor-primary.is-anon { color: var(--text-2); font-weight: var(--fw-medium); }
.visitor-secondary { display: flex; align-items: center; flex-wrap: wrap; gap: 0.35rem; min-width: 0; }
.visitor-contact { font-family: var(--font-mono); font-size: var(--fs-xs); color: var(--text-2); max-width: 240px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

.quick-copy-btn { display: inline-flex; align-items: center; justify-content: center; padding: 0.125rem; border-radius: 4px; background: transparent; border: none; color: var(--text-3); cursor: pointer; transition: color 0.15s, background 0.15s; }
.quick-copy-btn:hover { color: var(--brand); background: var(--surface-soft); }
.quick-copy-btn.copied { color: var(--success); }

.id-badge { font-family: var(--font-mono); font-size: var(--fs-2xs); color: var(--text-3); background: var(--surface-deep); border: 1px solid var(--border-soft); padding: 0.0625rem 0.35rem; border-radius: 4px; white-space: nowrap; transition: border-color 0.15s, color 0.15s; }
.id-badge:hover { border-color: var(--brand-border); color: var(--text-2); }
.id-badge.copied { border-color: var(--success); color: var(--success); }

.traffic-pill { display: inline-flex; align-items: center; padding: 0.125rem 0.5rem; border-radius: 9999px; font-size: var(--fs-xs); font-weight: var(--fw-semibold); white-space: nowrap; }
.traffic-pill.is-human { background: var(--brand-soft); color: var(--brand); border: 1px solid var(--brand-border); }
.traffic-pill.is-bot { background: var(--danger-soft); color: var(--danger-light); border: 1px solid var(--danger-border); }
.traffic-pill.is-inconclusive { background: var(--warning-soft); color: var(--warning-light); border: 1px solid var(--warning-border); }

.ltv-positive { font-family: var(--font-mono); font-size: var(--fs-md); font-weight: var(--fw-semibold); color: var(--success); white-space: nowrap; }
.pixel-text { font-weight: var(--fw-medium); color: var(--text-1); white-space: nowrap; }
.geo { display: inline-flex; align-items: center; gap: 0.35rem; font-size: var(--fs-sm); color: var(--text-2); white-space: nowrap; }
.flag { font-size: var(--fs-md); }
.origin-cell { display: flex; flex-direction: column; gap: 0.0625rem; min-width: 0; }
.origin-source { font-family: var(--font-mono); font-size: var(--fs-xs); font-weight: var(--fw-medium); color: var(--text-1); white-space: nowrap; }
.origin-source.is-direct { font-family: var(--font-sans); color: var(--text-2); }
.origin-campaign { font-size: var(--fs-2xs); color: var(--text-3); white-space: nowrap; }
.time { font-family: var(--font-mono); font-size: var(--fs-sm); color: var(--text-2); white-space: nowrap; }
.muted { color: var(--text-3); font-size: var(--fs-sm); }
.action-cell { text-align: right; width: 36px; }
.row-arrow { background: transparent; border: none; color: var(--text-3); padding: 0.25rem; cursor: pointer; border-radius: 4px; }
.row:hover .row-arrow { color: var(--brand); background: var(--brand-soft); }
.state-row { display: flex; flex-direction: column; align-items: center; justify-content: center; padding: 3rem 1rem; gap: 0.75rem; color: var(--text-3); }
.empty-msg { font-size: var(--fs-md); }
.spinner { width: 1.5rem; height: 1.5rem; border: 3px solid var(--border); border-top-color: var(--brand); border-radius: 50%; animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
</style>
