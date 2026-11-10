import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { GoogleAdsAccount } from '#shared/types'
import { useTenantStore } from '~/stores/useTenantStore'

export const useGoogleAdsStore = defineStore('googleAds', () => {
  const accounts = ref<GoogleAdsAccount[]>([])
  const loading = ref(false)
  const loaded = ref(false)
  const loadedAccountId = ref<string | null>(null)
  const error = ref<string | null>(null)

  let inFlightPromise: Promise<GoogleAdsAccount[]> | null = null

  async function fetch(force = false): Promise<GoogleAdsAccount[]> {
    const tenantStore = useTenantStore()
    const currentAcc = tenantStore.activeAccountId || ''

    if (!force && loaded.value && loadedAccountId.value === currentAcc) {
      return accounts.value
    }
    if (!force && inFlightPromise && loadedAccountId.value === currentAcc) {
      return inFlightPromise
    }

    loading.value = true
    error.value = null
    loadedAccountId.value = currentAcc

    const api = useAPI()
    inFlightPromise = (async () => {
      try {
        const res = await api.get<{ data: GoogleAdsAccount[] }>('/admin/google-ads-accounts')
        const list = res.data || []
        accounts.value = list
        loaded.value = true
        return list
      } catch (e: any) {
        error.value = e.message ?? 'Erro ao carregar contas Google Ads'
        throw e
      } finally {
        loading.value = false
        inFlightPromise = null
      }
    })()

    return inFlightPromise
  }

  function invalidate(): void {
    loaded.value = false
    loadedAccountId.value = null
    accounts.value = []
    error.value = null
    inFlightPromise = null
  }

  function upsertAccount(acc: GoogleAdsAccount): void {
    const idx = accounts.value.findIndex((a) => a.id === acc.id)
    if (idx >= 0) {
      accounts.value[idx] = acc
    } else {
      accounts.value = [acc, ...accounts.value]
    }
  }

  function markDisconnected(id: number): GoogleAdsAccount | null {
    const idx = accounts.value.findIndex((a) => a.id === id)
    if (idx < 0) return null
    const updated: GoogleAdsAccount = {
      ...accounts.value[idx]!,
      connected: false,
      oauth_connected: false,
      oauth_email: '',
    }
    accounts.value[idx] = updated
    return updated
  }

  function removeAccount(id: number): void {
    accounts.value = accounts.value.filter((a) => a.id !== id)
  }

  return {
    accounts,
    loading,
    loaded,
    loadedAccountId,
    error,
    fetch,
    invalidate,
    upsertAccount,
    markDisconnected,
    removeAccount,
  }
})

