<template>
  <div v-if="user" class="user-attribution-tab">
    <div class="attribution-grid">
      <!-- Coluna 1: Primeiro Acesso (Descoberta) -->
      <div class="attribution-card">
        <div class="attribution-card-header">
          <span class="attr-badge first">Primeiro Acesso (Descoberta)</span>
          <span class="attr-date">{{ formatDate(user.created_at) }}</span>
        </div>

        <div class="attr-body">
          <div class="attr-url-block">
            <span class="attr-url-label">Primeira URL Acessada</span>
            <input
              v-if="editing"
              v-model="form.first_url"
              type="url"
              class="url-edit"
              placeholder="https://..."
            />
            <a
              v-else-if="safeExternalUrl(user.first_url || user.url)"
              :href="safeExternalUrl(user.first_url || user.url)"
              target="_blank"
              rel="noopener noreferrer"
              class="url-link"
              :title="user.first_url || user.url"
            >
              <span class="url-text">{{ user.first_url || user.url }}</span>
              <ExternalLink class="w-3.5 h-3.5 flex-shrink-0" />
            </a>
            <span v-else-if="user.first_url || user.url" class="url-text text-[var(--text-2)]">
              {{ user.first_url || user.url }}
            </span>
            <span v-else class="muted">—</span>
          </div>

          <div class="utm-subgrid">
            <UserField label="Source" v-model="form.first_utm_source" :editing="editing" :original="decodeUTM(user.first_utm_source)" mono small />
            <UserField label="Medium" v-model="form.first_utm_medium" :editing="editing" :original="decodeUTM(user.first_utm_medium)" mono small />
            <UserField label="Campaign" v-model="form.first_utm_campaign" :editing="editing" :original="decodeUTM(user.first_utm_campaign)" mono small />
            <UserField label="Content" v-model="form.first_utm_content" :editing="editing" :original="decodeUTM(user.first_utm_content)" mono small />
            <UserField label="Term" v-model="form.first_utm_term" :editing="editing" :original="decodeUTM(user.first_utm_term)" mono small />
          </div>
        </div>
      </div>

      <!-- Coluna 2: Último Acesso (Conversão) -->
      <div class="attribution-card">
        <div class="attribution-card-header">
          <span class="attr-badge last">Último Acesso (Conversão)</span>
          <span class="attr-date">{{ formatDate(user.updated_at) }}</span>
        </div>

        <div class="attr-body">
          <div class="attr-url-block">
            <span class="attr-url-label">Última URL Acessada</span>
            <input
              v-if="editing"
              v-model="form.url"
              type="url"
              class="url-edit"
              placeholder="https://..."
            />
            <a
              v-else-if="safeExternalUrl(user.url)"
              :href="safeExternalUrl(user.url)"
              target="_blank"
              rel="noopener noreferrer"
              class="url-link"
              :title="user.url"
            >
              <span class="url-text">{{ user.url }}</span>
              <ExternalLink class="w-3.5 h-3.5 flex-shrink-0" />
            </a>
            <span v-else-if="user.url" class="url-text text-[var(--text-2)]">
              {{ user.url }}
            </span>
            <span v-else class="muted">—</span>
          </div>

          <div class="utm-subgrid">
            <UserField label="Source" v-model="form.last_utm_source" :editing="editing" :original="decodeUTM(user.last_utm_source)" mono small />
            <UserField label="Medium" v-model="form.last_utm_medium" :editing="editing" :original="decodeUTM(user.last_utm_medium)" mono small />
            <UserField label="Campaign" v-model="form.last_utm_campaign" :editing="editing" :original="decodeUTM(user.last_utm_campaign)" mono small />
            <UserField label="Content" v-model="form.last_utm_content" :editing="editing" :original="decodeUTM(user.last_utm_content)" mono small />
            <UserField label="Term" v-model="form.last_utm_term" :editing="editing" :original="decodeUTM(user.last_utm_term)" mono small />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ExternalLink } from 'lucide-vue-next'
import UserField from '~/components/users/UserField.vue'
import type { TrackedUser, TrackedUserUpdateInput } from '#shared/types'

interface Props {
  user: TrackedUser | null
  form: Required<TrackedUserUpdateInput>
  editing: boolean
}

defineProps<Props>()

const safeExternalUrl = (raw: string | null | undefined): string | undefined => {
  if (!raw) return undefined
  try {
    const u = new URL(raw.trim())
    if (u.protocol === 'http:' || u.protocol === 'https:') {
      return u.toString()
    }
  } catch {
    return undefined
  }
  return undefined
}

const decodeUTM = (val: string | null | undefined): string => {
  if (!val) return ''
  try {
    return decodeURIComponent(val.replace(/\+/g, ' '))
  } catch {
    return val.replace(/\+/g, ' ')
  }
}

const formatDate = (iso: string): string => {
  if (!iso) return ''
  return new Date(iso).toLocaleString('pt-BR', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}
</script>

<style scoped>
.user-attribution-tab {
  display: flex;
  flex-direction: column;
  width: 100%;
  min-width: 0;
}

.attribution-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 1rem;
  width: 100%;
  min-width: 0;
}

.attribution-card {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  padding: 1rem;
  border-radius: 10px;
  background: var(--surface-soft-2);
  border: 1px solid var(--border-soft);
  min-width: 0;
  overflow: hidden;
}

.attribution-card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  min-width: 0;
}

.attr-badge {
  font-size: var(--fs-xs);
  font-weight: var(--fw-bold);
  text-transform: uppercase;
  letter-spacing: 0.04em;
  padding: 0.125rem 0.5rem;
  border-radius: 4px;
  white-space: nowrap;
}

.attr-badge.first { background: color-mix(in srgb, var(--info) 15%, transparent); color: var(--info); border: 1px solid color-mix(in srgb, var(--info) 30%, transparent); }
.attr-badge.last { background: var(--brand-soft); color: var(--brand); border: 1px solid var(--brand-border); }

.attr-date {
  font-size: var(--fs-xs);
  color: var(--text-3);
  white-space: nowrap;
}

.attr-body {
  display: flex;
  flex-direction: column;
  gap: 0.875rem;
  min-width: 0;
  overflow: hidden;
}

.attr-url-block {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  background: var(--surface-deep);
  padding: 0.625rem;
  border-radius: 8px;
  border: 1px solid var(--border-soft);
  min-width: 0;
  overflow: hidden;
}

.attr-url-label { font-size: var(--fs-2xs); font-weight: var(--fw-semibold); text-transform: uppercase; letter-spacing: 0.04em; color: var(--text-3); }
.url-link {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  font-family: var(--font-mono);
  font-size: var(--fs-sm);
  color: var(--brand);
  text-decoration: none;
  min-width: 0;
  width: 100%;
  overflow: hidden;
}
.url-text {
  min-width: 0;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.url-link:hover .url-text { text-decoration: underline; }
.url-edit { font-size: var(--fs-sm); font-family: var(--font-mono); background: var(--surface-deep); border: 1px solid var(--border); border-radius: 6px; padding: 0.375rem 0.625rem; color: var(--text-1); width: 100%; min-width: 0; outline: none; transition: border-color 0.15s; }
.url-edit:focus { border-color: var(--brand); }
.utm-subgrid { display: grid; grid-template-columns: minmax(0, 1fr); gap: 0.5rem; min-width: 0; width: 100%; }
.muted { color: var(--text-3); font-style: italic; font-size: var(--fs-sm); }

@media (max-width: 768px) {
  .attribution-grid {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
