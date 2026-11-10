<template>
  <div class="field">
    <span class="field-label">{{ label }}</span>
    <input
      v-if="editing && !readonly"
      class="field-input"
      :class="{ mono, small }"
      type="text"
      :value="modelValue"
      @input="$emit('update:modelValue', ($event.target as HTMLInputElement).value)"
    />
    <span
      v-else
      class="field-value"
      :class="{ mono, small, readonly }"
    >
      {{ displayValue }}
    </span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  label: string
  modelValue?: string
  original?: string
  value?: string
  editing?: boolean
  mono?: boolean
  small?: boolean
  readonly?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: '',
  original: '',
  value: '',
  editing: false,
  mono: false,
  small: false,
  readonly: false,
})

defineEmits<{
  'update:modelValue': [val: string]
}>()

const displayValue = computed(() => {
  return props.value || props.original || '—'
})
</script>

<style scoped>
.field {
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
  min-width: 0;
  width: 100%;
}

.field-label {
  font-size: var(--fs-xs);
  color: var(--text-3);
  font-weight: var(--fw-medium);
}

.field-value {
  font-size: var(--fs-base);
  font-weight: var(--fw-medium);
  color: var(--text-1);
  word-break: break-all;
  overflow-wrap: anywhere;
  min-width: 0;
}

.field-value.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
}

.field-value.small {
  font-size: var(--fs-sm);
}

.field-value.readonly {
  color: var(--text-2);
}

.field-input {
  font-size: var(--fs-base);
  background: var(--surface-deep);
  border: 1px solid var(--border);
  border-radius: 6px;
  padding: 0.375rem 0.625rem;
  color: var(--text-1);
  outline: none;
  width: 100%;
  transition: border-color 0.15s;
}

.field-input:focus {
  border-color: var(--brand);
  box-shadow: 0 0 0 2px var(--brand-soft);
}

.field-input.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
}

.field-input.small {
  font-size: var(--fs-sm);
}
</style>
