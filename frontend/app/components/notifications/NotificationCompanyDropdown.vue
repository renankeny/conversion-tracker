<template>
  <div ref="dropdownRef" class="scope-dropdown">
    <span class="scope-label">Empresas ativas nas notificações</span>

    <button
      ref="triggerRef"
      type="button"
      class="scope-trigger"
      :class="{ 'is-open': isOpen }"
      @click="toggle"
    >
      <Building2 :size="16" class="trigger-lead-icon" />
      <span class="scope-value">{{ summaryText }}</span>
      <ChevronDown :size="16" class="trigger-chevron" :class="{ rotated: isOpen }" />
    </button>

    <Teleport to="body">
      <Transition name="fade-down">
        <div v-if="isOpen" ref="menuRef" class="scope-menu" :style="menuStyle">
          <div class="quick-actions">
            <button
              type="button"
              class="quick-btn"
              :class="{ active: mode === 'all' }"
              @click="enableAll"
            >
              Ativar em todas
            </button>
            <button
              type="button"
              class="quick-btn danger"
              :class="{ active: mode === 'none' }"
              @click="disableAll"
            >
              Desativar todas
            </button>
          </div>

          <div class="account-list">
            <div
              v-for="acc in accounts"
              :key="acc.id"
              class="account-row"
              :class="{ selected: isAccountActive(acc.id) }"
              @click="toggleAccount(acc.id)"
            >
              <BaseCheckbox
                size="sm"
                :checked="isAccountActive(acc.id)"
                @click.stop
                @change="toggleAccount(acc.id)"
              />
              <span class="account-name">{{ acc.name }}</span>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { Building2, ChevronDown } from 'lucide-vue-next'
import { onClickOutside } from '@vueuse/core'
import type { Account, AccountScopeMode } from '#shared/types'

const props = defineProps<{
  accounts: Account[]
  mode: AccountScopeMode
  selectedIds: number[]
}>()

const emit = defineEmits<{
  (e: 'change', payload: { account_scope_mode: AccountScopeMode; enabled_account_ids: number[] }): void
}>()

const dropdownRef = ref<HTMLElement | null>(null)
const triggerRef = ref<HTMLElement | null>(null)
const menuRef = ref<HTMLElement | null>(null)
const isOpen = ref(false)
const menuStyle = ref<Record<string, string>>({})

const isAccountActive = (id: number): boolean => {
  if (props.mode === 'all') return true
  if (props.mode === 'none') return false
  return props.selectedIds.includes(id)
}

const summaryText = computed(() => {
  if (props.mode === 'all') return `Ativas em todas (${props.accounts.length})`
  if (props.mode === 'none') return 'Desativadas em todas'
  const names = props.accounts
    .filter((a) => props.selectedIds.includes(a.id))
    .map((a) => a.name)
  if (names.length === 0) return 'Desativadas em todas'
  if (names.length <= 2) return names.join(', ')
  return `${names.length} de ${props.accounts.length} empresas ativas`
})

const enableAll = () => {
  emit('change', { account_scope_mode: 'all', enabled_account_ids: [] })
}

const disableAll = () => {
  emit('change', { account_scope_mode: 'none', enabled_account_ids: [] })
}

const toggleAccount = (id: number) => {
  const allIds = props.accounts.map((a) => a.id)
  const currentSet = new Set<number>(
    props.mode === 'all' ? allIds : props.mode === 'none' ? [] : props.selectedIds,
  )
  if (currentSet.has(id)) currentSet.delete(id)
  else currentSet.add(id)

  const next = Array.from(currentSet)
  if (next.length === 0) {
    disableAll()
  } else if (next.length >= allIds.length) {
    enableAll()
  } else {
    emit('change', { account_scope_mode: 'custom', enabled_account_ids: next })
  }
}

const updatePosition = () => {
  if (!triggerRef.value) return
  const rect = triggerRef.value.getBoundingClientRect()
  menuStyle.value = {
    top: `${rect.bottom + 8}px`,
    left: `${rect.left}px`,
    width: `${Math.max(rect.width, 280)}px`,
  }
}

const toggle = () => {
  if (!isOpen.value) updatePosition()
  isOpen.value = !isOpen.value
}

watch(isOpen, (open) => {
  if (!import.meta.client) return
  if (open) {
    window.addEventListener('scroll', updatePosition, true)
    window.addEventListener('resize', updatePosition)
  } else {
    window.removeEventListener('scroll', updatePosition, true)
    window.removeEventListener('resize', updatePosition)
  }
})

onBeforeUnmount(() => {
  if (!import.meta.client) return
  window.removeEventListener('scroll', updatePosition, true)
  window.removeEventListener('resize', updatePosition)
})

onClickOutside(dropdownRef, () => { isOpen.value = false }, { ignore: [menuRef] })
</script>

<style scoped>
.scope-dropdown { display: flex; flex-direction: column; gap: 0.35rem; min-width: 280px; }
.scope-label { font-size: var(--fs-sm); font-weight: var(--fw-medium); color: var(--text-3); }
.scope-trigger {
  height: 38px; display: flex; align-items: center; gap: 0.5rem; padding: 0 0.75rem;
  border-radius: 8px; border: 1px solid var(--border); background: var(--surface-2);
  color: var(--text-1); cursor: pointer; transition: all 0.15s ease; text-align: left;
}
.scope-trigger:hover { border-color: var(--brand-border); }
.scope-trigger.is-open { border-color: var(--brand); box-shadow: 0 0 0 2px var(--brand-soft); background: var(--surface-1); }
.trigger-lead-icon { color: var(--brand); flex-shrink: 0; }
.scope-value { flex: 1; font-size: var(--fs-base); font-weight: var(--fw-medium); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.trigger-chevron { color: var(--text-3); transition: transform 0.2s ease; flex-shrink: 0; }
.trigger-chevron.rotated { transform: rotate(180deg); }
.scope-menu {
  position: fixed; z-index: 9999; background: var(--surface-2); border: 1px solid var(--border);
  border-radius: 10px; box-shadow: var(--shadow-elevated); padding: 0.5rem;
}
.quick-actions { display: flex; gap: 0.375rem; padding-bottom: 0.5rem; margin-bottom: 0.375rem; border-bottom: 1px solid var(--border-soft); }
.quick-btn {
  flex: 1; padding: 0.375rem 0.5rem; border-radius: 6px; border: 1px solid var(--border-soft);
  background: var(--surface-1); color: var(--text-2); font-size: var(--fs-xs); font-weight: var(--fw-semibold);
  cursor: pointer; transition: all 0.15s ease;
}
.quick-btn:hover, .quick-btn.active { background: var(--brand-soft); color: var(--brand); border-color: var(--brand-border); }
.quick-btn.danger:hover, .quick-btn.danger.active { background: var(--danger-soft); color: var(--danger); border-color: var(--danger); }
.account-list { max-height: 240px; overflow-y: auto; display: flex; flex-direction: column; gap: 2px; }
.account-row {
  display: flex; align-items: center; gap: 0.625rem; padding: 0.45rem 0.625rem;
  border-radius: 6px; cursor: pointer; color: var(--text-2); font-size: var(--fs-base);
}
.account-row:hover { background: var(--surface-soft); color: var(--text-1); }
.account-row.selected { color: var(--text-strong); font-weight: var(--fw-medium); }
.account-name { flex: 1; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.fade-down-enter-active, .fade-down-leave-active { transition: opacity 0.15s, transform 0.15s; }
.fade-down-enter-from, .fade-down-leave-to { opacity: 0; transform: translateY(-6px); }
</style>
