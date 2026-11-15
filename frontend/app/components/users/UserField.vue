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
    <div v-else class="field-value-row">
      <span
        class="field-value"
        :class="{ mono, small, readonly, muted: !rawText }"
      >
        {{ displayValue }}
      </span>
      <button
        v-if="copyable && textToCopy"
        type="button"
        class="field-copy-btn"
        :class="{ copied }"
        :title="copied ? 'Copiado!' : `Copiar ${label}`"
        @click.stop="handleCopy"
      >
        <Check v-if="copied" :size="12" />
        <Copy v-else :size="12" />
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onBeforeUnmount } from 'vue'
import { Copy, Check } from 'lucide-vue-next'

interface Props {
  label: string
  modelValue?: string
  original?: string
  value?: string
  copyValue?: string
  editing?: boolean
  mono?: boolean
  small?: boolean
  readonly?: boolean
  copyable?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: '',
  original: '',
  value: '',
  copyValue: '',
  editing: false,
  mono: false,
  small: false,
  readonly: false,
  copyable: false,
})

defineEmits<{
  'update:modelValue': [val: string]
}>()

const copied = ref(false)
let copyTimer: ReturnType<typeof setTimeout> | null = null

const rawText = computed(() => (props.value || props.original || '').trim())
const displayValue = computed(() => rawText.value || '-')
const textToCopy = computed(() => (props.copyValue || rawText.value).trim())

const handleCopy = async () => {
  if (!textToCopy.value) return
  try {
    await navigator.clipboard.writeText(textToCopy.value)
    copied.value = true
    if (copyTimer) clearTimeout(copyTimer)
    copyTimer = setTimeout(() => {
      copied.value = false
    }, 2000)
  } catch {
    /* silent */
  }
}

onBeforeUnmount(() => {
  if (copyTimer) clearTimeout(copyTimer)
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

.field-value-row {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  min-width: 0;
}

.field-value {
  font-size: var(--fs-base);
  font-weight: var(--fw-medium);
  color: var(--text-1);
  word-break: break-all;
  overflow-wrap: anywhere;
  min-width: 0;
}

.field-value.mono { font-family: var(--font-mono); }
.field-value.small { font-size: var(--fs-sm); }
.field-value.readonly { color: var(--text-2); }
.field-value.muted { color: var(--text-3); }

.field-copy-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  padding: 0.1875rem;
  border-radius: 4px;
  background: transparent;
  border: 1px solid transparent;
  color: var(--text-3);
  cursor: pointer;
  transition: all 0.15s ease;
}

.field-copy-btn:hover {
  color: var(--brand);
  background: var(--surface-soft);
  border-color: var(--border-soft);
}

.field-copy-btn.copied {
  color: var(--success);
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

.field-input.mono { font-family: var(--font-mono); }
.field-input.small { font-size: var(--fs-sm); }
</style>
