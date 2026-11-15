import type {
  MetaAccount,
  MetaAccountInput,
  MetaAccountTestInput,
  MetaAccountTestResponse,
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

  const listByAccount = async (accountId?: number | null, force = false): Promise<MetaAccount[]> => {
    if (!accountId || accountId <= 0) {
      return list(force)
    }
    const res = await api.get<{ data: MetaAccount[] }>('/admin/meta-accounts', {
      headers: { 'X-Account-ID': String(accountId) },
    })
    return res.data || []
  }

  const get = async (id: number): Promise<MetaAccount> => {
    const acc = await api.get<MetaAccount>(`/admin/meta-accounts/${id}`)
    metaAdsStore.upsertAccount(acc)
    return acc
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

  const getOAuthStartUrl = async (
    id: number,
    opts?: { switchAccount?: boolean; popup?: boolean },
  ): Promise<string> => {
    const q = new URLSearchParams()
    if (opts?.switchAccount) q.set('switch_account', '1')
    if (opts?.popup) q.set('popup', '1')
    const qs = q.toString() ? `?${q.toString()}` : ''
    const res = await api.get<MetaOAuthStartResponse>(`/admin/meta-accounts/${id}/oauth/start${qs}`)
    return res?.url || ''
  }

  const startOAuth = async (id: number, switchAccount = false): Promise<void> => {
    const url = await getOAuthStartUrl(id, { switchAccount, popup: false })
    if (url && import.meta.client) {
      window.location.href = url
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
    refresh = false,
    accountId?: number | null,
  ): Promise<{ ad_accounts: MetaRemoteAdAccount[]; oauth_picture_url?: string }> => {
    const qs = refresh ? '?refresh=1' : ''
    return api.get<{ ad_accounts: MetaRemoteAdAccount[]; oauth_picture_url?: string }>(
      `/admin/meta-accounts/${id}/ad-accounts${qs}`,
      accountId && accountId > 0 ? { headers: { 'X-Account-ID': String(accountId) } } : undefined,
    )
  }

  const listPages = (id: number, refresh = false): Promise<{ pages: MetaRemotePage[] }> => {
    const qs = refresh ? '?refresh=1' : ''
    return api.get<{ pages: MetaRemotePage[] }>(`/admin/meta-accounts/${id}/pages${qs}`)
  }

  const listPixels = (
    id: number,
    adAccountId?: string,
    refresh = false,
    accountId?: number | null,
  ): Promise<{ pixels: MetaRemotePixel[] }> => {
    const q = new URLSearchParams()
    if (adAccountId) q.set('ad_account_id', adAccountId)
    if (refresh) q.set('refresh', '1')
    const qs = q.toString() ? `?${q.toString()}` : ''
    return api.get<{ pixels: MetaRemotePixel[] }>(
      `/admin/meta-accounts/${id}/pixels${qs}`,
      accountId && accountId > 0 ? { headers: { 'X-Account-ID': String(accountId) } } : undefined,
    )
  }

  const testEvent = (id: number, body: MetaAccountTestInput): Promise<MetaAccountTestResponse> => {
    return api.post<MetaAccountTestResponse>(`/admin/meta-accounts/${id}/test-event`, body)
  }

  return {
    list,
    listByAccount,
    get,
    create,
    update,
    remove,
    getOAuthStartUrl,
    startOAuth,
    reuseOAuth,
    disconnect,
    listAdAccounts,
    listPages,
    listPixels,
    testEvent,
  }
}
