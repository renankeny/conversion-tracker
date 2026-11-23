import type {
  APIKeysListResponse,
  CreateAPIKeyInput,
  CreateAPIKeyResponse,
  UpdateAPIKeyInput,
  UpdateAPIKeyResponse,
} from '#shared/types'

export const useAPIKeys = () => {
  const api = useAPI()

  const list = (): Promise<APIKeysListResponse> =>
    api.get<APIKeysListResponse>('/admin/api-keys')

  const create = (input: CreateAPIKeyInput): Promise<CreateAPIKeyResponse> =>
    api.post<CreateAPIKeyResponse>('/admin/api-keys', input)

  const update = (id: number, input: UpdateAPIKeyInput): Promise<UpdateAPIKeyResponse> =>
    api.put<UpdateAPIKeyResponse>(`/admin/api-keys/${id}`, input)

  const regenerate = (id: number): Promise<CreateAPIKeyResponse> =>
    api.post<CreateAPIKeyResponse>(`/admin/api-keys/${id}/regenerate`, {})

  const revoke = (id: number): Promise<void> =>
    api.delete<void>(`/admin/api-keys/${id}`)

  return { list, create, update, regenerate, revoke }
}
