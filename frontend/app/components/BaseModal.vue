<template>
  <ClientOnly>
    <Teleport to="body">
      <Transition name="modal">
        <div v-if="modelValue" class="modal-overlay" @click="onOverlayClick">
          <div class="modal" :class="modalClasses" @click.stop>
            <div v-if="title || $slots.header" class="modal-header">
              <div class="modal-header-content">
                <slot name="header">
                  <h2 class="modal-title">{{ title }}</h2>
                  <p v-if="description" class="modal-description">{{ description }}</p>
                </slot>
              </div>
              <button class="modal-close" @click="close" aria-label="Fechar">
                <X :size="20" />
              </button>
            </div>

            <div class="modal-body">
              <slot />
            </div>

            <div v-if="$slots.footer" class="modal-footer">
              <slot name="footer" />
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>
  </ClientOnly>
</template>

<script setup lang="ts">
import { X } from 'lucide-vue-next'
interface Props {
  modelValue: boolean
  title?: string
  description?: string
  size?: 'sm' | 'md' | 'lg' | 'xl' | '2xl'
  closeOnOverlay?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  size: 'md',
  closeOnOverlay: true,
})

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  close: []
}>()

const modalClasses = computed(() => `modal-size-${props.size}`)

const close = () => {
  emit('update:modelValue', false)
  emit('close')
}

const onOverlayClick = () => {
  if (props.closeOnOverlay) close()
}

// Bloqueia scroll do body quando o modal está aberto (suporta modais aninhados e unmount)
let isLockingBody = false
const updateBodyLock = (open: boolean) => {
  if (!import.meta.client) return
  const currentCount = Number(document.body.dataset.modalCount || '0')
  if (open && !isLockingBody) {
    isLockingBody = true
    const next = currentCount + 1
    document.body.dataset.modalCount = String(next)
    document.body.style.overflow = 'hidden'
  } else if (!open && isLockingBody) {
    isLockingBody = false
    const next = Math.max(0, currentCount - 1)
    document.body.dataset.modalCount = String(next)
    if (next === 0) {
      document.body.style.overflow = ''
    }
  }
}

watch(
  () => props.modelValue,
  (open) => {
    updateBodyLock(open)
  },
  { immediate: true }
)

// ESC fecha o modal
onMounted(() => {
  if (!import.meta.client) return
  const onKey = (e: KeyboardEvent) => {
    if (e.key === 'Escape' && props.modelValue) close()
  }
  window.addEventListener('keydown', onKey)
  onBeforeUnmount(() => {
    window.removeEventListener('keydown', onKey)
    updateBodyLock(false)
  })
})
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.65);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 1rem;
  overflow-y: auto;
}

.modal {
  width: 100%;
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: 12px;
  box-shadow: var(--shadow-elevated);
  display: flex;
  flex-direction: column;
  max-height: 90vh;
  max-height: 90dvh;
  min-width: 0;
  overflow: hidden;
}

.modal-size-sm {
  max-width: 400px;
}

.modal-size-md {
  max-width: 520px;
}

.modal-size-lg {
  max-width: 720px;
}

.modal-size-xl {
  max-width: 900px;
}

.modal-size-2xl {
  max-width: 1040px;
}

.modal-header {
  display: flex;
  align-items: flex-start;
  gap: 0.75rem;
  padding: 1rem 1.25rem;
  border-bottom: 1px solid var(--border-soft);
}

.modal-header-content {
  flex: 1;
}

.modal-title {
  font-family: 'Instrument Sans', sans-serif;
  font-size: var(--fs-xl);
  font-weight: var(--fw-semibold);
  color: var(--text-strong);
  letter-spacing: -0.01em;
}

.modal-description {
  font-size: var(--fs-base);
  color: var(--text-2);
  margin-top: 0.15rem;
}

.modal-close {
  background: transparent;
  border: none;
  color: var(--text-3);
  cursor: pointer;
  padding: 0.25rem;
  border-radius: 5px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.15s ease;
  flex-shrink: 0;
}

.modal-close:hover {
  background: var(--surface-soft);
  color: var(--text-1);
}

.modal-body {
  padding: 1.25rem;
  overflow-y: auto;
  overflow-x: hidden;
  flex: 1;
  min-width: 0;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
  padding: 0.875rem 1.25rem;
  border-top: 1px solid var(--border-soft);
  flex-wrap: wrap;
}

.modal-enter-active,
.modal-leave-active {
  transition: all 0.2s ease;
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}

.modal-enter-from .modal,
.modal-leave-to .modal {
  transform: scale(0.96) translateY(-8px);
}
</style>
