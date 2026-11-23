import type { TrackableLink, TrackableLinkInput, TrackableLinksOverview } from '#shared/types'

const buildLinkStatsQuery = (params: Record<string, string | undefined>): string => {
  const entries = Object.entries(params).filter(([, v]) => v !== undefined && v !== '')
  if (!entries.length) return ''
  return '?' + entries.map(([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(v!)}`).join('&')
}

export const useTrackableLinks = () => {
  const api = useAPI()

  const listByPixel = (pixelId: number) =>
    api.get<TrackableLinksOverview>(`/admin/pixels/${pixelId}/trackable-links`)

  const create = (pixelId: number, input: TrackableLinkInput) =>
    api.post<TrackableLink>(`/admin/pixels/${pixelId}/trackable-links`, input)

  const update = (pixelId: number, linkId: number, input: TrackableLinkInput) =>
    api.put<TrackableLink>(`/admin/pixels/${pixelId}/trackable-links/${linkId}`, input)

  const remove = (pixelId: number, linkId: number) =>
    api.del<{ deleted: boolean }>(`/admin/pixels/${pixelId}/trackable-links/${linkId}`)

  const dashboardStats = (
    filters: { pixel_name?: string; from?: string; to?: string } = {},
    signal?: AbortSignal,
  ) =>
    api.get<TrackableLinksOverview>(
      `/admin/trackable-links/stats${buildLinkStatsQuery(filters)}`,
      { signal },
    )

  return {
    listByPixel,
    create,
    update,
    remove,
    dashboardStats,
  }
}
