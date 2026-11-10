import type {
  EventDetail,
  EventFilters,
  EventsResponse,
  GroupedEventsResponse,
  SentEvent,
} from '#shared/types'

export const useEvents = () => {
  const api = useAPI()

  /** Listagem flat (uma linha por canal). Mantido pra compat. */
  const list = (filters: EventFilters = {}): Promise<EventsResponse> => {
    return api.get<EventsResponse>('/admin/sent-events' + buildQuery(filters))
  }

  /** Listagem agrupada — uma linha por evento com array de canais. */
  const listGrouped = (filters: EventFilters = {}): Promise<GroupedEventsResponse> => {
    return api.get<GroupedEventsResponse>('/admin/sent-events/grouped' + buildQuery(filters))
  }

  /** Detalhes completos de um evento (todos canais com payload + response). */
  const findByEventID = (eventId: string, pixelName: string): Promise<EventDetail> => {
    const qs = buildQuery({ pixel: pixelName })
    return api.get<EventDetail>(`/admin/sent-events/by-event-id/${encodeURIComponent(eventId)}${qs}`)
  }

  const find = (id: number): Promise<SentEvent> => {
    return api.get<SentEvent>(`/admin/sent-events/${id}`)
  }

  /** Remove múltiplos eventos por event_id (máx 500). */
  const deleteByEventIDs = (eventIds: string[]): Promise<{ deleted: number }> => {
    return api.delete<{ deleted: number }>('/admin/sent-events', {
      body: { event_ids: eventIds },
    })
  }

  /** Reenfileira eventos falhos (máx 100). */
  const retryByEventIDs = (eventIds: string[]): Promise<{ queued: number; not_found: number; total: number }> => {
    return api.post<{ queued: number; not_found: number; total: number }>(
      '/admin/sent-events/retry',
      { event_ids: eventIds },
    )
  }

  return { list, listGrouped, findByEventID, find, deleteByEventIDs, retryByEventIDs }
}

const buildQuery = (params: Record<string, string | number | boolean | undefined>): string => {
  const entries = Object.entries(params).filter(
    ([, v]) => v !== undefined && v !== '' && v !== null
  )
  if (entries.length === 0) return ''
  return '?' + entries
    .map(([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(String(v))}`)
    .join('&')
}
