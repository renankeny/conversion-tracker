<template>
  <div class="date-divider">
    <span class="date-pill">{{ label }}</span>
  </div>
</template>

<script setup lang="ts">
const props = defineProps<{
  date: string | Date
}>()

const label = computed(() => {
  if (!props.date) return ''
  const d = new Date(props.date)
  if (isNaN(d.getTime())) return ''

  const now = new Date()
  const dDay = new Date(d.getFullYear(), d.getMonth(), d.getDate())
  const nowDay = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  const diffDays = Math.round((nowDay.getTime() - dDay.getTime()) / (1000 * 60 * 60 * 24))

  if (diffDays === 0) return 'Hoje'
  if (diffDays === 1) return 'Ontem'

  return new Intl.DateTimeFormat('pt-BR', {
    day: 'numeric',
    month: 'long',
    year: d.getFullYear() !== now.getFullYear() ? 'numeric' : undefined,
    timeZone: 'America/Sao_Paulo',
  }).format(d)
})
</script>

<style scoped>
.date-divider {
  display: flex;
  justify-content: center;
  align-items: center;
  margin: 0.625rem 0;
  position: sticky;
  top: 0.25rem;
  z-index: 2;
  pointer-events: none;
}
.date-pill {
  display: inline-flex;
  align-items: center;
  padding: 0.25rem 0.75rem;
  background: var(--surface-2);
  border: 1px solid var(--border-soft);
  border-radius: 9999px;
  font-size: var(--fs-xs);
  font-weight: var(--fw-medium);
  color: var(--text-2);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.2);
  user-select: none;
}
</style>

