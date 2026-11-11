export type WhatsAppOriginType =
  | 'meta_ads_ctwa'
  | 'meta_ads_lp'
  | 'google_ads'
  | 'tiktok_ads'
  | 'bio_link_organic'
  | 'website_organic'
  | 'unknown'

export type WhatsAppMediaStatus =
  | 'none'
  | 'pending'
  | 'ready'
  | 'expired'
  | 'failed'
  | 'skipped_size'

export interface WhatsAppServer {
  id: number
  account_id: number
  name: string
  url: string
  has_api_key: boolean
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface WhatsAppServerHealth {
  healthy: boolean
  version: string
  database: string
  instance_count: number
}

export type WhatsAppProvider = 'evolution_go' | 'api_premium' | 'uazapi'

export interface WhatsAppInstance {
  id: number
  account_id: number
  pixel_id: number
  pixel_name?: string
  server_id: number
  provider: WhatsAppProvider
  external_api_url: string
  has_instance_token: boolean
  webhook_url?: string
  name: string
  instance_name: string
  evolution_uuid: string
  owner_jid: string
  profile_name: string
  profile_pic_url: string
  share_token?: string | null
  connection_status: 'CREATED' | 'CONNECTING' | 'OPEN' | 'CLOSED' | 'DISCONNECTED' | string
  purchase_trigger: string
  lead_trigger: string
  custom_event_name: string
  custom_event_trigger: string
  auto_create_crm_lead: boolean
  default_crm_stage_id?: number | null
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface WhatsAppInstanceInput {
  name: string
  pixel_id: number
  provider?: WhatsAppProvider
  external_api_url?: string
  instance_token?: string
  purchase_trigger?: string
  lead_trigger?: string
  custom_event_name?: string
  custom_event_trigger?: string
  auto_create_crm_lead?: boolean
  default_crm_stage_id?: number | null
  enabled?: boolean
}

export interface WhatsAppChat {
  id: number
  account_id: number
  instance_id: number
  instance_name?: string
  remote_jid: string
  tracked_user_id?: number | null
  crm_lead_id?: number | null
  phone: string
  lid_jid: string
  push_name: string
  contact_name?: string | null
  profile_pic?: string | null
  origin_type: WhatsAppOriginType | string
  ctwa_clid: string
  source_id: string
  page_id: string
  source_app: string
  ad_name: string
  adset_id: string
  adset_name: string
  campaign_id: string
  campaign_name: string
  ad_thumbnail_url: string
  unread_count: number
  last_message_time: string
  created_at: string
  updated_at: string
}

export interface WhatsAppMessage {
  id: number
  chat_id: number
  whatsapp_message_id: string
  is_from_me: boolean
  message_type: 'text' | 'audio' | 'image' | 'video' | 'document' | 'sticker' | string
  content: string
  has_media: boolean
  media_mime_type?: string | null
  media_status: WhatsAppMediaStatus
  media_size_bytes: number
  media_duration_sec: number
  media_filename: string
  media_expires_at?: string | null
  media_stream_url?: string
  timestamp: string
  created_at: string
}

export interface WhatsAppChatWithMessages {
  chat: WhatsAppChat | null
  messages: WhatsAppMessage[]
}

