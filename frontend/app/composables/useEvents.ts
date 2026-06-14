import type { EventFilters, EventsResponse, SentEvent } from '#shared/types'

export const useEvents = () => {
  const api = useAPI()

  const list = (filters: EventFilters = {}): Promise<EventsResponse> => {
    return api.get<EventsResponse>('/admin/sent-events' + buildQuery(filters))
  }

  const find = (id: number): Promise<SentEvent> => {
    return api.get<SentEvent>(`/admin/sent-events/${id}`)
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
