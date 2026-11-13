<template>
  <div ref="dropdownRef" class="base-dropdown" :class="[`size-${size}`]">
    <label v-if="label" :for="dropdownId" class="dropdown-label">
      {{ label }}
    </label>

    <div
      :id="dropdownId"
      ref="triggerRef"
      class="dropdown-trigger"
      :class="{ 'is-open': isOpen, 'is-disabled': disabled }"
      @click="toggle"
    >
      <span class="dropdown-value" :class="{ 'is-placeholder': !selectedOption && placeholder }">
        {{ selectedOption ? selectedOption.label : (placeholder || 'Selecione...') }}
      </span>
      <svg class="dropdown-icon" :class="{ 'is-rotated': isOpen }" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
        <path fill-rule="evenodd" d="M5.23 7.21a.75.75 0 011.06.02L10 11.085l3.71-3.855a.75.75 0 111.08 1.04l-4.25 4.4a.75.75 0 01-1.08 0l-4.25-4.4a.75.75 0 01.02-1.06z" clip-rule="evenodd" />
      </svg>
    </div>

    <small v-if="hint" class="dropdown-hint">{{ hint }}</small>

    <Teleport to="body">
      <Transition name="fade-down">
        <ul v-if="isOpen" ref="menuRef" class="dropdown-menu" :style="menuStyle">
          <li v-if="options.length === 0" class="dropdown-empty">
            Nenhuma opção
          </li>
          <li
            v-for="option in options"
            :key="String(option.value)"
            class="dropdown-item"
            :class="{ 'is-selected': isOptionSelected(option.value) }"
            @click="select(option.value)"
          >
            <span class="item-label">{{ option.label }}</span>
            <svg v-if="isOptionSelected(option.value)" class="check-icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
              <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd" />
            </svg>
          </li>
        </ul>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, useId, watch, onBeforeUnmount } from 'vue'
import { onClickOutside } from '@vueuse/core'

export type DropdownValue = string | number | boolean | null | undefined

export interface DropdownOption {
  value: DropdownValue
  label: string
}

interface Props {
  modelValue?: DropdownValue
  label?: string
  hint?: string
  options: DropdownOption[]
  placeholder?: string
  disabled?: boolean
  size?: 'sm' | 'md'
}

const props = withDefaults(defineProps<Props>(), {
  size: 'md',
})

const emit = defineEmits<{
  'update:modelValue': [value: DropdownValue]
}>()

const dropdownId = useId()
const dropdownRef = ref<HTMLElement | null>(null)
const triggerRef = ref<HTMLElement | null>(null)
const menuRef = ref<HTMLElement | null>(null)
const isOpen = ref(false)
const menuStyle = ref<Record<string, string>>({})

const isOptionSelected = (optVal: DropdownValue): boolean => {
  if (props.modelValue === optVal) return true
  if (props.modelValue == null && (optVal === '' || optVal === null)) return props.modelValue === optVal
  return String(props.modelValue ?? '') === String(optVal ?? '')
}

const selectedOption = computed(() =>
  props.options.find((opt) => isOptionSelected(opt.value))
)

const updateMenuPosition = () => {
  if (!triggerRef.value) return
  const rect = triggerRef.value.getBoundingClientRect()
  menuStyle.value = {
    top: `${rect.bottom + 6}px`,
    left: `${rect.left}px`,
    minWidth: `${rect.width}px`,
  }
}

const toggle = () => {
  if (props.disabled) return
  if (!isOpen.value) updateMenuPosition()
  isOpen.value = !isOpen.value
}

const select = (value: DropdownValue) => {
  emit('update:modelValue', value)
  isOpen.value = false
}

watch(isOpen, (open) => {
  if (!import.meta.client) return
  if (open) {
    window.addEventListener('scroll', updateMenuPosition, true)
    window.addEventListener('resize', updateMenuPosition)
  } else {
    window.removeEventListener('scroll', updateMenuPosition, true)
    window.removeEventListener('resize', updateMenuPosition)
  }
})

onBeforeUnmount(() => {
  if (!import.meta.client) return
  window.removeEventListener('scroll', updateMenuPosition, true)
  window.removeEventListener('resize', updateMenuPosition)
})

onClickOutside(dropdownRef, () => {
  isOpen.value = false
}, { ignore: [menuRef] })
</script>

<style scoped>
.base-dropdown { position: relative; display: flex; flex-direction: column; gap: 0.375rem; }
.dropdown-label { font-size: var(--fs-base); font-weight: var(--fw-medium); color: var(--text-2); letter-spacing: -0.01em; }
.dropdown-hint { font-size: var(--fs-sm); color: var(--text-3); line-height: 1.35; }
.dropdown-trigger { width: 100%; height: 38px; display: flex; align-items: center; justify-content: space-between; gap: 0.5rem; padding: 0.45rem 0.75rem; border-radius: 6px; border: 1px solid var(--border); background: var(--surface-2); cursor: pointer; transition: all 0.15s ease; user-select: none; }
.size-sm .dropdown-trigger { height: 34px; padding: 0.35rem 0.65rem; }
.dropdown-trigger:hover:not(.is-disabled) { border-color: var(--brand-border); }
.dropdown-trigger.is-open { border-color: var(--brand); box-shadow: 0 0 0 2px var(--brand-soft); background: var(--surface-1); }
.dropdown-trigger.is-disabled { opacity: 0.5; cursor: not-allowed; }
.dropdown-value { color: var(--text-1); font-size: var(--fs-md); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.size-sm .dropdown-value { font-size: var(--fs-base); }
.dropdown-value.is-placeholder { color: var(--text-3); font-size: var(--fs-base); }
.dropdown-icon { width: 1rem; height: 1rem; color: var(--text-3); transition: transform 0.2s ease; flex-shrink: 0; }
.dropdown-icon.is-rotated { transform: rotate(180deg); }
.dropdown-menu { position: fixed; max-height: 240px; overflow-y: auto; background: var(--surface-2); backdrop-filter: blur(12px); border: 1px solid var(--border); border-radius: 8px; box-shadow: var(--shadow-elevated); z-index: 9999; list-style: none; padding: 0.35rem; margin: 0; }
.dropdown-empty { padding: 0.5rem 0.75rem; color: var(--text-3); font-size: var(--fs-base); text-align: center; }
.dropdown-item { display: flex; align-items: center; justify-content: space-between; gap: 0.75rem; padding: 0.45rem 0.65rem; border-radius: 5px; color: var(--text-2); font-size: var(--fs-base); cursor: pointer; transition: all 0.12s ease; }
.dropdown-item:hover { background: var(--surface-soft); color: var(--text-1); }
.dropdown-item.is-selected { background: var(--brand-soft); color: var(--brand); font-weight: var(--fw-medium); }
.check-icon { width: 0.875rem; height: 0.875rem; color: var(--brand); flex-shrink: 0; }
.fade-down-enter-active, .fade-down-leave-active { transition: opacity 0.2s, transform 0.2s; }
.fade-down-enter-from, .fade-down-leave-to { opacity: 0; transform: translateY(-10px); }
</style>
