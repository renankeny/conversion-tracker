<template>
  <div ref="rootRef" class="ad-account-select">
    <label v-if="label" :for="selectId" class="select-label">{{ label }}</label>

    <button
      :id="selectId"
      ref="triggerRef"
      type="button"
      class="select-trigger"
      :class="{ 'is-open': isOpen, 'is-disabled': disabled }"
      :disabled="disabled"
      @click="toggle"
    >
      <div v-if="selectedOption" class="opt-main">
        <div class="opt-avatar" :class="{ 'is-none': selectedOption.channel === 'none' }">
          <img
            v-if="selectedOption.avatarUrl && !brokenUrls[selectedOption.avatarUrl]"
            :src="selectedOption.avatarUrl"
            :alt="selectedOption.label"
            referrerpolicy="no-referrer"
            class="opt-avatar-img"
            @error="markBroken(selectedOption.avatarUrl)"
          />
          <span v-else-if="selectedOption.channel === 'none'" class="opt-none-dot" />
          <span v-else class="opt-initials">{{ initialsOf(selectedOption.label) }}</span>
        </div>
        <div class="opt-texts">
          <span class="opt-title">{{ selectedOption.label }}</span>
          <span v-if="selectedOption.sublabel" class="opt-sub">{{ selectedOption.sublabel }}</span>
        </div>
        <span v-if="selectedOption.badge" class="opt-badge">{{ selectedOption.badge }}</span>
      </div>

      <span v-else class="opt-placeholder">
        {{ loading ? 'Carregando contas de anúncios...' : (placeholder || 'Selecione uma conta de anúncios...') }}
      </span>

      <svg class="chevron-icon" :class="{ 'is-rotated': isOpen }" viewBox="0 0 20 20" fill="currentColor">
        <path fill-rule="evenodd" d="M5.23 7.21a.75.75 0 011.06.02L10 11.085l3.71-3.855a.75.75 0 111.08 1.04l-4.25 4.4a.75.75 0 01-1.08 0l-4.25-4.4a.75.75 0 01.02-1.06z" clip-rule="evenodd" />
      </svg>
    </button>

    <small v-if="hint" class="select-hint">{{ hint }}</small>

    <Teleport to="body">
      <Transition name="fade-down">
        <ul v-if="isOpen" ref="menuRef" class="select-menu" :style="menuStyle">
          <li v-if="!options.length" class="select-empty">
            Nenhuma conta de anúncios disponível
          </li>
          <li
            v-for="opt in options"
            :key="String(opt.value)"
            class="select-item"
            :class="{ 'is-selected': isSelected(opt.value) }"
            @click="choose(opt.value)"
          >
            <div class="opt-main">
              <div class="opt-avatar" :class="{ 'is-none': opt.channel === 'none' }">
                <img
                  v-if="opt.avatarUrl && !brokenUrls[opt.avatarUrl]"
                  :src="opt.avatarUrl"
                  :alt="opt.label"
                  referrerpolicy="no-referrer"
                  class="opt-avatar-img"
                  @error="markBroken(opt.avatarUrl)"
                />
                <span v-else-if="opt.channel === 'none'" class="opt-none-dot" />
                <span v-else class="opt-initials">{{ initialsOf(opt.label) }}</span>
              </div>
              <div class="opt-texts">
                <span class="opt-title">{{ opt.label }}</span>
                <span v-if="opt.sublabel" class="opt-sub">{{ opt.sublabel }}</span>
              </div>
              <span v-if="opt.badge" class="opt-badge">{{ opt.badge }}</span>
            </div>
            <svg v-if="isSelected(opt.value)" class="check-icon" viewBox="0 0 20 20" fill="currentColor">
              <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd" />
            </svg>
          </li>
        </ul>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, useId, watch, onBeforeUnmount } from 'vue'
import { onClickOutside } from '@vueuse/core'

export interface AdAccountSelectOption {
  value: any
  label: string
  sublabel?: string
  avatarUrl?: string
  channel?: 'meta' | 'google_ads' | 'none'
  badge?: string
}

const props = defineProps<{
  modelValue?: any
  label?: string
  hint?: string
  placeholder?: string
  disabled?: boolean
  loading?: boolean
  options: AdAccountSelectOption[]
}>()

const emit = defineEmits<{
  'update:modelValue': [value: any]
}>()

const selectId = useId()
const rootRef = ref<HTMLElement | null>(null)
const triggerRef = ref<HTMLElement | null>(null)
const menuRef = ref<HTMLElement | null>(null)
const isOpen = ref(false)
const menuStyle = ref<Record<string, string>>({})
const brokenUrls = reactive<Record<string, boolean>>({})

const markBroken = (url?: string) => {
  if (url) brokenUrls[url] = true
}

const initialsOf = (name: string): string => {
  const parts = (name || '').trim().split(/\s+/).filter(Boolean)
  if (!parts.length) return 'AD'
  if (parts.length === 1) return parts[0]!.slice(0, 2).toUpperCase()
  return (parts[0]![0]! + parts[1]![0]!).toUpperCase()
}

const isSelected = (optVal: any): boolean => {
  if (props.modelValue === optVal) return true
  if (props.modelValue == null && (optVal === '' || optVal === null)) return props.modelValue === optVal
  return String(props.modelValue ?? '') === String(optVal ?? '')
}

const selectedOption = computed(() => props.options.find((o) => isSelected(o.value)))

const updateMenuPosition = () => {
  if (!triggerRef.value) return
  const rect = triggerRef.value.getBoundingClientRect()
  menuStyle.value = {
    top: `${rect.bottom + 6}px`,
    left: `${rect.left}px`,
    width: `${rect.width}px`,
  }
}

const toggle = () => {
  if (props.disabled) return
  if (!isOpen.value) updateMenuPosition()
  isOpen.value = !isOpen.value
}

const choose = (val: any) => {
  emit('update:modelValue', val)
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

onClickOutside(rootRef, () => { isOpen.value = false }, { ignore: [menuRef] })
</script>

<style scoped>
.ad-account-select { position: relative; display: flex; flex-direction: column; gap: 0.375rem; }
.select-label { font-size: var(--fs-base); font-weight: var(--fw-medium); color: var(--text-2); }
.select-hint { font-size: var(--fs-sm); color: var(--text-3); line-height: 1.35; }
.select-trigger { width: 100%; min-height: 44px; display: flex; align-items: center; justify-content: space-between; gap: 0.625rem; padding: 0.45rem 0.75rem; border-radius: 8px; border: 1px solid var(--border); background: var(--surface-2); cursor: pointer; text-align: left; transition: all 0.15s ease; }
.select-trigger:hover:not(.is-disabled) { border-color: var(--brand-border); }
.select-trigger.is-open { border-color: var(--brand); box-shadow: 0 0 0 2px var(--brand-soft); background: var(--surface-1); }
.select-trigger.is-disabled { opacity: 0.55; cursor: not-allowed; }
.opt-main { display: flex; align-items: center; gap: 0.625rem; flex: 1; min-width: 0; }
.opt-avatar { width: 28px; height: 28px; border-radius: 7px; background: var(--brand-soft); border: 1px solid var(--brand-border); display: flex; align-items: center; justify-content: center; overflow: hidden; flex-shrink: 0; }
.opt-avatar.is-none { background: var(--surface-soft); border: 1px dashed var(--border); }
.opt-avatar-img { width: 100%; height: 100%; object-fit: cover; display: block; }
.opt-initials { font-size: var(--fs-xs); font-weight: var(--fw-bold); color: var(--brand); }
.opt-none-dot { width: 8px; height: 8px; border-radius: 50%; background: var(--text-3); opacity: 0.5; }
.opt-texts { display: flex; flex-direction: column; min-width: 0; flex: 1; line-height: 1.2; }
.opt-title { font-size: var(--fs-md); font-weight: var(--fw-medium); color: var(--text-1); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.opt-sub { font-family: var(--font-mono); font-size: var(--fs-xs); color: var(--text-3); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; margin-top: 0.1rem; }
.opt-badge { font-size: var(--fs-2xs); font-weight: var(--fw-semibold); padding: 0.15rem 0.45rem; border-radius: 999px; background: var(--surface-soft); color: var(--text-2); border: 1px solid var(--border-soft); white-space: nowrap; }
.opt-placeholder { font-size: var(--fs-base); color: var(--text-3); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.chevron-icon { width: 1rem; height: 1rem; color: var(--text-3); transition: transform 0.2s ease; flex-shrink: 0; }
.chevron-icon.is-rotated { transform: rotate(180deg); }
.select-menu { position: fixed; max-height: 280px; overflow-y: auto; background: var(--surface-2); border: 1px solid var(--border); border-radius: 10px; box-shadow: var(--shadow-elevated); z-index: 9999; list-style: none; padding: 0.35rem; margin: 0; }
.select-empty { padding: 0.65rem 0.75rem; color: var(--text-3); font-size: var(--fs-base); text-align: center; }
.select-item { display: flex; align-items: center; justify-content: space-between; gap: 0.625rem; padding: 0.45rem 0.6rem; border-radius: 7px; cursor: pointer; transition: background 0.12s ease; }
.select-item:hover { background: var(--surface-soft); }
.select-item.is-selected { background: var(--brand-soft); }
.select-item.is-selected .opt-title { color: var(--brand); font-weight: var(--fw-semibold); }
.check-icon { width: 0.9rem; height: 0.9rem; color: var(--brand); flex-shrink: 0; }
.fade-down-enter-active, .fade-down-leave-active { transition: opacity 0.18s, transform 0.18s; }
.fade-down-enter-from, .fade-down-leave-to { opacity: 0; transform: translateY(-6px); }
</style>

