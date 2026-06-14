import type { UrlMapping, UrlMappingInput, UrlMappingsListResponse } from '#shared/types'

export const useUrlMappings = () => {
  const api = useAPI()

  const list = (): Promise<UrlMappingsListResponse> =>
    api.get<UrlMappingsListResponse>('/admin/url-mappings')

  const create = (input: UrlMappingInput): Promise<UrlMapping> =>
    api.post<UrlMapping>('/admin/url-mappings', input)

  const update = (id: number, input: UrlMappingInput): Promise<UrlMapping> =>
    api.put<UrlMapping>(`/admin/url-mappings/${id}`, input)

  const remove = (id: number): Promise<void> =>
    api.delete<void>(`/admin/url-mappings/${id}`)

  return { list, create, update, remove }
}
