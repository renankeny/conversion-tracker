/** Canais pelos quais um evento pode ter sido enviado. */
export type Channel = 'meta' | 'tiktok' | 'google_ads' | 'ga4'

export interface SentEvent {
  id: number
  event_id: string
  pixel_name: string
  event_name: string
  external_id: string
  channel: Channel
  status: EventStatus
  response_code: number
  error: string
  response_body: string
  /** Payload JSON enviado para a plataforma (com PII hasheada, tokens redatados). */
  payload?: string
  purchase_value: number
  created_at: string
  updated_at: string
}

export type EventStatus = 'sent' | 'pending' | 'failed' | 'queued'

export interface EventFilters {
  pixel?: string
  event?: string
  external_id?: string
  status?: EventStatus
  traffic_source?: string
  from?: string
  to?: string
  limit?: number
  offset?: number
  include_ltv?: boolean
  include_user_context?: boolean
}

export interface EventsResponse {
  data: SentEvent[]
  total: number
  limit: number
  offset: number
}

/** Resumo por canal mostrado na linha agrupada. */
export interface ChannelBrief {
  id: number
  channel: Channel
  status: EventStatus
  response_code: number
  has_error: boolean
}

export interface AggregatedItem {
  order_id?: string
  product_id?: string
  product_title?: string
  item_type?: 'main' | 'order_bump' | 'upsell' | string
  value: number
  gateway?: string
  created_at: number
}

/** Uma linha por evento (event_id + pixel_name) com array de canais. */
export interface GroupedEvent {
  event_id: string
  event_name: string
  pixel_name: string
  external_id: string
  purchase_value: number  // valor monetário (0 se não aplicável)
  created_at: string
  channels: ChannelBrief[]
  /** LTV do cliente (todas as compras, todo o histórico). Só vem preenchido quando a busca pede include_ltv. */
  ltv?: number
  /** Origem (utm_source) do primeiro e do último acesso do cliente. Só vem preenchido com include_user_context. */
  first_utm_source?: string
  last_utm_source?: string

  /** Agrupamento de Conversões (Order Bump & Upsell) */
  aggregated_count?: number
  aggregated_items?: AggregatedItem[]
}

export interface GroupedEventsResponse {
  data: GroupedEvent[]
  total: number
  limit: number
  offset: number
}

/** Detalhe de um evento (todos os canais com payload + response). */
export interface EventDetail {
  event_id: string
  event_name: string
  pixel_name: string
  external_id: string
  created_at: string
  channels: SentEvent[]
}
