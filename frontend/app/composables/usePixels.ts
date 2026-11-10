import type {
  CreatePixelInput,
  DashboardConfig,
  Pixel,
  PixelDefaults,
  TestEventInput,
  TestEventResponse,
  UpdatePixelInput,
} from '#shared/types'
import { usePixelStore } from '~/stores/usePixelStore'

export const usePixels = () => {
  const api = useAPI()
  const pixelStore = usePixelStore()

  const list = async (force = false): Promise<Pixel[]> => {
    return pixelStore.fetch(force)
  }

  const find = async (id: number): Promise<Pixel> => {
    return api.get<Pixel>(`/admin/pixels/${id}`)
  }

  const create = async (input: CreatePixelInput): Promise<Pixel> => {
    const created = await api.post<Pixel>('/admin/pixels', input)
    pixelStore.upsertPixel(created)
    return created
  }

  const update = async (id: number, input: UpdatePixelInput): Promise<Pixel> => {
    const updated = await api.put<Pixel>(`/admin/pixels/${id}`, input)
    // Se o pixel foi transferido para outra conta, remove da lista local da conta ativa
    const activeAcc = useTenantStore().activeAccountId
    if (activeAcc && String(updated.account_id) !== String(activeAcc)) {
      pixelStore.removePixel(id)
    } else {
      pixelStore.upsertPixel(updated)
    }
    return updated
  }

  const remove = async (id: number): Promise<void> => {
    await api.delete<void>(`/admin/pixels/${id}`, {
      body: { confirm: true },
    })
    pixelStore.removePixel(id)
  }

  const defaults = async (): Promise<PixelDefaults> => {
    return api.get<PixelDefaults>('/admin/pixels/defaults')
  }

  const sendTestEvent = async (
    id: number,
    input: TestEventInput = {},
  ): Promise<TestEventResponse> => {
    return api.post<TestEventResponse>(`/admin/pixels/${id}/test-event`, input)
  }

  const regenerateToken = async (id: number): Promise<{ public_token: string }> => {
    const res = await api.post<{ public_token: string }>(`/admin/pixels/${id}/regenerate-token`, { confirm: true })
    const existing = pixelStore.pixels.find((p) => p.id === id)
    if (existing) {
      pixelStore.upsertPixel({ ...existing, public_token: res.public_token })
    }
    return res
  }

  const updateDashboardConfig = async (
    id: number,
    config: DashboardConfig,
  ): Promise<{ ok: boolean; dashboard_config: DashboardConfig }> => {
    return api.put(`/admin/pixels/${id}/dashboard-config`, config)
  }

  const generateDashboardShareLink = async (id: number): Promise<{ dashboard_share_token: string }> => {
    return api.post(`/admin/pixels/${id}/dashboard-share-link`)
  }

  const revokeDashboardShareLink = async (id: number): Promise<void> => {
    return api.delete(`/admin/pixels/${id}/dashboard-share-link`)
  }

  return {
    list,
    find,
    create,
    update,
    remove,
    defaults,
    sendTestEvent,
    regenerateToken,
    updateDashboardConfig,
    generateDashboardShareLink,
    revokeDashboardShareLink,
  }
}
