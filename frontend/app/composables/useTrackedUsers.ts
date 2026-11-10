import type { CustomerJourneyResponse, TrackedUser, TrackedUserFilters, TrackedUserUpdateInput, TrackedUsersResponse } from '#shared/types'

export const useTrackedUsers = () => {
  const api = useAPI()

  const list = (filters: TrackedUserFilters = {}): Promise<TrackedUsersResponse> => {
    return api.get<TrackedUsersResponse>('/admin/tracked-users' + buildQuery(filters as Record<string, string | number | undefined>))
  }

  const find = (id: number): Promise<TrackedUser> => {
    return api.get<TrackedUser>(`/admin/tracked-users/${id}`)
  }

  // Busca exata por id, external_id, email ou phone (passe só um).
  const lookup = (params: { id?: number; external_id?: string; email?: string; phone?: string }): Promise<TrackedUser> => {
    return api.get<TrackedUser>('/admin/tracked-users/lookup' + buildQuery(params as Record<string, string | number | undefined>))
  }

  // update aceita patch parcial — só os campos enviados são alterados.
  // Pra limpar um campo, envie string vazia explícita.
  const update = (id: number, patch: TrackedUserUpdateInput): Promise<TrackedUser> => {
    return api.put<TrackedUser>(`/admin/tracked-users/${id}`, patch)
  }

  // getJourney busca a jornada unificada (cross-device/cross-browser) com todas as sessões e eventos vinculados.
  const getJourney = (externalId: string): Promise<CustomerJourneyResponse> => {
    return api.get<CustomerJourneyResponse>('/admin/tracked-users/journey' + buildQuery({ external_id: externalId }))
  }

  return { list, find, lookup, update, getJourney }
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
