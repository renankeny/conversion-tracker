import type {
  APIKeysListResponse,
  CreateAPIKeyInput,
  CreateAPIKeyResponse,
} from '#shared/types'

// useAPIKeys encapsula CRUD dos tokens server-to-server.
//
// Members só veem/criam keys da própria conta. Owners podem operar em
// qualquer conta passando account_id no create.
export const useAPIKeys = () => {
  const api = useAPI()

  const list = (): Promise<APIKeysListResponse> =>
    api.get<APIKeysListResponse>('/admin/api-keys')

  const create = (input: CreateAPIKeyInput): Promise<CreateAPIKeyResponse> =>
    api.post<CreateAPIKeyResponse>('/admin/api-keys', input)

  // Revoke marca como revogada — não apaga a row (preserva histórico
  // last_used_at). É idempotente.
  const revoke = (id: number): Promise<void> =>
    api.delete<void>(`/admin/api-keys/${id}`)

  return { list, create, revoke }
}
