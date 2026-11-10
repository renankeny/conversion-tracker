<template>
  <label
    class="base-checkbox"
    :class="[
      `size-${size}`,
      { 'is-disabled': disabled, 'has-hint': !!hint }
    ]"
    @click.prevent="toggle"
  >
    <span
      class="checkbox-box"
      :class="{
        'is-checked': isChecked,
        'is-indeterminate': indeterminate && !isChecked
      }"
      role="checkbox"
      :aria-checked="indeterminate && !isChecked ? 'mixed' : isChecked"
      :aria-disabled="disabled"
      :tabindex="disabled ? -1 : 0"
      @keydown.space.prevent="toggle"
      @keydown.enter.prevent="toggle"
    >
      <Minus v-if="indeterminate && !isChecked" :size="iconSize" class="checkbox-icon" />
      <Check v-else-if="isChecked" :size="iconSize" class="checkbox-icon" />
    </span>

    <span v-if="label || $slots.default || hint" class="checkbox-content">
      <span v-if="label || $slots.default" class="checkbox-label">
        <slot>{{ label }}</slot>
      </span>
      <span v-if="hint" class="checkbox-hint">{{ hint }}</span>
    </span>
  </label>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Check, Minus } from 'lucide-vue-next'

interface Props {
  modelValue?: boolean | any[]
  checked?: boolean
  value?: any
  indeterminate?: boolean
  label?: string
  hint?: string
  disabled?: boolean
  size?: 'sm' | 'md'
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: undefined,
  checked: undefined,
  value: undefined,
  indeterminate: false,
  label: '',
  hint: '',
  disabled: false,
  size: 'md',
})

const emit = defineEmits<{
  'update:modelValue': [value: any]
  change: [checked: boolean]
}>()

const isChecked = computed<boolean>(() => {
  if (Array.isArray(props.modelValue)) {
    return props.modelValue.includes(props.value)
  }
  if (props.modelValue !== undefined) {
    return Boolean(props.modelValue)
  }
  return Boolean(props.checked)
})

const iconSize = computed(() => (props.size === 'sm' ? 11 : 13))

const toggle = () => {
  if (props.disabled) return
  if (Array.isArray(props.modelValue)) {
    const exists = props.modelValue.includes(props.value)
    const next = exists
      ? props.modelValue.filter((item) => item !== props.value)
      : [...props.modelValue, props.value]
    emit('update:modelValue', next)
    emit('change', !exists)
    return
  }
  const next = !isChecked.value
  emit('update:modelValue', next)
  emit('change', next)
}
</script>

<style scoped>
.base-checkbox {
  display: inline-flex;
  align-items: center;
  gap: 0.625rem;
  cursor: pointer;
  user-select: none;
  color: var(--text-1);
}

.base-checkbox.has-hint {
  align-items: flex-start;
}

.base-checkbox.is-disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.checkbox-box {
  width: 18px;
  height: 18px;
  border-radius: 5px;
  background: var(--surface-2);
  border: 1.5px solid var(--border);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  transition: all 0.15s ease;
  outline: none;
}

.size-sm .checkbox-box {
  width: 16px;
  height: 16px;
  border-radius: 4px;
}

.base-checkbox.has-hint .checkbox-box {
  margin-top: 2px;
}

.base-checkbox:hover:not(.is-disabled) .checkbox-box:not(.is-checked):not(.is-indeterminate) {
  border-color: var(--brand-border);
  background: var(--surface-soft);
}

.checkbox-box:focus-visible {
  box-shadow: 0 0 0 2px var(--brand-soft);
  border-color: var(--brand);
}

.checkbox-box.is-checked,
.checkbox-box.is-indeterminate {
  background: var(--brand);
  border-color: var(--brand);
  box-shadow: 0 0 10px var(--brand-soft);
}

.checkbox-icon {
  color: var(--brand-text);
  stroke-width: 3;
}

.checkbox-content {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
  min-width: 0;
}

.checkbox-label {
  font-size: var(--fs-md);
  line-height: 1.35;
  color: var(--text-1);
}

.size-sm .checkbox-label {
  font-size: var(--fs-base);
}

.checkbox-hint {
  font-size: var(--fs-sm);
  color: var(--text-3);
  line-height: 1.35;
}
</style>

