export type ConfigurableNotificationCategory =
  | 'whatsapp_connection'
  | 'lead_created'
  | 'ai_lead_qualified'
  | 'sale_approved'
  | 'sale_refunded'
  | 'meta_account_lifecycle'
  | 'google_ads_account_lifecycle'
  | 'pixel_lifecycle'
  | 'event_dispatch_error'

export type NotificationCategory =
  | ConfigurableNotificationCategory
  | 'company_lifecycle'
  | 'api_key_lifecycle'

export type AccountScopeMode = 'all' | 'custom' | 'none'

export interface NotificationPreferences {
  user_id: number
  account_scope_mode: AccountScopeMode
  enabled_account_ids: number[]
  whatsapp_connection: boolean
  lead_created: boolean
  ai_lead_qualified: boolean
  sale_approved: boolean
  sale_refunded: boolean
  meta_account_lifecycle: boolean
  google_ads_account_lifecycle: boolean
  pixel_lifecycle: boolean
  event_dispatch_error: boolean
  updated_at?: string
}

export interface PushSubscriptionItem {
  id: number
  user_id: number
  endpoint: string
  device_label: string
  user_agent?: string
  created_at: string
  last_seen_at: string
}

export interface UserNotificationItem {
  id: number
  user_id: number
  account_id?: number | null
  account_name?: string
  category: NotificationCategory
  title: string
  body: string
  action_url: string
  is_read: boolean
  created_at: string
}

export interface NotificationsOverview {
  vapid_public_key: string
  unread_count: number
  preferences: NotificationPreferences
  subscriptions: PushSubscriptionItem[]
  notifications: UserNotificationItem[]
}
