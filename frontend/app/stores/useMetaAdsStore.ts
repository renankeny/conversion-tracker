import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { MetaAccount } from '#shared/types'
import { useTenantStore } from '~/stores/useTenantStore'
import { errorMessage } from '~/utils/errorMessage'

export const useMetaAdsStore = defineStore('metaAds', () => {
  const accounts = ref<MetaAccount[]>([])
  const loading = ref(false)
  const loaded = ref(false)
  const loadedAccountId = ref<string | null>(null)
  const error = ref<string | null>(null)

  let inFlightPromise: Promise<MetaAccount[]> | null = null

  async function fetch(force = false): Promise<MetaAccount[]> {
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
        const res = await api.get<{ data: MetaAccount[] }>('/admin/meta-accounts')
        const list = res.data || []
        accounts.value = list
        loaded.value = true
        return list
      } catch (e: unknown) {
        error.value = errorMessage(e, 'Erro ao carregar contas Meta Ads')
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

  function upsertAccount(acc: MetaAccount): void {
    const idx = accounts.value.findIndex((a) => a.id === acc.id)
    if (idx >= 0) {
      accounts.value[idx] = acc
    } else {
      accounts.value = [acc, ...accounts.value]
    }
  }

  function markDisconnected(id: number): MetaAccount | null {
    const idx = accounts.value.findIndex((a) => a.id === id)
    if (idx < 0) return null
    const updated: MetaAccount = {
      ...accounts.value[idx]!,
      connected: false,
      oauth_connected: false,
      oauth_user_id: '',
      oauth_name: '',
      oauth_email: '',
      token_expires_at: null,
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

