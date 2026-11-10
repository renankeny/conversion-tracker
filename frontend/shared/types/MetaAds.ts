export interface MetaAccount {
  id: number
  account_id: number
  name: string
  connected: boolean
  oauth_connected: boolean
  oauth_user_id: string
  oauth_name: string
  oauth_email: string
  oauth_picture_url?: string
  picture_url?: string
  token_expires_at?: string | null
  ad_account_id: string
  ad_account_name: string
  ad_account_picture_url?: string
  default_page_id: string
  default_page_name: string
  default_page_picture_url?: string
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface MetaAccountInput {
  name: string
  ad_account_id?: string
  ad_account_name?: string
  ad_account_picture_url?: string
  default_page_id?: string
  default_page_name?: string
  default_page_picture_url?: string
  reuse_oauth_from_id?: number
  enabled?: boolean
}

export interface MetaOAuthStartResponse {
  url: string
  state: string
}

export interface MetaRemoteAdAccount {
  id: string
  account_id: string
  name: string
  account_status?: number
  currency?: string
  business_name?: string
  picture_url?: string
}

export interface MetaRemotePage {
  id: string
  name: string
  category?: string
  picture_url?: string
}

export interface MetaRemotePixel {
  id: string
  name: string
}
