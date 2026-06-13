export interface CounterSummaryRow {
  pixel_name?: string
  event_name: string
  count_total: number
  count_sent: number
  count_failed: number
}

export interface CounterSummaryResponse {
  from?: string
  to?: string
  pixel?: string
  event?: string
  events: CounterSummaryRow[]
}

export interface CounterSeriesPoint {
  bucket_at: string
  event_name: string
  count_total: number
  count_sent: number
  count_failed: number
}

export interface CounterSeriesResponse {
  from?: string
  to?: string
  pixel?: string
  event?: string
  granularity: 'hour' | 'day'
  points: CounterSeriesPoint[]
}

export interface CounterFilters {
  from?: string
  to?: string
  pixel?: string
  event?: string
  granularity?: 'hour' | 'day'
}
