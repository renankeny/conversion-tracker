export interface TrackedUser {
  id: number
  account_id?: number
  external_id: string
  id_wpp: string
  pixel_name: string
  fbp: string
  fbc: string
  gclid: string
  wbraid: string
  gbraid: string
  ttclid: string
  ttp: string
  ctwa_clid?: string
  wa_chat_id?: number | null
  wa_name?: string
  wa_ad_id?: string
  wa_source_app?: string
  wa_lid?: string
  ga4_client_id: string
  ga4_session_id: string
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
  first_url?: string
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
  pageview_count?: number
  checkout_count?: number
  last_seen_at?: string
  ltv?: number
  time_to_purchase?: string
  referrer?: string
  traffic_type?: 'human' | 'bot' | 'inconclusive'
  traffic_score?: number
  traffic_category?: string
  traffic_rules?: TrafficRule[]
}

export interface TrafficRule {
  id: string
  label: string
  impact: 'human' | 'automated' | 'neutral'
  weight: number
  description: string
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

// TrackedUserUpdateInput é o payload de PUT /admin/tracked-users/{id}.
// Todos os campos são opcionais — só atualiza o que vier preenchido.
// Pra limpar um campo, envie string vazia explícita (não null).
//
// Campos protegidos não estão aqui: id, external_id, account_id,
// created_at, updated_at — são apenas lidos.
export interface TrackedUserUpdateInput {
  id_wpp?: string
  pixel_name?: string
  fbp?: string
  fbc?: string
  ctwa_clid?: string
  gclid?: string
  wbraid?: string
  gbraid?: string
  ttclid?: string
  ttp?: string
  ga4_client_id?: string
  ga4_session_id?: string
  client_ip?: string
  client_user_agent?: string
  country?: string
  state?: string
  city?: string
  postal_code?: string
  first_name?: string
  last_name?: string
  email?: string
  phone?: string
  url?: string
  first_url?: string
  first_utm_source?: string
  first_utm_medium?: string
  first_utm_campaign?: string
  first_utm_content?: string
  first_utm_term?: string
  last_utm_source?: string
  last_utm_medium?: string
  last_utm_campaign?: string
  last_utm_content?: string
  last_utm_term?: string
}

export interface CustomerJourneySession {
  external_id: string
  created_at: string
  device_type: string // "Celular", "Computador", "Tablet"
  browser: string
  url?: string
  first_utm_source?: string
  first_utm_campaign?: string
  pageview_count: number
  checkout_count: number
  is_initial: boolean
}

export interface CustomerJourneyResponse {
  primary_user: TrackedUser
  first_touch: TrackedUser
  sessions_count: number
  sessions: CustomerJourneySession[]
  total_pageviews: number
  total_checkouts: number
  time_to_purchase: string
  events: import('./Event').GroupedEvent[]
}

