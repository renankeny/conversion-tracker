export type HealthAlertSeverity = 'warning' | 'danger' | 'info'

export interface HealthAlert {
  id: string
  type: string
  severity: HealthAlertSeverity
  title: string
  message: string
  action_url?: string
  action_text?: string
  created_at: string
}

export interface HealthAlertsResponse {
  alerts: HealthAlert[]
  count: number
}

