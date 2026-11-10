export interface AdminUser {
  id: number
  email: string
  name?: string
  phone?: string
  // Multi-tenant: 'owner' tem acesso global, 'member' apenas à própria account_id.
  role?: string
  account_id?: number | null
  parent_id?: number | null
  permissions?: Record<string, any>
  preferences?: Record<string, any>
  created_at?: string
  updated_at?: string
}

export interface LoginRequest {
  email: string
  password: string
  remember_me?: boolean
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
