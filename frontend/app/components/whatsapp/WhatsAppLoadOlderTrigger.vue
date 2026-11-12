<template>
  <div ref="containerRef" class="load-older-container">
    <div v-if="hasMore" class="trigger-wrap">
      <button
        type="button"
        class="load-more-btn"
        :disabled="loading"
        @click="$emit('loadMore')"
      >
        <span v-if="loading" class="spinner-mini" />
        <ArrowUp v-else :size="13" class="btn-arrow" />
        <span>{{ loading ? 'Carregando anteriores...' : 'Carregar mensagens anteriores' }}</span>
      </button>
    </div>
    <div v-else-if="totalLoaded > 0" class="start-banner">
      <Lock :size="12" class="lock-icon" />
      <span>Início da conversa do WhatsApp</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ArrowUp, Lock } from 'lucide-vue-next'

const props = withDefaults(
  defineProps<{
    hasMore: boolean
    loading: boolean
    totalLoaded: number
    canAutoTrigger?: boolean
  }>(),
  {
    canAutoTrigger: true,
  },
)

const emit = defineEmits<{
  (e: 'loadMore'): void
}>()

const containerRef = ref<HTMLElement | null>(null)
let observer: IntersectionObserver | null = null

onMounted(() => {
  if (typeof IntersectionObserver !== 'undefined' && containerRef.value) {
    observer = new IntersectionObserver(
      (entries) => {
        const first = entries[0]
        if (first?.isIntersecting && props.hasMore && !props.loading && props.canAutoTrigger) {
          emit('loadMore')
        }
      },
      { threshold: 0.1 },
    )
    observer.observe(containerRef.value)
  }
})

onUnmounted(() => {
  observer?.disconnect()
  observer = null
})
</script>

<style scoped>
.load-older-container {
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 0.5rem 0;
  width: 100%;
}
.trigger-wrap {
  display: flex;
  justify-content: center;
}
.load-more-btn {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  padding: 0.35rem 0.75rem;
  border-radius: 9999px;
  background: var(--surface-2);
  border: 1px solid var(--border);
  color: var(--text-2);
  font-size: var(--fs-xs);
  font-weight: var(--fw-medium);
  cursor: pointer;
  transition: background 0.15s ease, color 0.15s ease;
}
.load-more-btn:hover:not(:disabled) {
  background: var(--surface-soft);
  color: var(--text-strong);
}
.load-more-btn:disabled {
  opacity: 0.7;
  cursor: default;
}
.btn-arrow {
  color: var(--brand);
}
.start-banner {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.25rem 0.65rem;
  background: var(--surface-soft-2);
  border: 1px dashed var(--border-soft);
  border-radius: 6px;
  color: var(--text-3);
  font-size: var(--fs-2xs);
}
.lock-icon {
  color: var(--brand);
  opacity: 0.8;
}
.spinner-mini {
  width: 0.75rem;
  height: 0.75rem;
  border: 2px solid var(--border);
  border-top-color: var(--brand);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }
</style>

