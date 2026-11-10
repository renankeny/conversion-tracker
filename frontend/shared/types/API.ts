export interface APIError {
  error: string
  code?: string
}

export interface APIResponse<T> {
  data?: T
  error?: string
  code?: string
}

export interface PaginatedResponse<T> {
  data: T[]
  total: number
  limit: number
  offset: number
}
