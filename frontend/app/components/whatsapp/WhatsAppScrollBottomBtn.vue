<template>
  <button
    type="button"
    class="scroll-bottom-btn"
    title="Rolar para o final"
    @click="$emit('click')"
  >
    <ChevronDown :size="18" class="btn-icon" />
    <span v-if="(unreadCount ?? 0) > 0" class="unread-badge">
      {{ (unreadCount ?? 0) > 99 ? '99+' : unreadCount }}
    </span>
  </button>
</template>

<script setup lang="ts">
import { ChevronDown } from 'lucide-vue-next'

withDefaults(
  defineProps<{
    unreadCount?: number
  }>(),
  {
    unreadCount: 0,
  },
)

defineEmits<{
  (e: 'click'): void
}>()
</script>

<style scoped>
.scroll-bottom-btn {
  position: absolute;
  right: 1rem;
  bottom: 1rem;
  z-index: 10;
  width: 2.25rem;
  height: 2.25rem;
  border-radius: 50%;
  background: var(--surface-2);
  border: 1px solid var(--border);
  color: var(--text-2);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.35);
  transition: transform 0.15s ease, background 0.15s ease, color 0.15s ease;
}
.scroll-bottom-btn:hover {
  background: var(--surface-1);
  color: var(--text-strong);
  transform: translateY(-2px);
}
.btn-icon {
  display: block;
}
.unread-badge {
  position: absolute;
  top: -0.35rem;
  right: -0.35rem;
  min-width: 1.125rem;
  height: 1.125rem;
  padding: 0 0.25rem;
  border-radius: 9999px;
  background: var(--brand);
  color: var(--brand-text);
  font-size: var(--fs-2xs);
  font-weight: var(--fw-bold);
  display: flex;
  align-items: center;
  justify-content: center;
  line-height: 1;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.3);
}
</style>

