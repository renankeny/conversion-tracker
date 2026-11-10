<template>
  <div ref="dropdownRef" class="multi-dropdown">
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
      <span class="dropdown-value" :class="{ 'is-placeholder': !modelValue.length }">
        {{ summaryLabel }}
      </span>
      <svg class="dropdown-icon" :class="{ 'is-rotated': isOpen }" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
        <path fill-rule="evenodd" d="M5.23 7.21a.75.75 0 011.06.02L10 11.085l3.71-3.855a.75.75 0 111.08 1.04l-4.25 4.4a.75.75 0 01-1.08 0l-4.25-4.4a.75.75 0 01.02-1.06z" clip-rule="evenodd" />
      </svg>
    </div>

    <Teleport to="body">
      <Transition name="fade-down">
        <ul v-if="isOpen" ref="menuRef" class="dropdown-menu" :style="menuStyle">
          <li v-if="options.length === 0" class="dropdown-empty">
            Nenhuma opção
          </li>
          <li v-if="modelValue.length" class="dropdown-item dropdown-clear" @click="clearAll">
            <span class="item-label">Limpar seleção</span>
          </li>
          <li
            v-for="option in options"
            :key="option.value"
            class="dropdown-item"
            :class="{ 'is-selected': isChecked(option.value) }"
            @click="toggleOption(option.value)"
          >
            <BaseCheckbox
              size="sm"
              :checked="isChecked(option.value)"
              @click.stop
              @change="toggleOption(option.value)"
            />
            <span class="item-label">{{ option.label }}</span>
          </li>
        </ul>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, useId, watch, onBeforeUnmount } from 'vue'
import { onClickOutside } from '@vueuse/core'

interface Option {
  value: string
  label: string
}

interface Props {
  modelValue: string[]
  label?: string
  options: Option[]
  placeholder?: string
  disabled?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: () => [],
})

const emit = defineEmits<{
  'update:modelValue': [value: string[]]
}>()

const dropdownId = useId()
const dropdownRef = ref<HTMLElement | null>(null)
const triggerRef = ref<HTMLElement | null>(null)
const menuRef = ref<HTMLElement | null>(null)
const isOpen = ref(false)
const menuStyle = ref<Record<string, string>>({})

const labelByValue = computed(() => {
  const map = new Map<string, string>()
  for (const opt of props.options) map.set(opt.value, opt.label)
  return map
})

const summaryLabel = computed(() => {
  if (!props.modelValue.length) return props.placeholder || 'Todos'
  return props.modelValue.map((v) => labelByValue.value.get(v) || v).join(', ')
})

const isChecked = (value: string) => props.modelValue.includes(value)

const toggleOption = (value: string) => {
  const next = isChecked(value)
    ? props.modelValue.filter((v) => v !== value)
    : [...props.modelValue, value]
  emit('update:modelValue', next)
}

const clearAll = () => emit('update:modelValue', [])

const updateMenuPosition = () => {
  if (!triggerRef.value) return
  const rect = triggerRef.value.getBoundingClientRect()
  menuStyle.value = {
    top: `${rect.bottom + 8}px`,
    left: `${rect.left}px`,
    width: `${rect.width}px`,
  }
}

const toggle = () => {
  if (props.disabled) return
  if (!isOpen.value) updateMenuPosition()
  isOpen.value = !isOpen.value
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
  window.removeEventListener('scroll', updateMenuPosition, true)
  window.removeEventListener('resize', updateMenuPosition)
})

onClickOutside(dropdownRef, () => {
  isOpen.value = false
}, { ignore: [menuRef] })
</script>

<style scoped>
.multi-dropdown { position: relative; display: flex; flex-direction: column; gap: 0.375rem; }
.dropdown-label { font-size: var(--fs-base); font-weight: var(--fw-medium); color: var(--text-2); letter-spacing: -0.01em; }
.dropdown-trigger { width: 100%; height: 36px; display: flex; align-items: center; justify-content: space-between; padding: 0.45rem 0.75rem; border-radius: 6px; border: 1px solid var(--border); background: var(--surface-2); cursor: pointer; transition: all 0.15s ease; user-select: none; }
.dropdown-trigger:hover:not(.is-disabled) { border-color: var(--brand-border); }
.dropdown-trigger.is-open { border-color: var(--brand); box-shadow: 0 0 0 2px var(--brand-soft); background: var(--surface-1); }
.dropdown-trigger.is-disabled { opacity: 0.5; cursor: not-allowed; }
.dropdown-value { color: var(--text-1); font-size: var(--fs-md); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.dropdown-value.is-placeholder { color: var(--text-3); font-size: var(--fs-base); }
.dropdown-icon { width: 1rem; height: 1rem; color: var(--text-3); transition: transform 0.2s ease; flex-shrink: 0; }
.dropdown-icon.is-rotated { transform: rotate(180deg); }
.dropdown-menu { position: fixed; max-height: 220px; overflow-y: auto; background: var(--surface-2); backdrop-filter: blur(12px); border: 1px solid var(--border); border-radius: 8px; box-shadow: var(--shadow-elevated); z-index: 9999; list-style: none; padding: 0.35rem; margin: 0; }
.dropdown-empty { padding: 0.5rem 0.75rem; color: var(--text-3); font-size: var(--fs-base); text-align: center; }
.dropdown-item { display: flex; align-items: center; gap: 0.5rem; padding: 0.4rem 0.65rem; border-radius: 5px; color: var(--text-2); font-size: var(--fs-base); cursor: pointer; transition: all 0.12s ease; }
.dropdown-item:hover { background: var(--surface-soft); color: var(--text-1); }
.dropdown-item.is-selected { color: var(--brand); }
.dropdown-clear { color: var(--text-3); font-size: var(--fs-sm); padding-bottom: 0.35rem; border-bottom: 1px solid var(--border-soft); margin-bottom: 0.25rem; }
.dropdown-clear:hover { color: var(--danger); background: var(--danger-soft); }
.fade-down-enter-active, .fade-down-leave-active { transition: opacity 0.2s, transform 0.2s; }
.fade-down-enter-from, .fade-down-leave-to { opacity: 0; transform: translateY(-10px); }
</style>
