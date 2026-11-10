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

  const get = (id: number): Promise<GoogleAdsAccount> => {
    return api.get<GoogleAdsAccount>(`/admin/google-ads-accounts/${id}`)
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

  const startOAuth = async (id: number): Promise<void> => {
    const res = await api.get<GoogleAdsOAuthStartResponse>(
      `/admin/google-ads-accounts/${id}/oauth/start`,
    )
    if (res?.url && import.meta.client) {
      window.location.href = res.url
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
  ): Promise<{ customers: GoogleAdsRemoteCustomer[]; oauth_picture_url?: string; warning?: string }> => {
    return api.get<{ customers: GoogleAdsRemoteCustomer[]; oauth_picture_url?: string; warning?: string }>(
      `/admin/google-ads-accounts/${id}/customers`,
    )
  }

  const listConversionActions = (
    id: number,
    customerId?: string,
    loginCustomerId?: string,
  ): Promise<{ conversion_actions: GoogleAdsRemoteConversionAction[]; warning?: string }> => {
    const q = new URLSearchParams()
    if (customerId) q.set('customer_id', customerId)
    if (loginCustomerId) q.set('login_customer_id', loginCustomerId)
    const qs = q.toString() ? `?${q.toString()}` : ''
    return api.get<{ conversion_actions: GoogleAdsRemoteConversionAction[]; warning?: string }>(
      `/admin/google-ads-accounts/${id}/conversion-actions${qs}`,
    )
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
    get,
    create,
    update,
    remove,
    startOAuth,
    reuseOAuth,
    listCustomers,
    listConversionActions,
    disconnect,
    testEvent,
  }
}
