<template>
  <BaseModal
    :model-value="modelValue"
    :title="title"
    :description="description"
    size="sm"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <p class="confirm-message">{{ message }}</p>

    <div v-if="requireTyping" class="confirm-typing">
      <p class="typing-hint">
        Digite <strong>{{ typingPhrase }}</strong> para confirmar:
      </p>
      <BaseInput v-model="typed" :placeholder="typingPhrase" />
    </div>

    <template #footer>
      <BaseButton variant="ghost" @click="onCancel">{{ cancelLabel }}</BaseButton>
      <BaseButton
        :variant="variant"
        :disabled="requireTyping && typed !== typingPhrase"
        :loading="loading"
        @click="onConfirm"
      >
        {{ confirmLabel }}
      </BaseButton>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
interface Props {
  modelValue: boolean
  title: string
  description?: string
  message: string
  confirmLabel?: string
  cancelLabel?: string
  variant?: 'primary' | 'danger' | 'success'
  loading?: boolean
  requireTyping?: boolean
  typingPhrase?: string
}

const props = withDefaults(defineProps<Props>(), {
  confirmLabel: 'Confirmar',
  cancelLabel: 'Cancelar',
  variant: 'primary',
  loading: false,
  requireTyping: false,
  typingPhrase: 'CONFIRMAR',
})

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  confirm: []
  cancel: []
}>()

const typed = ref('')

watch(
  () => props.modelValue,
  (open) => {
    if (open) typed.value = ''
  }
)

const onConfirm = () => {
  emit('confirm')
}

const onCancel = () => {
  emit('cancel')
  emit('update:modelValue', false)
}
</script>

<style scoped>
.confirm-message {
  font-size: var(--fs-md);
  color: var(--text-1);
  line-height: 1.5;
}

.confirm-typing {
  margin-top: 0.875rem;
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
}

.typing-hint {
  font-size: var(--fs-base);
  color: var(--text-2);
}

.typing-hint strong {
  color: var(--danger);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  background: var(--danger-soft);
  padding: 0.125rem 0.5rem;
  border-radius: 4px;
}
</style>
