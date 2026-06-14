export interface AdminUser {
  id: number
  email: string
  name?: string
  // Multi-tenant: 'owner' tem acesso global, 'member' apenas à própria account_id.
  role?: 'owner' | 'member'
  account_id?: number | null
  created_at?: string
  updated_at?: string
}

export interface LoginRequest {
  email: string
  password: string
}

export interface LoginResponse {
  access_token: string
  user: AdminUser
}

export interface AuthState {
  user: AdminUser | null
  token: string | null
  isAuthenticated: boolean
}
