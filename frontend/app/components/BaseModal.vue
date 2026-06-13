<template>
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
              <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                <path
                  fill-rule="evenodd"
                  d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
                  clip-rule="evenodd"
                />
              </svg>
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
</template>

<script setup lang="ts">
interface Props {
  modelValue: boolean
  title?: string
  description?: string
  size?: 'sm' | 'md' | 'lg' | 'xl'
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

// Bloqueia scroll do body quando o modal está aberto
watch(
  () => props.modelValue,
  (open) => {
    if (!import.meta.client) return
    document.body.style.overflow = open ? 'hidden' : ''
  }
)

// ESC fecha o modal
onMounted(() => {
  if (!import.meta.client) return
  const onKey = (e: KeyboardEvent) => {
    if (e.key === 'Escape' && props.modelValue) close()
  }
  window.addEventListener('keydown', onKey)
  onBeforeUnmount(() => window.removeEventListener('keydown', onKey))
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
  border-radius: 16px;
  box-shadow: var(--shadow-elevated);
  display: flex;
  flex-direction: column;
  max-height: 90vh;
}

.modal-size-sm {
  max-width: 420px;
}

.modal-size-md {
  max-width: 560px;
}

.modal-size-lg {
  max-width: 760px;
}

.modal-size-xl {
  max-width: 960px;
}

.modal-header {
  display: flex;
  align-items: flex-start;
  gap: 1rem;
  padding: 1.25rem 1.5rem;
  border-bottom: 1px solid var(--border-soft);
}

.modal-header-content {
  flex: 1;
}

.modal-title {
  font-family: 'Instrument Sans', sans-serif;
  font-size: 1.25rem;
  font-weight: 600;
  color: var(--text-strong);
}

.modal-description {
  font-size: 0.875rem;
  color: var(--text-2);
  margin-top: 0.25rem;
}

.modal-close {
  background: transparent;
  border: none;
  color: var(--text-3);
  cursor: pointer;
  padding: 0.25rem;
  border-radius: 6px;
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
  padding: 1.5rem;
  overflow-y: auto;
  flex: 1;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  padding: 1rem 1.5rem;
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
