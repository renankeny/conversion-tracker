import { computed } from 'vue'
import { storeToRefs } from 'pinia'
import { useTenantStore } from '~/stores/useTenantStore'

export const useAccountNavigation = () => {
  const route = useRoute()
  const { user, isOwner: isSuperAdmin, canManageAccounts, logout } = useAuth()
  const { isDark, toggle: toggleTheme } = useTheme()
  const tenantStore = useTenantStore()
  const { accounts, activeAccount, accountOptions, activeAccountId, switchVersion } = storeToRefs(tenantStore)

  const selectedAccountId = computed<string | undefined>({
    get: () => activeAccountId.value ?? undefined,
    set: (val) => {
      if (val) {
        tenantStore.switchAccount(val)
      }
    },
  })

  const loadAccounts = (force = false) => tenantStore.fetchAccounts(force)
  const refreshAccounts = () => tenantStore.fetchAccounts(true)
  const switchAccount = (id: string | number) => tenantStore.switchAccount(id)

  const isActive = (path: string) => {
    if (path === '/') return route.path === '/'
    return route.path === path || route.path.startsWith(path + '/')
  }

  const userInitials = computed(() => {
    const name = user.value?.name || user.value?.email || 'A'
    return name
      .split(/[\s@]+/)
      .map((p: string) => p.charAt(0).toUpperCase())
      .slice(0, 2)
      .join('')
  })

  return {
    route,
    user,
    logout,
    isDark,
    toggleTheme,
    accounts,
    accountOptions,
    selectedAccountId,
    activeAccount,
    switchVersion,
    isSuperAdmin,
    canManageAccounts,
    loadAccounts,
    refreshAccounts,
    switchAccount,
    isActive,
    userInitials,
  }
}
