// Account é uma conta/empresa cliente. Cada empresa isola seus pixels,
// tracked_users, sent_events, CRM e configurações.
export interface Account {
  id: number
  name: string
  slug: string
  enabled: boolean
  owner_id?: number | null
  pixels_count?: number
  users_count?: number
  sent_events_retention_days?: number | null
  event_counters_retention_days?: number | null
  // Nomes customizados dos eventos na dashboard (cards/funil). Chave é o
  // event_name original (ex: "InitiateCheckout").
  event_label_overrides?: Record<string, string>
  created_at: string
  updated_at: string
}

export interface AccountsListResponse {
  data: Account[]
}

export interface CreateAccountInput {
  name: string
  slug?: string
  enabled?: boolean
  sent_events_retention_days?: number | null
  event_counters_retention_days?: number | null
}

export interface UpdateAccountInput {
  name?: string
  slug?: string
  enabled?: boolean
  sent_events_retention_days?: number | null
  event_counters_retention_days?: number | null
  update_retention?: boolean
}
