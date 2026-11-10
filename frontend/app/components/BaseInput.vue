<template>
  <div class="base-input">
    <label v-if="label" :for="inputId" class="input-label">
      {{ label }}
      <span v-if="required" class="text-danger ml-1">*</span>
    </label>

    <div class="input-wrapper" :class="{ 'has-icon': !!$slots.icon }">
      <div v-if="$slots.icon" class="input-icon">
        <slot name="icon" />
      </div>

      <input
        :id="inputId"
        :type="type"
        :value="modelValue"
        :placeholder="placeholder"
        :disabled="disabled"
        :required="required"
        :class="['input-base', { 'input-error-state': !!error }]"
        @input="onInput"
        @blur="$emit('blur', $event)"
        @focus="$emit('focus', $event)"
      />
    </div>

    <p v-if="error" class="input-error">{{ error }}</p>
    <p v-else-if="hint" class="input-hint">{{ hint }}</p>
  </div>
</template>

<script setup lang="ts">
interface Props {
  modelValue?: string | number
  type?: string
  label?: string
  placeholder?: string
  error?: string
  hint?: string
  disabled?: boolean
  required?: boolean
}

defineProps<Props>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
  blur: [event: FocusEvent]
  focus: [event: FocusEvent]
}>()

const inputId = useId()

const onInput = (event: Event) => {
  const target = event.target as HTMLInputElement
  emit('update:modelValue', target.value)
}
</script>

<style scoped>
.base-input {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
}

.input-label {
  font-size: var(--fs-base);
  font-weight: var(--fw-medium);
  color: var(--text-2);
  letter-spacing: -0.01em;
}

.input-wrapper {
  position: relative;
}

.input-icon {
  position: absolute;
  left: 0.75rem;
  top: 50%;
  transform: translateY(-50%);
  color: var(--text-3);
  pointer-events: none;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 1rem;
  height: 1rem;
  z-index: 1;
}

.input-base {
  width: 100%;
  height: 36px;
  padding: 0.45rem 0.75rem;
  border-radius: 6px;
  border: 1px solid var(--border);
  background: var(--surface-2);
  color: var(--text-1);
  font-size: var(--fs-md);
  transition: all 0.15s ease;
}

.input-wrapper.has-icon .input-base {
  padding-left: 2.25rem;
}

.input-base:focus {
  outline: none;
  border-color: var(--brand);
  box-shadow: 0 0 0 2px var(--brand-soft);
  background: var(--surface-1);
}

.input-base:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.input-base::placeholder {
  color: var(--text-3);
  font-size: var(--fs-base);
}

.input-error-state {
  border-color: var(--danger);
}

.input-error-state:focus {
  border-color: var(--danger);
  box-shadow: 0 0 0 2px var(--danger-soft);
}

.input-error {
  font-size: var(--fs-sm);
  color: var(--danger);
  margin-top: 0.125rem;
}

.input-hint {
  font-size: var(--fs-sm);
  color: var(--text-3);
  margin-top: 0.125rem;
}
</style>
