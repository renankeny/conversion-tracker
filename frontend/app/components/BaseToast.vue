<template>
  <div :class="toastClasses">
    <div class="toast-icon">
      <CheckCircle v-if="type === 'success'" :size="20" />
      <XCircle     v-else-if="type === 'error'"   :size="20" />
      <AlertTriangle v-else-if="type === 'warning'" :size="20" />
      <Info v-else :size="20" />
    </div>

    <div class="toast-content">
      <h4 class="toast-title">{{ title }}</h4>
      <p class="toast-message">{{ message }}</p>
    </div>

    <button class="toast-close" @click="$emit('close')">
      <X :size="16" />
    </button>
  </div>
</template>

<script setup lang="ts">
import { CheckCircle, XCircle, AlertTriangle, Info, X } from 'lucide-vue-next'
interface Props {
  type: 'success' | 'error' | 'warning' | 'info'
  title: string
  message: string
}

const props = defineProps<Props>()

defineEmits<{
  close: []
}>()

const toastClasses = computed(() => {
  const base = 'toast'
  const types = {
    success: 'toast-success',
    error: 'toast-error',
    warning: 'toast-warning',
    info: 'toast-info',
  }
  return [base, types[props.type]]
})
</script>

<style scoped>
.toast {
  display: flex;
  align-items: start;
  gap: 0.625rem;
  padding: 0.75rem 1rem;
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: 8px;
  box-shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.4), 0 8px 10px -6px rgba(0, 0, 0, 0.3);
  min-width: 280px;
  max-width: 380px;
  animation: slideIn 0.25s cubic-bezier(0.16, 1, 0.3, 1);
}

.toast-success {
  border-left: 3px solid var(--success);
}

.toast-error {
  border-left: 3px solid var(--danger);
}

.toast-warning {
  border-left: 3px solid var(--warning);
}

.toast-info {
  border-left: 3px solid var(--info);
}

.toast-icon {
  flex-shrink: 0;
  margin-top: 1px;
}

.toast-success .toast-icon {
  color: var(--success);
}

.toast-error .toast-icon {
  color: var(--danger);
}

.toast-warning .toast-icon {
  color: var(--warning);
}

.toast-info .toast-icon {
  color: var(--info);
}

.toast-content {
  flex: 1;
}

.toast-title {
  font-size: var(--fs-base);
  font-weight: var(--fw-semibold);
  color: var(--text-1);
  letter-spacing: -0.01em;
}

.toast-message {
  font-size: var(--fs-sm);
  color: var(--text-2);
  margin-top: 0.125rem;
  line-height: 1.4;
}

.toast-close {
  background: transparent;
  border: none;
  color: var(--text-3);
  cursor: pointer;
  padding: 0.15rem;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.15s ease;
  flex-shrink: 0;
}

.toast-close:hover {
  background: var(--surface-soft);
  color: var(--text-1);
}

@keyframes slideIn {
  from {
    transform: translateX(100%);
    opacity: 0;
  }
  to {
    transform: translateX(0);
    opacity: 1;
  }
}
</style>
