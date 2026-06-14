import type { AdminUser } from './Auth'

export interface AdminUsersListResponse {
  data: AdminUser[]
}

export interface CreateAdminInput {
  email: string
  password: string
  name?: string
  phone?: string
  // Multi-tenant: 'owner' enxerga tudo, 'member' apenas a própria account_id.
  role?: 'owner' | 'member'
  account_id?: number | null
}

export interface UpdateProfileInput {
  name: string
  phone: string
}

export interface ChangePasswordInput {
  current_password: string
  new_password: string
}

export interface GeoIPStatus {
  path: string
  license_key_set: boolean
  loaded: boolean
  exists?: boolean
  size_bytes?: number
  modified_at?: string
}

export interface GeoIPRefreshResponse {
  path: string
  installed: boolean
  size_bytes?: number
  modified_at?: string
  reloaded?: boolean
  reload_error?: string
}

export interface MaintenanceStatusResponse {
  sent_events_retention_days: number
  event_counters_retention_days: number
  now: string
  // Multi-tenant: presentes quando o caller está em escopo de conta.
  account_id?: number
  account_sent_events_retention_days?: number | null
  account_event_counters_retention_days?: number | null
  effective_sent_events_retention_days?: number
  effective_event_counters_retention_days?: number
}

export interface UpdateRetentionInput {
  // null = remove override e volta pro env global
  sent_events_retention_days?: number | null
  event_counters_retention_days?: number | null
  account_id?: number
}

export interface ResetAccountResponse {
  account_id: number
  deleted_sent_events: number
  deleted_event_counters: number
}

export interface PruneResultRow {
  table: string
  retention_days: number
  threshold: string
  deleted: number
  duration_ms?: number
}

export interface PruneResponse {
  results: PruneResultRow[]
}
