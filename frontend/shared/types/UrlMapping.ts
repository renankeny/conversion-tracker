export interface UrlMapping {
  id: number
  source: string
  target: string
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface UrlMappingInput {
  source: string
  target: string
  enabled?: boolean
}

export interface UrlMappingsListResponse {
  data: UrlMapping[]
}
