<template>
  <button
    :type="type"
    :disabled="disabled || loading"
    :class="buttonClasses"
    @click="$emit('click', $event)"
  >
    <span v-if="loading" class="spinner"></span>
    <slot v-else-if="$slots.iconLeft" name="iconLeft" />
    <span v-if="!loading"><slot /></span>
    <slot v-if="!loading && $slots.iconRight" name="iconRight" />
  </button>
</template>

<script setup lang="ts">
interface Props {
  variant?: 'primary' | 'success' | 'danger' | 'ghost' | 'outline'
  size?: 'sm' | 'md' | 'lg'
  type?: 'button' | 'submit' | 'reset'
  disabled?: boolean
  loading?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  variant: 'primary',
  size: 'md',
  type: 'button',
  disabled: false,
  loading: false,
})

defineEmits<{
  click: [event: MouseEvent]
}>()

const buttonClasses = computed(() => {
  const base = 'inline-flex items-center justify-center gap-1.5 font-medium rounded-md transition-all duration-150 select-none cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed disabled:pointer-events-none active:scale-[0.98]'
  
  const sizes = {
    sm: 'px-2.5 py-1 text-xs leading-tight min-h-[30px]',
    md: 'px-3.5 py-1.5 text-xs font-medium leading-normal min-h-[34px]',
    lg: 'px-4.5 py-2 text-sm leading-normal min-h-[40px]',
  }
  
  const variants = {
    primary: 'bg-primary text-[var(--brand-text)] font-semibold shadow-sm hover:brightness-105 hover:shadow-[0_0_12px_var(--brand-soft)] active:brightness-95',
    success: 'bg-secondary text-white shadow-sm hover:brightness-105 hover:shadow-[0_0_12px_var(--brand-soft)] active:brightness-95',
    danger: 'bg-danger text-white shadow-sm hover:bg-danger/90 hover:shadow-[0_0_12px_var(--danger-soft)] active:brightness-95',
    ghost: 'bg-transparent text-[var(--text-2)] hover:text-[var(--text-1)] hover:bg-[var(--surface-soft)] active:bg-[var(--surface-soft-2)]',
    outline: 'bg-transparent border border-[var(--border)] text-[var(--text-1)] hover:bg-[var(--surface-soft)] hover:border-[var(--brand-border)] hover:text-[var(--brand)]',
  }
  
  return [base, sizes[props.size], variants[props.variant]]
})
</script>

<style scoped>
.spinner {
  width: 1em;
  height: 1em;
  border: 2px solid currentColor;
  border-top-color: transparent;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
