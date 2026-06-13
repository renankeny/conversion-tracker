import type {
  CreatePixelInput,
  DNSVerifyResponse,
  Pixel,
  PixelDefaults,
  PixelsListResponse,
  TestEventInput,
  TestEventResponse,
  UpdatePixelInput,
} from '#shared/types'

export const usePixels = () => {
  const api = useAPI()

  const list = async (): Promise<Pixel[]> => {
    const response = await api.get<PixelsListResponse>('/admin/pixels')
    return response.data || []
  }

  const find = async (id: number): Promise<Pixel> => {
    return api.get<Pixel>(`/admin/pixels/${id}`)
  }

  const create = async (input: CreatePixelInput): Promise<Pixel> => {
    return api.post<Pixel>('/admin/pixels', input)
  }

  const update = async (id: number, input: UpdatePixelInput): Promise<Pixel> => {
    return api.put<Pixel>(`/admin/pixels/${id}`, input)
  }

  const remove = async (id: number): Promise<void> => {
    return api.delete<void>(`/admin/pixels/${id}`, {
      body: { confirm: true },
    })
  }

  const defaults = async (): Promise<PixelDefaults> => {
    return api.get<PixelDefaults>('/admin/pixels/defaults')
  }

  const verifyDNS = async (id: number): Promise<DNSVerifyResponse> => {
    return api.post<DNSVerifyResponse>(`/admin/pixels/${id}/verify-dns`)
  }

  const sendTestEvent = async (
    id: number,
    input: TestEventInput = {}
  ): Promise<TestEventResponse> => {
    return api.post<TestEventResponse>(`/admin/pixels/${id}/test-event`, input)
  }

  return {
    list,
    find,
    create,
    update,
    remove,
    defaults,
    verifyDNS,
    sendTestEvent,
  }
}
