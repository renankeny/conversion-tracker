<template>
  <article class="meta-card">
    <div class="meta-top">
      <header class="meta-header">
        <div class="meta-avatar-wrap">
          <img
            v-if="avatarSrc && !imgBroken"
            :src="avatarSrc"
            :alt="account.name"
            referrerpolicy="no-referrer"
            class="meta-avatar-img"
            @error="imgBroken = true"
          >
          <div v-else class="meta-avatar">
            <BrandIcon channel="meta" size="md" :colored="true" />
          </div>
          <span class="avatar-dot" :class="dotClass" />
        </div>

        <div class="meta-titles">
          <h3 class="meta-name">{{ account.name }}</h3>
          <p class="meta-sub">{{ headerSubtitle }}</p>
        </div>

        <span class="status-pill" :class="pillClass">
          <span class="status-dot" :class="dotClass" />
          {{ statusLabel }}
        </span>
      </header>

      <div class="meta-body">
        <div class="meta-row">
          <span class="meta-label">Perfil OAuth</span>
          <span class="meta-val" :class="{ muted: !oauthProfile }">
            {{ oauthProfile || 'Não autenticado' }}
          </span>
        </div>

        <div class="meta-row">
          <span class="meta-label">Conta de Anúncios</span>
          <span class="meta-val meta-mono" :class="{ muted: !account.ad_account_id }">
            {{ account.ad_account_id || 'Não selecionada' }}
          </span>
        </div>

        <div class="meta-row">
          <span class="meta-label">Página CTWA</span>
          <span class="meta-val" :class="{ muted: !pageDisplay }">
            {{ pageDisplay || 'Nenhuma página' }}
          </span>
        </div>
      </div>
    </div>

    <footer class="meta-actions">
      <button
        type="button"
        class="action-main"
        @click="$emit('edit', account)"
      >
        <Settings :size="16" />
        <span>Configurar conta</span>
      </button>

      <div class="action-divider" />

      <button
        type="button"
        class="action-icon action-delete"
        title="Excluir conta Meta Ads"
        @click="$emit('delete', account)"
      >
        <Trash2 :size="16" />
      </button>
    </footer>
  </article>
</template>

<script setup lang="ts">
import { Settings, Trash2 } from 'lucide-vue-next'
import BrandIcon from '~/components/events/BrandIcon.vue'
import type { MetaAccount } from '#shared/types'

const props = defineProps<{
  account: MetaAccount
}>()

defineEmits<{
  edit: [account: MetaAccount]
  delete: [account: MetaAccount]
}>()

const imgBroken = ref(false)
const avatarSrc = computed(
  () => props.account.ad_account_picture_url || props.account.picture_url || props.account.oauth_picture_url || '',
)
watch(avatarSrc, () => {
  imgBroken.value = false
})

const statusState = computed<'connected' | 'pending' | 'disabled'>(() => {
  if (!props.account.enabled) return 'disabled'
  return props.account.connected ? 'connected' : 'pending'
})

const statusLabel = computed(() => {
  if (!props.account.enabled) return 'Desativada'
  if (!props.account.connected) return 'Aguardando OAuth'
  return 'Conectada'
})

const pillClass = computed(() => `pill-${statusState.value}`)
const dotClass = computed(() => `dot-${statusState.value}`)

const oauthProfile = computed(() => props.account.oauth_name || props.account.oauth_email || '')
const pageDisplay = computed(() => props.account.default_page_name || props.account.default_page_id || '')

const headerSubtitle = computed(() => {
  if (props.account.ad_account_name && props.account.ad_account_name !== props.account.name) {
    return props.account.ad_account_name
  }
  if (props.account.ad_account_id) return props.account.ad_account_id
  if (oauthProfile.value) return oauthProfile.value
  return 'Conta Meta Ads'
})
</script>

<style scoped>
.meta-card { display: flex; flex-direction: column; justify-content: space-between; background: var(--surface-2); border: 1px solid var(--border); border-radius: 14px; overflow: hidden; transition: border-color 0.2s ease, transform 0.2s ease; }
.meta-card:hover { border-color: var(--brand-border); transform: translateY(-1px); }
.meta-header { display: flex; align-items: center; gap: 0.875rem; padding: 1.125rem 1.25rem; border-bottom: 1px solid var(--border-soft); }
.meta-avatar-wrap { position: relative; width: 46px; height: 46px; flex-shrink: 0; }
.meta-avatar, .meta-avatar-img { width: 46px; height: 46px; border-radius: 12px; object-fit: cover; display: flex; align-items: center; justify-content: center; }
.meta-avatar-img { border: 1px solid var(--border); background: var(--surface-deep); }
.meta-avatar { background: var(--surface-soft); border: 1px solid var(--border); }
.avatar-dot { position: absolute; right: -2px; bottom: -2px; width: 11px; height: 11px; border-radius: 50%; border: 2px solid var(--surface-2); }
.meta-titles { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 0.15rem; }
.meta-name { font-family: var(--font-heading); font-size: var(--fs-md); font-weight: var(--fw-semibold); color: var(--text-strong); margin: 0; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.meta-sub { font-size: var(--fs-xs); color: var(--text-3); margin: 0; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.status-pill { display: inline-flex; align-items: center; gap: 0.4rem; padding: 0.3rem 0.7rem; border-radius: 999px; font-size: var(--fs-2xs); font-weight: var(--fw-semibold); text-transform: uppercase; letter-spacing: 0.04em; white-space: nowrap; flex-shrink: 0; }
.pill-connected { background: var(--surface-soft); color: var(--success); border: 1px solid var(--border-soft); }
.pill-pending { background: var(--surface-soft); color: var(--warning); border: 1px solid var(--border-soft); }
.pill-disabled { background: var(--surface-deep); color: var(--text-3); border: 1px solid var(--border-soft); }
.status-dot { width: 7px; height: 7px; border-radius: 50%; flex-shrink: 0; }
.dot-connected { background: var(--success); }
.dot-pending { background: var(--warning); }
.dot-disabled { background: var(--text-3); }
.meta-body { padding: 0.95rem 1.25rem; display: flex; flex-direction: column; gap: 0.65rem; background: var(--surface-deep); border-bottom: 1px solid var(--border-soft); }
.meta-row { display: flex; align-items: center; justify-content: space-between; gap: 0.75rem; font-size: var(--fs-xs); }
.meta-label { color: var(--text-3); font-weight: var(--fw-medium); white-space: nowrap; }
.meta-val { color: var(--text-1); font-weight: var(--fw-semibold); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; text-align: right; }
.meta-mono { font-family: var(--font-mono); }
.muted { color: var(--text-3); font-weight: var(--fw-normal); }
.meta-actions { display: flex; align-items: stretch; height: 46px; background: var(--surface-2); }
.action-main { flex: 1; display: inline-flex; align-items: center; justify-content: center; gap: 0.5rem; padding: 0 1rem; border: none; background: transparent; color: var(--text-1); font-size: var(--fs-sm); font-weight: var(--fw-semibold); cursor: pointer; white-space: nowrap; transition: background 0.15s ease, color 0.15s ease; }
.action-main:hover { background: var(--surface-soft); color: var(--brand); }
.action-divider { width: 1px; background: var(--border-soft); flex-shrink: 0; }
.action-icon { width: 46px; display: inline-flex; align-items: center; justify-content: center; border: none; background: transparent; color: var(--text-2); cursor: pointer; flex-shrink: 0; transition: background 0.15s ease, color 0.15s ease; }
.action-delete { color: var(--danger); }
.action-delete:hover { background: var(--surface-soft); color: var(--danger); }
</style>
