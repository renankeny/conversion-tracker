<template>
  <article class="meta-card">
    <header class="meta-card-header">
      <div class="meta-title">
        <div class="card-avatar">
          <img
            v-if="avatarSrc && !imgBroken"
            :src="avatarSrc"
            :alt="account.name"
            referrerpolicy="no-referrer"
            class="card-avatar-img"
            @error="imgBroken = true"
          />
          <BrandIcon v-else channel="meta" size="md" :colored="true" />
        </div>
        <div class="meta-name">
          <h3>{{ account.ad_account_name || account.name }}</h3>
          <span v-if="account.ad_account_id" class="meta-sub-id">
            {{ account.ad_account_id }}
          </span>
        </div>
      </div>
      <span class="status-pill" :class="statusClass">{{ statusLabel }}</span>
    </header>

    <div class="meta-details">
      <div v-if="account.oauth_name || account.oauth_email" class="detail-row">
        <span class="detail-label">Perfil autorizado</span>
        <span class="detail-value">{{ account.oauth_name || account.oauth_email }}</span>
      </div>
      <div v-if="account.ad_account_id" class="detail-row">
        <span class="detail-label">Conta de Anúncios</span>
        <span class="detail-value mono">{{ account.ad_account_id }}</span>
      </div>
      <div v-if="account.default_page_id" class="detail-row">
        <span class="detail-label">Página CTWA</span>
        <span class="detail-value">{{ account.default_page_name || account.default_page_id }}</span>
      </div>
    </div>

    <footer class="meta-actions">
      <BaseButton variant="ghost" size="sm" @click="$emit('edit', account)">
        Configurar
      </BaseButton>
      <BaseButton variant="ghost" size="sm" @click="$emit('delete', account)">
        Excluir
      </BaseButton>
    </footer>
  </article>
</template>

<script setup lang="ts">
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
watch(avatarSrc, () => { imgBroken.value = false })

const statusClass = computed(() => {
  if (!props.account.enabled) return 'disabled'
  return props.account.connected ? 'connected' : 'pending'
})

const statusLabel = computed(() => {
  if (!props.account.enabled) return 'Desativada'
  if (!props.account.connected) return 'Aguardando OAuth'
  return 'Conectada'
})
</script>

<style scoped>
.meta-card { display: flex; flex-direction: column; gap: 1rem; padding: 1.25rem; background: var(--surface-2); border: 1px solid var(--border); border-radius: 12px; transition: border-color 0.15s; }
.meta-card:hover { border-color: var(--brand-border); }
.meta-card-header { display: flex; justify-content: space-between; align-items: flex-start; gap: 0.75rem; }
.meta-title { display: flex; gap: 0.75rem; align-items: center; flex: 1; min-width: 0; }
.card-avatar { width: 40px; height: 40px; border-radius: 10px; background: var(--surface-soft); border: 1px solid var(--border); display: flex; align-items: center; justify-content: center; overflow: hidden; flex-shrink: 0; }
.card-avatar-img { width: 100%; height: 100%; object-fit: cover; display: block; }
.meta-name { min-width: 0; }
.meta-name h3 { font-family: var(--font-heading); font-size: var(--fs-lg); font-weight: var(--fw-semibold); color: var(--text-strong); margin: 0; line-height: 1.25; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.meta-sub-id { display: block; margin-top: 0.15rem; font-family: var(--font-mono); font-size: var(--fs-xs); color: var(--text-3); }
.status-pill { font-size: var(--fs-xs); font-weight: var(--fw-semibold); padding: 0.25rem 0.625rem; border-radius: 999px; white-space: nowrap; }
.status-pill.connected { background: var(--success-soft); color: var(--success); }
.status-pill.pending { background: var(--warning-soft); color: var(--warning); }
.status-pill.disabled { background: var(--surface-soft); color: var(--text-3); }
.meta-details { display: flex; flex-direction: column; gap: 0.5rem; }
.detail-row { display: flex; justify-content: space-between; align-items: center; gap: 0.5rem; font-size: var(--fs-base); }
.detail-label { color: var(--text-2); }
.detail-value { color: var(--text-1); font-weight: var(--fw-medium); text-align: right; word-break: break-all; }
.detail-value.mono { font-family: var(--font-mono); font-size: var(--fs-sm); }
.meta-actions { display: flex; gap: 0.5rem; justify-content: flex-end; border-top: 1px solid var(--border-faint); padding-top: 0.75rem; margin-top: auto; }
</style>
