<template>
  <article class="ga-card">
    <header class="ga-card-header">
      <div class="ga-title">
        <div class="ga-avatar">
          <img
            v-if="(account.picture_url || account.oauth_picture_url) && !imgBroken"
            :src="account.picture_url || account.oauth_picture_url"
            :alt="account.name"
            referrerpolicy="no-referrer"
            class="ga-avatar-img"
            @error="imgBroken = true"
          />
          <BrandIcon v-else channel="google_ads" size="md" :colored="true" />
        </div>
        <div class="ga-name">
          <h3>{{ account.name }}</h3>
          <span v-if="account.customer_id" class="ga-customer-id">
            ID: {{ formatCustomerID(account.customer_id) }}
          </span>
        </div>
      </div>
      <span class="status-pill" :class="statusClass">{{ statusLabel }}</span>
    </header>

    <div class="ga-meta">
      <div v-if="account.oauth_email" class="meta-row">
        <span class="meta-label">Conta autorizada</span>
        <span class="meta-value">{{ account.oauth_email }}</span>
      </div>
      <div v-if="account.login_customer_id" class="meta-row">
        <span class="meta-label">MCC (Login Customer ID)</span>
        <span class="meta-value">{{ formatCustomerID(account.login_customer_id) }}</span>
      </div>
      <div class="meta-row">
        <span class="meta-label">Conversion Actions</span>
        <span class="meta-value">{{ activeActionsCount }} ativas</span>
      </div>
    </div>

    <footer class="ga-actions">
      <BaseButton variant="ghost" size="sm" @click="$emit('edit', account)">
        Editar
      </BaseButton>
      <BaseButton variant="ghost" size="sm" @click="$emit('delete', account)">
        Excluir
      </BaseButton>
    </footer>
  </article>
</template>

<script setup lang="ts">
import BrandIcon from '~/components/events/BrandIcon.vue'
import type { GoogleAdsAccount } from '#shared/types'

interface Props {
  account: GoogleAdsAccount
}
const props = defineProps<Props>()
const imgBroken = ref(false)

defineEmits<{
  edit: [account: GoogleAdsAccount]
  delete: [account: GoogleAdsAccount]
}>()

const statusClass = computed(() => {
  if (!props.account.enabled) return 'disabled'
  return props.account.connected ? 'connected' : 'pending'
})

const statusLabel = computed(() => {
  if (!props.account.enabled) return 'Desativada'
  if (!props.account.connected) return 'Aguardando OAuth'
  return 'Conectada'
})

const activeActionsCount = computed(() => {
  return (props.account.conversion_actions || []).filter(
    (ca) => ca.enabled && ca.conversion_action_id,
  ).length
})

const formatCustomerID = (id: string) => {
  // Customer ID do Google Ads vem como "1234567890" — formata como
  // 123-456-7890 pra leitura.
  const digits = id.replace(/\D/g, '')
  if (digits.length !== 10) return id
  return `${digits.slice(0, 3)}-${digits.slice(3, 6)}-${digits.slice(6)}`
}
</script>

<style scoped>
.ga-card {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  padding: 1.25rem;
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: 12px;
  transition: border-color 0.15s, box-shadow 0.15s;
}

.ga-card:hover {
  border-color: var(--brand-border);
}

.ga-card-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 0.75rem;
}

.ga-title {
  display: flex;
  gap: 0.75rem;
  align-items: center;
  flex: 1;
  min-width: 0;
}
.ga-avatar {
  width: 40px;
  height: 40px;
  border-radius: 10px;
  background: var(--surface-soft);
  border: 1px solid var(--border);
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  flex-shrink: 0;
}
.ga-avatar-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.ga-name h3 {
  font-family: var(--font-heading);
  font-size: var(--fs-lg);
  font-weight: var(--fw-semibold);
  color: var(--text-strong);
  margin: 0;
  line-height: 1.2;
}

.ga-customer-id {
  display: block;
  margin-top: 0.25rem;
  font-size: var(--fs-sm);
  color: var(--text-3);
  font-family: monospace;
}

.status-pill {
  font-size: var(--fs-xs);
  font-weight: var(--fw-semibold);
  padding: 0.25rem 0.625rem;
  border-radius: 999px;
  white-space: nowrap;
}

.status-pill.connected {
  background: var(--success-soft);
  color: var(--success);
}

.status-pill.pending {
  background: var(--warning-soft);
  color: var(--warning);
}

.status-pill.disabled {
  background: var(--surface-soft);
  color: var(--text-3);
}

.ga-meta {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.meta-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 0.5rem;
  font-size: var(--fs-base);
}

.meta-label {
  color: var(--text-2);
}

.meta-value {
  color: var(--text-1);
  font-weight: var(--fw-medium);
  text-align: right;
  word-break: break-all;
}

.ga-actions {
  display: flex;
  gap: 0.5rem;
  justify-content: flex-end;
  border-top: 1px solid var(--border-faint);
  padding-top: 0.75rem;
  margin-top: 0.25rem;
}
</style>
