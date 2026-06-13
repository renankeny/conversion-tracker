export interface AdminUser {
  id: number
  email: string
  name?: string
  created_at?: string
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
