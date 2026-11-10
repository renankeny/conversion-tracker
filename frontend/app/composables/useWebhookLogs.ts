import type { WebhookLog, WebhookLogsResponse } from '#shared/types'

export const useWebhookLogs = () => {
  const api = useAPI()

  const list = (filters: {
    source?: string
    status?: string
    from?: string
    to?: string
    limit?: number
    offset?: number
  } = {}): Promise<WebhookLogsResponse> => {
    const params = new URLSearchParams()
    if (filters.source)  params.set('source',  filters.source)
    if (filters.status)  params.set('status',  filters.status)
    if (filters.from)    params.set('from',    filters.from)
    if (filters.to)      params.set('to',      filters.to)
    if (filters.limit)   params.set('limit',   String(filters.limit))
    if (filters.offset)  params.set('offset',  String(filters.offset))
    const qs = params.toString() ? '?' + params.toString() : ''
    return api.get<WebhookLogsResponse>('/admin/webhook-logs' + qs)
  }

  const replay = (id: number): Promise<Record<string, unknown>> => {
    return api.post<Record<string, unknown>>(`/admin/webhook-logs/${id}/replay`)
  }

  return { list, replay }
}
