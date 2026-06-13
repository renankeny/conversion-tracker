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
  const base = 'inline-flex items-center justify-center gap-2 font-medium rounded-lg transition-all duration-200 disabled:opacity-50 disabled:cursor-not-allowed'
  
  const sizes = {
    sm: 'px-3 py-1.5 text-sm',
    md: 'px-4 py-2 text-base',
    lg: 'px-5 py-2.5 text-lg',
  }
  
  const variants = {
    primary: 'bg-primary text-white hover:bg-primary/90 hover:shadow-lg hover:-translate-y-0.5 active:translate-y-0',
    success: 'bg-secondary text-white hover:bg-secondary/90 hover:shadow-lg hover:-translate-y-0.5 active:translate-y-0',
    danger: 'bg-danger text-white hover:bg-danger/90 hover:shadow-lg hover:-translate-y-0.5 active:translate-y-0',
    ghost: 'bg-transparent text-[var(--text-1)] hover:bg-[var(--surface-soft)]',
    outline: 'bg-transparent border border-[var(--border)] text-[var(--text-1)] hover:bg-[var(--surface-soft)] hover:border-[var(--brand)]',
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
