<template>
  <button
    type="button"
    class="status-switch"
    :class="{ 'is-active': active, 'is-disabled': disabled || loading }"
    :disabled="disabled || loading"
    :title="active ? 'Ativo — clique para pausar' : 'Pausado — clique para ativar'"
    role="switch"
    :aria-checked="active"
    @click.stop="emit('toggle')"
  >
    <span class="switch-thumb">
      <Loader2 v-if="loading" :size="10" class="spin-icon" />
    </span>
  </button>
</template>

<script setup lang="ts">
import { Loader2 } from 'lucide-vue-next'

defineProps<{
  active: boolean
  disabled?: boolean
  loading?: boolean
}>()

const emit = defineEmits<{
  toggle: []
}>()
</script>

<style scoped>
.status-switch {
  position: relative;
  width: 34px;
  height: 19px;
  border-radius: 999px;
  border: 1px solid var(--border);
  background: var(--surface-soft);
  padding: 2px;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  transition: background 0.16s ease, border-color 0.16s ease;
  flex-shrink: 0;
}
.status-switch.is-active {
  background: var(--brand);
  border-color: var(--brand);
}
.status-switch.is-disabled {
  opacity: 0.45;
  cursor: not-allowed;
}
.switch-thumb {
  width: 13px;
  height: 13px;
  border-radius: 50%;
  background: var(--text-1);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  transform: translateX(0);
  transition: transform 0.16s ease, background 0.16s ease;
}
.status-switch.is-active .switch-thumb {
  transform: translateX(15px);
  background: var(--brand-text);
}
.spin-icon {
  animation: spin 0.8s linear infinite;
}
@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
