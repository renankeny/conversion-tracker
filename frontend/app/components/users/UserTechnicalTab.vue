<template>
  <div v-if="user" class="user-technical-tab">
    <!-- Grid de Cookies de Plataforma -->
    <div class="cookies-grid">
      <!-- Meta Cookies -->
      <div class="cookie-card">
        <div class="cookie-card-header">
          <div class="flex items-center gap-1.5">
            <BrandIcon channel="meta" size="sm" />
            <span class="cookie-card-title">Cookies Meta (Facebook/IG)</span>
          </div>
        </div>
        <div class="cookie-card-fields">
          <UserField label="FBP" v-model="form.fbp" :editing="editing" :original="user.fbp" mono small />
          <UserField label="FBC" v-model="form.fbc" :editing="editing" :original="user.fbc" mono small />
          <UserField v-if="user.ctwa_clid || editing" label="CTWA Clid" v-model="form.ctwa_clid" :editing="editing" :original="user.ctwa_clid" mono small />
        </div>
      </div>

      <!-- Google Ads Cookies -->
      <div class="cookie-card">
        <div class="cookie-card-header">
          <div class="flex items-center gap-1.5">
            <BrandIcon channel="google_ads" size="sm" />
            <span class="cookie-card-title">Cookies Google Ads</span>
          </div>
        </div>
        <div class="cookie-card-fields">
          <UserField label="GCLID" v-model="form.gclid" :editing="editing" :original="user.gclid" mono small />
          <UserField label="GBraid" v-model="form.gbraid" :editing="editing" :original="user.gbraid" mono small />
          <UserField label="WBraid" v-model="form.wbraid" :editing="editing" :original="user.wbraid" mono small />
        </div>
      </div>

      <!-- TikTok Cookies -->
      <div class="cookie-card">
        <div class="cookie-card-header">
          <div class="flex items-center gap-1.5">
            <BrandIcon channel="tiktok" size="sm" />
            <span class="cookie-card-title">Cookies TikTok Ads</span>
          </div>
        </div>
        <div class="cookie-card-fields">
          <UserField label="TTP" v-model="form.ttp" :editing="editing" :original="user.ttp" mono small />
          <UserField label="TTClid" v-model="form.ttclid" :editing="editing" :original="user.ttclid" mono small />
        </div>
      </div>

      <!-- GA4 Cookies -->
      <div class="cookie-card">
        <div class="cookie-card-header">
          <div class="flex items-center gap-1.5">
            <BrandIcon channel="ga4" size="sm" />
            <span class="cookie-card-title">Google Analytics 4</span>
          </div>
        </div>
        <div class="cookie-card-fields">
          <UserField label="GA4 Client ID" v-model="form.ga4_client_id" :editing="editing" :original="user.ga4_client_id" mono small />
          <UserField label="GA4 Session ID" v-model="form.ga4_session_id" :editing="editing" :original="user.ga4_session_id" mono small />
        </div>
      </div>
    </div>

    <!-- User Agent -->
    <section v-if="editing || user.client_user_agent" class="section">
      <div class="section-header-row">
        <h4 class="section-title">User-Agent do Navegador</h4>
        <button
          v-if="user.client_user_agent && !editing"
          type="button"
          class="id-copy-action"
          :class="{ copied: copiedField === 'ua' }"
          @click="copyToClipboard(user.client_user_agent, 'ua')"
        >
          <Check v-if="copiedField === 'ua'" class="w-3 h-3 copy-check-icon" />
          <Copy v-else class="w-3 h-3" />
          <span>{{ copiedField === 'ua' ? 'Copiado' : 'Copiar UA' }}</span>
        </button>
      </div>
      <textarea
        v-if="editing"
        v-model="form.client_user_agent"
        class="ua-edit"
        rows="2"
      />
      <code v-else class="ua">{{ user.client_user_agent }}</code>
    </section>

    <!-- Timestamps -->
    <section class="section">
      <h4 class="section-title">Registros & Auditoria</h4>
      <div class="grid-timestamps">
        <UserField label="Criado em" :value="formatDate(user.created_at)" readonly />
        <UserField label="Atualizado em" :value="formatDate(user.updated_at)" readonly />
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { Copy, Check } from 'lucide-vue-next'
import BrandIcon from '~/components/events/BrandIcon.vue'
import UserField from '~/components/users/UserField.vue'
import type { TrackedUser, TrackedUserUpdateInput } from '#shared/types'

interface Props {
  user: TrackedUser | null
  form: Required<TrackedUserUpdateInput>
  editing: boolean
}

defineProps<Props>()

const copiedField = ref<string | null>(null)

const copyToClipboard = async (text: string | null | undefined, field: string) => {
  if (!text) return
  try {
    await navigator.clipboard.writeText(text)
    copiedField.value = field
    setTimeout(() => {
      if (copiedField.value === field) copiedField.value = null
    }, 2000)
  } catch {
    /* silent */
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
.user-technical-tab {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
  width: 100%;
  min-width: 0;
}

.cookies-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.875rem;
  width: 100%;
  min-width: 0;
}

.cookie-card {
  display: flex;
  flex-direction: column;
  gap: 0.625rem;
  padding: 0.875rem;
  border-radius: 10px;
  background: var(--surface-soft-2);
  border: 1px solid var(--border-soft);
  min-width: 0;
  overflow: hidden;
}

.cookie-card-header { display: flex; align-items: center; justify-content: space-between; }
.cookie-card-title { font-size: var(--fs-base); font-weight: var(--fw-semibold); color: var(--text-1); }
.cookie-card-fields { display: flex; flex-direction: column; gap: 0.5rem; }
.copy-check-icon { color: var(--success); }

.section { display: flex; flex-direction: column; gap: 0.75rem; }
.section-header-row { display: flex; align-items: center; justify-content: space-between; }
.section-title { font-size: var(--fs-sm); font-weight: var(--fw-semibold); color: var(--text-3); text-transform: uppercase; letter-spacing: 0.05em; margin: 0; }

.grid-timestamps {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 0.75rem 1rem;
  background: var(--surface-soft-2);
  border: 1px solid var(--border-soft);
  border-radius: 10px;
  padding: 1rem;
}

.ua, .ua-edit {
  display: block;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: var(--fs-sm);
  background: var(--surface-deep);
  border: 1px solid var(--border-soft);
  padding: 0.625rem;
  border-radius: 8px;
  word-break: break-all;
  color: var(--text-2);
  width: 100%;
  resize: vertical;
}

.ua-edit {
  color: var(--text-1);
}

.id-copy-action {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  padding: 0.1rem 0.375rem;
  font-size: var(--fs-2xs);
  font-weight: var(--fw-semibold);
  border-radius: 4px;
  background: var(--surface-deep);
  border: 1px solid var(--border-soft);
  color: var(--text-3);
  cursor: pointer;
  transition: all 0.15s ease;
}

.id-copy-action:hover {
  border-color: var(--brand);
  color: var(--brand);
}

.id-copy-action.copied {
  color: var(--success);
  border-color: var(--success);
}

@media (max-width: 768px) {
  .cookies-grid {
    grid-template-columns: 1fr;
  }
}
</style>
