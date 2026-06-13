export interface DashboardStats {
  status_totals: {
    sent: number
    pending: number
    failed: number
  }
  event_totals: Record<string, number>
  per_day: Array<{
    date: string
    count: number
  }>
  total_events: number
  tracked_users: number
}

export interface StatsFilters {
  from?: string
  to?: string
  pixel?: string
}
