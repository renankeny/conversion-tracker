// Account é uma conta cliente do TrackFind. Cada conta isola seus pixels,
// tracked_users, sent_events e configurações.
export interface Account {
  id: number
  name: string
  slug: string
  enabled: boolean
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
}

export interface UpdateAccountInput {
  name?: string
  slug?: string
  enabled?: boolean
}
