import type { TrackedUser, TrackedUserFilters, TrackedUsersResponse } from '#shared/types'

export const useTrackedUsers = () => {
  const api = useAPI()

  const list = (filters: TrackedUserFilters = {}): Promise<TrackedUsersResponse> => {
    return api.get<TrackedUsersResponse>('/admin/tracked-users' + buildQuery(filters))
  }

  const find = (id: number): Promise<TrackedUser> => {
    return api.get<TrackedUser>(`/admin/tracked-users/${id}`)
  }

  return { list, find }
}

const buildQuery = (params: Record<string, string | number | undefined>): string => {
  const entries = Object.entries(params).filter(
    ([, v]) => v !== undefined && v !== '' && v !== null
  )
  if (entries.length === 0) return ''
  return '?' + entries
    .map(([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(String(v))}`)
    .join('&')
}
