import type {
  MetaAccount,
  MetaAccountInput,
  MetaOAuthStartResponse,
  MetaRemoteAdAccount,
  MetaRemotePage,
  MetaRemotePixel,
} from '#shared/types'
import { useMetaAdsStore } from '~/stores/useMetaAdsStore'

export const useMetaAccounts = () => {
  const api = useAPI()
  const metaAdsStore = useMetaAdsStore()

  const list = async (force = false): Promise<MetaAccount[]> => {
    return metaAdsStore.fetch(force)
  }

  const get = (id: number): Promise<MetaAccount> => {
    return api.get<MetaAccount>(`/admin/meta-accounts/${id}`)
  }

  const create = async (input: MetaAccountInput): Promise<MetaAccount> => {
    const created = await api.post<MetaAccount>('/admin/meta-accounts', input)
    metaAdsStore.upsertAccount(created)
    return created
  }

  const update = async (id: number, input: MetaAccountInput): Promise<MetaAccount> => {
    const updated = await api.put<MetaAccount>(`/admin/meta-accounts/${id}`, input)
    metaAdsStore.upsertAccount(updated)
    return updated
  }

  const remove = async (id: number): Promise<{ deleted: boolean }> => {
    const res = await api.delete<{ deleted: boolean }>(`/admin/meta-accounts/${id}`)
    metaAdsStore.removeAccount(id)
    return res
  }

  const startOAuth = async (id: number, switchAccount = false): Promise<void> => {
    const qs = switchAccount ? '?switch_account=1' : ''
    const res = await api.get<MetaOAuthStartResponse>(`/admin/meta-accounts/${id}/oauth/start${qs}`)
    if (res?.url && import.meta.client) {
      window.location.href = res.url
    }
  }

  const reuseOAuth = async (id: number, sourceAccountId: number): Promise<MetaAccount> => {
    const updated = await api.post<MetaAccount>(`/admin/meta-accounts/${id}/reuse-oauth`, {
      source_account_id: sourceAccountId,
    })
    metaAdsStore.upsertAccount(updated)
    return updated
  }

  const disconnect = async (id: number): Promise<{ disconnected: boolean }> => {
    const res = await api.post<{ disconnected: boolean }>(`/admin/meta-accounts/${id}/disconnect`, {})
    metaAdsStore.markDisconnected(id)
    return res
  }

  const listAdAccounts = (
    id: number,
  ): Promise<{ ad_accounts: MetaRemoteAdAccount[]; oauth_picture_url?: string }> => {
    return api.get<{ ad_accounts: MetaRemoteAdAccount[]; oauth_picture_url?: string }>(
      `/admin/meta-accounts/${id}/ad-accounts`,
    )
  }

  const listPages = (id: number): Promise<{ pages: MetaRemotePage[] }> => {
    return api.get<{ pages: MetaRemotePage[] }>(`/admin/meta-accounts/${id}/pages`)
  }

  const listPixels = (id: number, adAccountId?: string): Promise<{ pixels: MetaRemotePixel[] }> => {
    const qs = adAccountId ? `?ad_account_id=${encodeURIComponent(adAccountId)}` : ''
    return api.get<{ pixels: MetaRemotePixel[] }>(`/admin/meta-accounts/${id}/pixels${qs}`)
  }

  return {
    list,
    get,
    create,
    update,
    remove,
    startOAuth,
    reuseOAuth,
    disconnect,
    listAdAccounts,
    listPages,
    listPixels,
  }
}

