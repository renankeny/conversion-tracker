<template>
  <div class="account-card" :class="{ 'is-selected': isActiveInPanel, 'is-disabled': !account.enabled }">
    <div class="account-main">
      <div class="account-icon" :class="{ active: isActiveInPanel }">
        <Building2 :size="18" />
      </div>

      <div class="account-body">
        <div class="account-title-row">
          <h3 class="account-name">{{ account.name }}</h3>
          <div class="account-badges">
            <BaseBadge v-if="isActiveInPanel" variant="info">Selecionada no painel</BaseBadge>
            <BaseBadge :variant="account.enabled ? 'success' : 'danger'">
              {{ account.enabled ? 'Ativa' : 'Desativada' }}
            </BaseBadge>
            <BaseBadge v-if="account.id === 1" variant="neutral">Principal</BaseBadge>
          </div>
        </div>

        <div class="account-meta-row">
          <span class="meta-item">ID <strong>#{{ account.id }}</strong></span>
          <span class="meta-sep">·</span>
          <span class="meta-item">Slug <code>{{ account.slug }}</code></span>
          <span class="meta-sep">·</span>
          <span class="meta-item">
            <strong>{{ account.pixels_count ?? 0 }}</strong>
            {{ (account.pixels_count ?? 0) === 1 ? 'pixel' : 'pixels' }}
          </span>
          <span class="meta-sep">·</span>
          <span class="meta-item">
            <strong>{{ account.users_count ?? 0 }}</strong>
            {{ (account.users_count ?? 0) === 1 ? 'usuário' : 'usuários' }}
          </span>
          <span class="meta-sep">·</span>
          <span class="meta-item">{{ retentionLabel }}</span>
          <span class="meta-sep">·</span>
          <span class="meta-item">Criada em {{ formatDate(account.created_at) }}</span>
        </div>
      </div>
    </div>

    <div class="account-actions">
      <BaseButton
        v-if="!isActiveInPanel"
        variant="outline"
        size="sm"
        @click="$emit('select', account)"
      >
        Selecionar no painel
      </BaseButton>
      <BaseButton variant="secondary" size="sm" @click="$emit('edit', account)">
        Editar
      </BaseButton>
      <BaseButton
        v-if="account.id !== 1"
        variant="ghost"
        size="sm"
        @click="$emit('delete', account)"
      >
        <span class="text-danger">Remover</span>
      </BaseButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Building2 } from 'lucide-vue-next'
import type { Account } from '#shared/types'
import { formatDateSP } from '~/utils/formatDate'

const props = defineProps<{
  account: Account
  isActiveInPanel: boolean
}>()

defineEmits<{
  select: [account: Account]
  edit: [account: Account]
  delete: [account: Account]
}>()

const retentionLabel = computed(() => {
  const s = props.account.sent_events_retention_days
  const c = props.account.event_counters_retention_days
  if (s == null && c == null) return 'Retenção padrão'
  const parts: string[] = []
  if (s != null) parts.push(`Eventos: ${s === 0 ? 'sem limite' : `${s}d`}`)
  if (c != null) parts.push(`Contadores: ${c === 0 ? 'sem limite' : `${c}d`}`)
  return parts.join(' / ')
})

const formatDate = (iso: string): string => formatDateSP(iso) || '—'
</script>

<style scoped>
.account-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  flex-wrap: wrap;
  padding: 1.125rem 1.25rem;
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: 12px;
  transition: border-color 0.15s ease, background 0.15s ease;
}
.account-card:hover {
  border-color: var(--border-soft);
}
.account-card.is-selected {
  border-color: var(--brand-border);
  background: var(--surface-2);
}
.account-card.is-disabled {
  opacity: 0.78;
}
.account-main {
  display: flex;
  align-items: flex-start;
  gap: 0.875rem;
  flex: 1;
  min-width: 260px;
}
.account-icon {
  width: 38px;
  height: 38px;
  border-radius: 10px;
  background: var(--surface-soft);
  border: 1px solid var(--border-soft);
  color: var(--text-2);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  margin-top: 0.125rem;
}
.account-icon.active {
  background: var(--brand-soft);
  border-color: var(--brand-border);
  color: var(--brand);
}
.account-body {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
  min-width: 0;
}
.account-title-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.5rem;
}
.account-name {
  font-family: var(--font-heading);
  font-size: var(--fs-lg);
  font-weight: var(--fw-semibold);
  color: var(--text-strong);
}
.account-badges {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.375rem;
}
.account-meta-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.375rem;
  font-size: var(--fs-sm);
  color: var(--text-3);
}
.meta-item strong {
  font-weight: var(--fw-semibold);
  color: var(--text-2);
}
.meta-item code {
  font-family: var(--font-mono);
  font-size: var(--fs-xs);
  background: var(--surface-soft);
  color: var(--text-2);
  padding: 0.0625rem 0.375rem;
  border-radius: 4px;
}
.meta-sep {
  color: var(--border);
}
.account-actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
}
.text-danger {
  color: var(--danger);
}
</style>

