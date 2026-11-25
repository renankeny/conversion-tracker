import type {
  GoogleAdsAccount,
  GoogleAdsAccountInput,
  GoogleAdsOAuthStartResponse,
  GoogleAdsRemoteCustomer,
  GoogleAdsRemoteConversionAction,
} from '#shared/types'
import { useGoogleAdsStore } from '~/stores/useGoogleAdsStore'

export const useGoogleAdsAccounts = () => {
  const api = useAPI()
  const googleAdsStore = useGoogleAdsStore()

  const list = async (force = false): Promise<GoogleAdsAccount[]> => {
    return googleAdsStore.fetch(force)
  }

  const listByAccount = async (accountId?: number | null, force = false): Promise<GoogleAdsAccount[]> => {
    if (!accountId || accountId <= 0) {
      return list(force)
    }
    const res = await api.get<{ data: GoogleAdsAccount[] }>('/admin/google-ads-accounts', {
      headers: { 'X-Account-ID': String(accountId) },
    })
    return res.data || []
  }

  const get = async (id: number): Promise<GoogleAdsAccount> => {
    const acc = await api.get<GoogleAdsAccount>(`/admin/google-ads-accounts/${id}`)
    googleAdsStore.upsertAccount(acc)
    return acc
  }

  const create = async (input: GoogleAdsAccountInput): Promise<GoogleAdsAccount> => {
    const created = await api.post<GoogleAdsAccount>('/admin/google-ads-accounts', input)
    googleAdsStore.upsertAccount(created)
    return created
  }

  const update = async (id: number, input: GoogleAdsAccountInput): Promise<GoogleAdsAccount> => {
    const updated = await api.put<GoogleAdsAccount>(`/admin/google-ads-accounts/${id}`, input)
    googleAdsStore.upsertAccount(updated)
    return updated
  }

  const remove = async (id: number): Promise<{ deleted: boolean }> => {
    const res = await api.delete<{ deleted: boolean }>(`/admin/google-ads-accounts/${id}`)
    googleAdsStore.removeAccount(id)
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
    const res = await api.get<GoogleAdsOAuthStartResponse>(
      `/admin/google-ads-accounts/${id}/oauth/start${qs}`,
    )
    return res?.url || ''
  }

  const startOAuth = async (id: number, switchAccount = false): Promise<void> => {
    const url = await getOAuthStartUrl(id, { switchAccount, popup: false })
    if (url && import.meta.client) {
      window.location.href = url
    }
  }

  const reuseOAuth = async (id: number, sourceAccountId: number): Promise<GoogleAdsAccount> => {
    const updated = await api.post<GoogleAdsAccount>(`/admin/google-ads-accounts/${id}/reuse-oauth`, {
      source_account_id: sourceAccountId,
    })
    googleAdsStore.upsertAccount(updated)
    return updated
  }

  const listCustomers = (
    id: number,
    refresh = false,
    accountId?: number | null,
  ): Promise<{ customers: GoogleAdsRemoteCustomer[]; oauth_picture_url?: string; warning?: string }> => {
    const qs = refresh ? '?refresh=1' : ''
    return api.get<{ customers: GoogleAdsRemoteCustomer[]; oauth_picture_url?: string; warning?: string }>(
      `/admin/google-ads-accounts/${id}/customers${qs}`,
      accountId && accountId > 0 ? { headers: { 'X-Account-ID': String(accountId) } } : undefined,
    )
  }

  const listConversionActions = (
    id: number,
    customerId?: string,
    loginCustomerId?: string,
    refresh = false,
  ): Promise<{ conversion_actions: GoogleAdsRemoteConversionAction[]; conversion_owner_id?: string; warning?: string }> => {
    const q = new URLSearchParams()
    if (customerId) q.set('customer_id', customerId)
    if (loginCustomerId) q.set('login_customer_id', loginCustomerId)
    if (refresh) q.set('refresh', '1')
    const qs = q.toString() ? `?${q.toString()}` : ''
    return api.get<{ conversion_actions: GoogleAdsRemoteConversionAction[]; conversion_owner_id?: string; warning?: string }>(
      `/admin/google-ads-accounts/${id}/conversion-actions${qs}`,
    )
  }

  const createConversionAction = (
    id: number,
    body: {
      customer_id: string
      login_customer_id?: string
      event_type: string
      name?: string
      currency?: string
    },
  ): Promise<{ conversion_action: GoogleAdsRemoteConversionAction }> => {
    return api.post(`/admin/google-ads-accounts/${id}/conversion-actions`, body)
  }

  const disconnect = async (id: number): Promise<{ disconnected: boolean }> => {
    const res = await api.post<{ disconnected: boolean }>(
      `/admin/google-ads-accounts/${id}/disconnect`,
      {},
    )
    googleAdsStore.markDisconnected(id)
    return res
  }

  const testEvent = (
    id: number,
    body: {
      event_type: string
      conversion_action_id?: string
      action_type?: string
      owner_customer_id?: string
      customer_id?: string
      login_customer_id?: string
      gclid?: string
      wbraid?: string
      gbraid?: string
      value?: number
      currency?: string
      order_id?: string
      email?: string
      phone?: string
      first_name?: string
      last_name?: string
      validate_only?: boolean
    },
  ): Promise<{
    validate_only: boolean
    event_type: string
    conversion_action_id: string
    results: Array<{
      customer_id: string
      status_code: number
      response: string
      request_body: string
      error?: string
    }>
  }> => {
    return api.post(`/admin/google-ads-accounts/${id}/test-event`, body)
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
    listCustomers,
    listConversionActions,
    createConversionAction,
    disconnect,
    testEvent,
  }
}
