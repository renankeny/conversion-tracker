import type { AdminUser } from './Auth'

export interface AdminUsersListResponse {
  data: AdminUser[]
}

export interface CreateAdminInput {
  email: string
  password: string
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
