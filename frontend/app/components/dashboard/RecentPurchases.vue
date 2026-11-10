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
      <RecentPurchaseRow
        v-for="ev in pagedPurchases"
        :key="ev.event_id"
        :event="ev"
        @select="$emit('select', $event)"
      />
    </div>

    <div v-if="purchases.length" class="purchases-footer">
      <div class="purchases-pagination">
        <button class="page-btn" :disabled="page === 1" @click="page--">
          <ChevronLeft :size="14" />
        </button>
        <span class="page-info">Página {{ page }} de {{ totalPages }}</span>
        <button class="page-btn" :disabled="page === totalPages" @click="page++">
          <ChevronRight :size="14" />
        </button>
      </div>
      <NuxtLink to="/events" class="see-all">Ver todos os eventos →</NuxtLink>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ChevronRight, ChevronLeft } from 'lucide-vue-next'
import RecentPurchaseRow from '~/components/dashboard/RecentPurchaseRow.vue'
import type { GroupedEvent } from '#shared/types'

const PAGE_SIZE = 5

interface Props {
  purchases: GroupedEvent[]
  subtitle?: string
}

const props = withDefaults(defineProps<Props>(), {
  subtitle: '',
})

defineEmits<{ select: [event: GroupedEvent] }>()

const page = ref(1)

const totalPages = computed(() => Math.max(1, Math.ceil(props.purchases.length / PAGE_SIZE)))

const pagedPurchases = computed(() => {
  const start = (page.value - 1) * PAGE_SIZE
  return props.purchases.slice(start, start + PAGE_SIZE)
})

watch(() => props.purchases, () => { page.value = 1 })
</script>

<style scoped>
.purchases-card {
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 1.125rem;
  box-shadow: var(--shadow-card);
  display: flex;
  flex-direction: column;
  height: 100%;
  box-sizing: border-box;
}

.purchases-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 0.875rem;
}

.purchases-title {
  font-family: 'Instrument Sans', sans-serif;
  font-size: var(--fs-lg);
  font-weight: var(--fw-semibold);
  color: var(--text-strong);
  letter-spacing: -0.01em;
}

.purchases-subtitle {
  font-size: var(--fs-sm);
  color: var(--text-3);
  margin-top: 0.125rem;
}

.purchases-empty {
  padding: 2.5rem 1rem;
  text-align: center;
  color: var(--text-3);
  font-size: var(--fs-base);
}

.purchases-list {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
}

.purchases-footer {
  margin-top: 1rem;
  padding-top: 0.75rem;
  border-top: 1px solid var(--border-soft);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.purchases-pagination {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.page-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border-radius: 6px;
  border: 1px solid var(--border-soft);
  background: var(--surface-soft-2);
  color: var(--text-2);
  cursor: pointer;
  transition: all 0.15s;
}

.page-btn:hover:not(:disabled) {
  background: var(--surface-soft);
  color: var(--text-1);
}

.page-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.page-info {
  font-size: var(--fs-sm);
  color: var(--text-3);
  white-space: nowrap;
}

.see-all {
  font-size: var(--fs-base);
  color: var(--brand);
  text-decoration: none;
  white-space: nowrap;
}

.see-all:hover {
  text-decoration: underline;
}
</style>
