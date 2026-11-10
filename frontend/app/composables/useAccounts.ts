import type {
  Account,
  AccountsListResponse,
  CreateAccountInput,
  UpdateAccountInput,
} from '#shared/types'
import { useTenantStore } from '~/stores/useTenantStore'

export const useAccounts = () => {
  const api = useAPI()
  const tenantStore = useTenantStore()

  const list = async (): Promise<AccountsListResponse> => {
    const res = await api.get<AccountsListResponse>('/admin/accounts')
    tenantStore.syncAccounts(res.data || [])
    return res
  }

  const get = (id: number) => api.get<Account>(`/admin/accounts/${id}`)

  const create = async (input: CreateAccountInput): Promise<Account> => {
    const created = await api.post<Account>('/admin/accounts', input)
    tenantStore.upsertAccount(created)
    return created
  }

  const update = async (id: number, input: UpdateAccountInput): Promise<Account> => {
    const updated = await api.put<Account>(`/admin/accounts/${id}`, input)
    tenantStore.upsertAccount(updated)
    return updated
  }

  const updateEventLabels = (id: number, overrides: Record<string, string>) =>
    api.put<{ ok: boolean; event_label_overrides: Record<string, string> }>(
      `/admin/accounts/${id}/event-labels`,
      { event_label_overrides: overrides },
    )

  const getEventsFilter = () =>
    api.get<{ events_filter: Record<string, string[]> }>('/admin/events-filter')

  const updateEventsFilter = (pixel: string, events: string[]) =>
    api.put<{ ok: boolean; pixel: string; events: string[] }>('/admin/events-filter', {
      pixel,
      events,
    })

  const remove = async (id: number): Promise<void> => {
    await api.delete(`/admin/accounts/${id}`, { body: { confirm: true } })
    tenantStore.removeAccount(id)
  }

  return { list, get, create, update, updateEventLabels, getEventsFilter, updateEventsFilter, remove }
}
