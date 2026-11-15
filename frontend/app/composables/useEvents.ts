import type {
  EventDetail,
  EventFilters,
  EventKeyRef,
  EventsResponse,
  GroupedEventsResponse,
} from '#shared/types'

export const useEvents = () => {
  const api = useAPI()

  /** Listagem flat (uma linha por canal). Mantido para compatibilidade. */
  const list = (filters: EventFilters = {}): Promise<EventsResponse> => {
    return api.get<EventsResponse>('/admin/sent-events' + buildQuery(filters))
  }

  /** Listagem agrupada (uma linha por evento com array de canais). */
  const listGrouped = (filters: EventFilters = {}): Promise<GroupedEventsResponse> => {
    return api.get<GroupedEventsResponse>('/admin/sent-events/grouped' + buildQuery(filters))
  }

  /** Detalhes completos de um evento (todos os canais com payload e resposta). */
  const findByEventID = (eventId: string, pixelName: string): Promise<EventDetail> => {
    return api.get<EventDetail>(
      '/admin/sent-events/by-event-id' + buildQuery({ event_id: eventId, pixel: pixelName }),
    )
  }

  /** Remove múltiplos eventos por referência composta (máx 500). */
  const deleteEvents = (items: EventKeyRef[]): Promise<{ deleted: number }> => {
    return api.delete<{ deleted: number }>('/admin/sent-events', {
      body: {
        items,
        event_ids: items.map((i) => i.event_id),
      },
    })
  }

  /** Reenfileira eventos falhos por referência composta (máx 100). */
  const retryEvents = (
    items: EventKeyRef[],
  ): Promise<{ queued: number; not_found: number; total: number }> => {
    return api.post<{ queued: number; not_found: number; total: number }>(
      '/admin/sent-events/retry',
      {
        items,
        event_ids: items.map((i) => i.event_id),
      },
    )
  }

  /** Alias legado para remoção por lista de event_ids. */
  const deleteByEventIDs = (eventIds: string[]): Promise<{ deleted: number }> => {
    return deleteEvents(eventIds.map((event_id) => ({ event_id })))
  }

  /** Alias legado para reenfileiramento por lista de event_ids. */
  const retryByEventIDs = (
    eventIds: string[],
  ): Promise<{ queued: number; not_found: number; total: number }> => {
    return retryEvents(eventIds.map((event_id) => ({ event_id })))
  }

  return {
    list,
    listGrouped,
    findByEventID,
    deleteEvents,
    retryEvents,
    deleteByEventIDs,
    retryByEventIDs,
  }
}

const buildQuery = (params: Record<string, unknown>): string => {
  const entries = Object.entries(params).filter(
    ([, v]) => v !== undefined && v !== '' && v !== null,
  )
  if (entries.length === 0) return ''
  return (
    '?' +
    entries
      .map(([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(String(v))}`)
      .join('&')
  )
}
