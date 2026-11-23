<template>
  <section class="history-card">
    <div class="history-head">
      <div>
        <h2 class="card-title">Últimas Notificações</h2>
        <p class="card-desc">Mantém automaticamente apenas as 10 mais recentes.</p>
      </div>
      <BaseButton
        v-if="unreadCount > 0"
        variant="outline"
        size="sm"
        @click="$emit('read-all')"
      >
        <CheckCheck :size="15" />
        <span>Marcar lidas</span>
      </BaseButton>
    </div>

    <div v-if="loading && !notifications.length" class="empty-box">
      Carregando histórico...
    </div>

    <div v-else-if="!notifications.length" class="empty-box">
      <Bell :size="24" class="empty-icon" />
      <p class="empty-title">Nenhuma notificação recente</p>
      <p class="empty-sub">Clique em "Testar" ao lado de qualquer gatilho para receber um exemplo.</p>
    </div>

    <div v-else class="history-list">
      <button
        v-for="item in notifications"
        :key="item.id"
        type="button"
        class="history-item"
        :class="{ unread: !item.is_read }"
        @click="$emit('open', item)"
      >
        <span class="unread-dot" :class="{ visible: !item.is_read }" />
        <div class="item-main">
          <div class="item-top">
            <span class="item-title">{{ item.title }}</span>
            <span class="item-time">{{ formatTime(item.created_at) }}</span>
          </div>
          <p class="item-body">{{ item.body }}</p>
          <div v-if="item.account_name" class="item-meta">
            <BaseBadge variant="default">{{ item.account_name }}</BaseBadge>
          </div>
        </div>
        <ChevronRight :size="16" class="item-arrow" />
      </button>
    </div>
  </section>
</template>

<script setup lang="ts">
import { Bell, CheckCheck, ChevronRight } from 'lucide-vue-next'
import type { UserNotificationItem } from '#shared/types'

defineProps<{
  notifications: UserNotificationItem[]
  unreadCount: number
  loading: boolean
}>()

defineEmits<{
  (e: 'open', item: UserNotificationItem): void
  (e: 'read-all'): void
}>()

const formatTime = (iso: string) => {
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
.history-card {
  background: var(--surface-2); border: 1px solid var(--border); border-radius: 14px;
  padding: 1.25rem; display: flex; flex-direction: column; gap: 1rem;
}
.history-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 0.75rem; flex-wrap: wrap; }
.card-title { font-size: var(--fs-lg); font-weight: var(--fw-semibold); color: var(--text-strong); margin: 0 0 0.15rem; }
.card-desc { font-size: var(--fs-sm); color: var(--text-3); margin: 0; }
.empty-box {
  padding: 2.25rem 1rem; border-radius: 10px; border: 1px dashed var(--border);
  background: var(--surface-1); display: flex; flex-direction: column; align-items: center;
  justify-content: center; gap: 0.35rem; text-align: center; color: var(--text-2); font-size: var(--fs-base);
}
.empty-icon { color: var(--text-3); margin-bottom: 0.25rem; }
.empty-title { font-size: var(--fs-md); font-weight: var(--fw-semibold); color: var(--text-1); margin: 0; }
.empty-sub { font-size: var(--fs-sm); color: var(--text-3); margin: 0; max-width: 320px; }
.history-list { display: flex; flex-direction: column; gap: 0.45rem; }
.history-item {
  width: 100%; display: flex; align-items: center; gap: 0.75rem; padding: 0.75rem 0.875rem;
  border-radius: 10px; background: var(--surface-1); border: 1px solid var(--border-soft);
  text-align: left; cursor: pointer; transition: all 0.15s ease;
}
.history-item:hover { border-color: var(--brand-border); background: var(--surface-soft); }
.history-item.unread { border-color: var(--brand-border); }
.unread-dot { width: 8px; height: 8px; border-radius: 50%; background: transparent; flex-shrink: 0; }
.unread-dot.visible { background: var(--brand); }
.item-main { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 0.2rem; }
.item-top { display: flex; align-items: baseline; justify-content: space-between; gap: 0.5rem; }
.item-title { font-size: var(--fs-base); font-weight: var(--fw-semibold); color: var(--text-strong); }
.item-time { font-size: var(--fs-xs); color: var(--text-3); flex-shrink: 0; }
.item-body { font-size: var(--fs-sm); color: var(--text-2); margin: 0; line-height: 1.45; }
.item-meta { margin-top: 0.2rem; }
.item-arrow { color: var(--text-3); flex-shrink: 0; }
</style>
