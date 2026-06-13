<template>
  <div class="purchases-card">
    <div class="purchases-header">
      <div>
        <h3 class="purchases-title">Últimas compras</h3>
        <p class="purchases-subtitle">{{ subtitle }}</p>
      </div>
      <BaseBadge variant="success">{{ purchases.length }}</BaseBadge>
    </div>

    <div v-if="!purchases.length" class="purchases-empty">
      Nenhuma compra registrada no período.
    </div>

    <div v-else class="purchases-list">
      <div v-for="ev in purchases" :key="ev.id" class="purchase-row">
        <div class="purchase-icon">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
            <path d="M3 1a1 1 0 000 2h1.22l.305 1.222a.997.997 0 00.01.042l1.358 5.43-.893.892C3.74 11.846 4.632 14 6.414 14H15a1 1 0 100-2H6.414l1-1H14a1 1 0 00.894-.553l3-6A1 1 0 0017 3H6.28l-.31-1.243A1 1 0 005 1H3z" />
            <circle cx="7" cy="17" r="2" />
            <circle cx="15" cy="17" r="2" />
          </svg>
        </div>
        <div class="purchase-content">
          <div class="purchase-line-1">
            <span class="purchase-pixel">{{ ev.pixel_name }}</span>
            <BaseBadge :variant="badgeVariant(ev.status)">{{ statusLabel(ev.status) }}</BaseBadge>
          </div>
          <div class="purchase-line-2">
            <code class="purchase-id" :title="ev.external_id">
              {{ truncate(ev.external_id, 14) }}
            </code>
            <span class="purchase-time">{{ formatDate(ev.created_at) }}</span>
          </div>
        </div>
      </div>
    </div>

    <div v-if="purchases.length" class="purchases-footer">
      <NuxtLink to="/events" class="see-all">Ver todos os eventos →</NuxtLink>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { SentEvent, EventStatus } from '#shared/types'

interface Props {
  purchases: SentEvent[]
  subtitle?: string
}

withDefaults(defineProps<Props>(), {
  subtitle: '',
})

const badgeVariant = (status: EventStatus): 'success' | 'warning' | 'danger' => {
  if (status === 'sent') return 'success'
  if (status === 'pending') return 'warning'
  return 'danger'
}

const statusLabel = (status: EventStatus): string => {
  if (status === 'sent') return 'Enviada'
  if (status === 'pending') return 'Pendente'
  return 'Falhou'
}

const truncate = (s: string, n: number): string => {
  if (!s) return ''
  return s.length > n ? s.slice(0, n) + '…' : s
}

const formatDate = (iso: string): string => {
  if (!iso) return ''
  const d = new Date(iso)
  return d.toLocaleString('pt-BR', {
    day: '2-digit',
    month: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}
</script>

<style scoped>
.purchases-card {
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 1.25rem;
  box-shadow: var(--shadow-card);
  display: flex;
  flex-direction: column;
}

.purchases-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 1rem;
}

.purchases-title {
  font-family: 'Instrument Sans', sans-serif;
  font-size: 1.125rem;
  font-weight: 600;
  color: var(--text-strong);
}

.purchases-subtitle {
  font-size: 0.8125rem;
  color: var(--text-3);
  margin-top: 0.125rem;
}

.purchases-empty {
  padding: 2rem 1rem;
  text-align: center;
  color: var(--text-3);
  font-size: 0.875rem;
  font-style: italic;
}

.purchases-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.purchase-row {
  display: flex;
  gap: 0.75rem;
  padding: 0.625rem 0.75rem;
  border-radius: 8px;
  background: var(--surface-soft-2);
  border: 1px solid var(--border-soft);
  transition: all 0.15s;
}

.purchase-row:hover {
  background: var(--surface-soft);
  border-color: var(--border);
}

.purchase-icon {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  background: var(--success-soft);
  color: var(--success);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.purchase-content {
  flex: 1;
  min-width: 0;
}

.purchase-line-1 {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  margin-bottom: 0.25rem;
}

.purchase-pixel {
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--text-1);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.purchase-line-2 {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  font-size: 0.75rem;
}

.purchase-id {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  background: var(--surface-soft);
  padding: 0.0625rem 0.375rem;
  border-radius: 4px;
  color: var(--text-2);
}

.purchase-time {
  color: var(--text-3);
}

.purchases-footer {
  margin-top: 1rem;
  padding-top: 0.75rem;
  border-top: 1px solid var(--border-soft);
  text-align: center;
}

.see-all {
  font-size: 0.8125rem;
  color: var(--brand);
  text-decoration: none;
  font-weight: 500;
}

.see-all:hover {
  text-decoration: underline;
}
</style>
