export type WebhookLogStatus = 'queued' | 'buffered' | 'failed' | 'replayed'

export interface WebhookLog {
  id: number
  account_id: number
  source: string
  pixel_name: string
  external_id: string
  event_name: string
  event_id: string
  status: WebhookLogStatus
  error?: string
  can_replay?: boolean
  received_at: string
}

export interface WebhookLogsResponse {
  data: WebhookLog[]
  total: number
  limit: number
  offset: number
}

export interface PlatformSummaryItem {
  has_token: boolean
  masked_token?: string
  token_updated_at?: string
  total_accepted: number
  total_failed: number
  last_received_at?: string
}

export interface PlatformsSummaryResponse {
  account_id: number
  items: Record<string, PlatformSummaryItem>
}
