/**
 * Tipos do dominio Google Ads no painel.
 */

export type GoogleAdsEventType =
  | 'Purchase'
  | 'Lead'
  | 'CompleteRegistration'
  | 'InitiateCheckout'
  | 'AddToCart'
  | 'ViewContent'
  | 'PageView'

export interface GoogleAdsConversionAction {
  event_type: GoogleAdsEventType | string
  conversion_action_id: string
  name: string
  enabled: boolean
}

export interface GoogleAdsAccount {
  id: number
  account_id: number
  name: string
  customer_id: string
  login_customer_id: string
  conversion_actions: GoogleAdsConversionAction[]
  oauth_email: string
  oauth_picture_url?: string
  picture_url?: string
  oauth_connected?: boolean
  enabled: boolean
  connected: boolean
  created_at: string
  updated_at: string
}

export interface GoogleAdsAccountInput {
  name: string
  customer_id?: string
  login_customer_id?: string
  conversion_actions?: GoogleAdsConversionAction[]
  reuse_oauth_from_id?: number
  enabled?: boolean
}

export interface GoogleAdsOAuthStartResponse {
  url: string
  state: string
}

export interface GoogleAdsRemoteCustomer {
  id: string
  formatted_id: string
  name: string
  is_manager: boolean
  manager_customer_id?: string
  currency_code?: string
  status?: string
}

export interface GoogleAdsRemoteConversionAction {
  id: string
  name: string
  category: string
  type: string
  status: string
  resource_name: string
}
