export interface SentEvent {
  id: number
  pixel_name: string
  event_name: string
  external_id: string
  id_wpp: string
  status: EventStatus
  error: string
  response_body: string
  created_at: string
  sent_at: string | null
}

export type EventStatus = 'sent' | 'pending' | 'failed'

export interface EventFilters {
  pixel?: string
  event?: string
  status?: EventStatus
  from?: string
  to?: string
  limit?: number
  offset?: number
}

export interface EventsResponse {
  data: SentEvent[]
  total: number
  limit: number
  offset: number
}
