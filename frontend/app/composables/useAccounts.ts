import type {
  Account,
  AccountsListResponse,
  CreateAccountInput,
  UpdateAccountInput,
} from '#shared/types'

// useAccounts encapsula o CRUD de contas (multi-tenant).
// Apenas owners conseguem criar/editar/remover. Members só veem a própria
// conta — o backend já aplica esse filtro automaticamente.
export const useAccounts = () => {
  const api = useAPI()

  const list = (): Promise<AccountsListResponse> =>
    api.get<AccountsListResponse>('/admin/accounts')

  const get = (id: number): Promise<Account> =>
    api.get<Account>(`/admin/accounts/${id}`)

  const create = (input: CreateAccountInput): Promise<Account> =>
    api.post<Account>('/admin/accounts', input)

  const update = (id: number, input: UpdateAccountInput): Promise<Account> =>
    api.put<Account>(`/admin/accounts/${id}`, input)

  const remove = (id: number): Promise<void> =>
    api.delete<void>(`/admin/accounts/${id}`, { body: { confirm: true } })

  return {
    list,
    get,
    create,
    update,
    remove,
  }
}
