import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import type { Account } from '#shared/types'
import { setBrowserCookie } from '~/utils/cookies'
import { useAuthStore } from '~/stores/useAuthStore'
import { useDashboardStore } from '~/stores/useDashboardStore'
import { usePixelStore } from '~/stores/usePixelStore'
import { useGoogleAdsStore } from '~/stores/useGoogleAdsStore'
import { useCustomWebhooksStore } from '~/stores/customWebhooks'
import { useSystemAlertsStore } from '~/stores/useSystemAlertsStore'
import { useLiveEventsStore } from '~/stores/useLiveEventsStore'

export const useTenantStore = defineStore('tenant', () => {
  const accounts = ref<Account[]>([])
  const loaded = ref(false)
  const loading = ref(false)
  const activeAccountId = ref<string | null>(null)
  const switchVersion = ref(0)

  let inFlightPromise: Promise<Account[]> | null = null

  const activeAccount = computed<Account | null>(() => {
    if (!accounts.value.length) return null
    return (
      accounts.value.find((a) => String(a.id) === String(activeAccountId.value)) ||
      accounts.value[0] ||
      null
    )
  })

  const accountOptions = computed(() =>
    accounts.value.map((a) => ({
      value: String(a.id),
      label: a.name,
    })),
  )

  function hydrate(initialAccountId?: string | null): void {
    if (initialAccountId) {
      activeAccountId.value = String(initialAccountId)
    }
  }

  function setActiveAccountId(id: string | number | null, maxAge = 60 * 60 * 24 * 30): void {
    const val = id === null || id === undefined || id === '' ? null : String(id)
    activeAccountId.value = val
    setBrowserCookie('active_account_id', val, maxAge)
  }

  function syncAccounts(list: Account[]): void {
    accounts.value = list
    loaded.value = true
    if (!list.length) return

    if (!activeAccountId.value && list[0]) {
      setActiveAccountId(list[0].id)
      return
    }
    if (activeAccountId.value) {
      const exists = list.some((a) => String(a.id) === String(activeAccountId.value))
      if (!exists && list[0]) {
        setActiveAccountId(list[0].id)
      }
    }
  }

  async function fetchAccounts(force = false): Promise<Account[]> {
    const authStore = useAuthStore()
    if (!authStore.canManageAccounts) return accounts.value
    if (loaded.value && !force) return accounts.value
    if (inFlightPromise && !force) return inFlightPromise

    loading.value = true
    const api = useAPI()
    inFlightPromise = (async () => {
      try {
        const res = await api.get<{ data: Account[] }>('/admin/accounts')
        const list = res.data || []
        syncAccounts(list)
        return list
      } catch {
        return accounts.value
      } finally {
        loading.value = false
        inFlightPromise = null
      }
    })()

    return inFlightPromise
  }

  function upsertAccount(acc: Account): void {
    const idx = accounts.value.findIndex((a) => a.id === acc.id)
    if (idx >= 0) {
      accounts.value[idx] = acc
    } else {
      accounts.value = [...accounts.value, acc]
    }
  }

  function removeAccount(id: number): void {
    accounts.value = accounts.value.filter((a) => a.id !== id)
    if (String(activeAccountId.value) === String(id)) {
      const fallback = accounts.value[0]?.id ? String(accounts.value[0].id) : '1'
      switchAccount(fallback)
    }
  }

  function switchAccount(id: string | number): void {
    const nextId = String(id)
    if (!nextId || nextId === String(activeAccountId.value)) return

    setActiveAccountId(nextId)

    const dashboardStore = useDashboardStore()
    dashboardStore.setSelectedPixel('')
    if (import.meta.client) {
      localStorage.removeItem('trackfind_selected_pixel')
    }

    const pixelStore = usePixelStore()
    pixelStore.invalidate()
    useGoogleAdsStore().invalidate()
    useCustomWebhooksStore().invalidate()
    useSystemAlertsStore().invalidate()

    const liveStore = useLiveEventsStore()
    liveStore.reconnectForAccount(Number(nextId) || 0)

    pixelStore.fetch(true).catch(() => {})
    switchVersion.value++
  }

  function reset(): void {
    accounts.value = []
    loaded.value = false
    loading.value = false
    activeAccountId.value = null
    inFlightPromise = null
  }

  return {
    accounts,
    loaded,
    loading,
    activeAccountId,
    switchVersion,
    activeAccount,
    accountOptions,
    hydrate,
    setActiveAccountId,
    syncAccounts,
    fetchAccounts,
    upsertAccount,
    removeAccount,
    switchAccount,
    reset,
  }
})

