<template>
  <div class="date-divider">
    <span class="date-pill">{{ label }}</span>
  </div>
</template>

<script setup lang="ts">
import { SP_TIMEZONE, dateKeySP } from '~/utils/formatDate'

const props = defineProps<{
  date: string | Date
}>()

const ymdToEpochDays = (ymd: string): number => {
  const [y = 1970, m = 1, d = 1] = ymd.split('-').map(Number)
  return Math.round(Date.UTC(y, m - 1, d) / 86_400_000)
}

const label = computed(() => {
  if (!props.date) return ''
  const d = new Date(props.date)
  if (isNaN(d.getTime())) return ''

  const dYmd = dateKeySP(d)
  const nowYmd = dateKeySP(new Date())
  const diffDays = ymdToEpochDays(nowYmd) - ymdToEpochDays(dYmd)

  if (diffDays === 0) return 'Hoje'
  if (diffDays === 1) return 'Ontem'

  return new Intl.DateTimeFormat('pt-BR', {
    day: 'numeric',
    month: 'long',
    year: dYmd.slice(0, 4) !== nowYmd.slice(0, 4) ? 'numeric' : undefined,
    timeZone: SP_TIMEZONE,
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

