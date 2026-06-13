export interface TrackedUser {
  id: number
  external_id: string
  id_wpp: string
  pixel_name: string
  fbp: string
  fbc: string
  gclid: string
  client_ip: string
  client_user_agent: string
  country: string
  state: string
  city: string
  postal_code: string
  first_name: string
  last_name: string
  email: string
  phone: string
  url: string
  first_utm_source: string
  first_utm_medium: string
  first_utm_campaign: string
  first_utm_content: string
  first_utm_term: string
  last_utm_source: string
  last_utm_medium: string
  last_utm_campaign: string
  last_utm_content: string
  last_utm_term: string
  created_at: string
  updated_at: string
}

export interface TrackedUserFilters {
  pixel?: string
  search?: string
  limit?: number
  offset?: number
}

export interface TrackedUsersResponse {
  data: TrackedUser[]
  total: number
  limit: number
  offset: number
}
