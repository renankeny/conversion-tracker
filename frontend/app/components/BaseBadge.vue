<template>
  <span class="base-badge" :class="badgeClasses">
    <slot />
  </span>
</template>

<script setup lang="ts">
interface Props {
  variant?: 'success' | 'warning' | 'danger' | 'info' | 'default'
}

const props = withDefaults(defineProps<Props>(), {
  variant: 'default',
})

const badgeClasses = computed(() => {
  const base = 'inline-flex items-center gap-1 px-2 py-0.5 rounded-full uppercase tracking-wider border leading-none'

  const variants = {
    success: 'bg-[var(--success-soft)] text-[var(--success)] border-[var(--brand-border)]',
    warning: 'bg-[var(--warning-soft)] text-[var(--warning)] border-[var(--warning-border)]',
    danger: 'bg-[var(--danger-soft)] text-[var(--danger)] border-[var(--danger-border)]',
    info: 'bg-[var(--info-soft)] text-[var(--info)] border-[var(--brand-border)]',
    default: 'bg-[var(--surface-soft)] text-[var(--text-2)] border-[var(--border)]',
  }

  return [base, variants[props.variant]]
})
</script>

<style scoped>
.base-badge { font-size: var(--fs-xs); font-weight: var(--fw-semibold); }
</style>
