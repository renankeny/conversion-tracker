export interface WebhookLog {
  id: number
  account_id: number
  source: string        // 'hotmart' | 'cakto' | 'kirvano' | 'kiwify' | 'pagtrust' | 'yampi'
  pixel_name: string
  external_id: string
  event_name: string
  event_id: string
  status: 'queued' | 'failed' | 'replayed'
  error?: string
  received_at: string
}

export interface WebhookLogsResponse {
  data: WebhookLog[]
  total: number
  limit: number
  offset: number
}
