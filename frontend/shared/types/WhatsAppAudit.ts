import type { CrmLead, CrmStage, CrmTag } from './Crm'

export type ConversationOriginCategory =
  | 'all'
  | 'meta_ads'
  | 'google_ads'
  | 'tiktok_ads'
  | 'bio_link'
  | 'site'

export type ConversationSLAStatus =
  | 'unanswered'
  | 'waiting_reply'
  | 'replied'

export interface ConversationAuditItem {
  chat_id: number
  instance_id: number
  instance_name: string
  pixel_id: number
  pixel_name: string
  currency: string
  tracked_user_id?: number | null
  crm_lead_id?: number | null
  lead_name: string
  phone: string
  email?: string
  profile_pic?: string | null
  origin_category: Exclude<ConversationOriginCategory, 'all'> | string
  origin_type: string
  campaign_name?: string
  ad_name?: string
  utm_source?: string
  trackable_link_name?: string
  stage_id?: number | null
  stage_name?: string
  stage_color?: string
  lead_value: number
  is_won: boolean
  tags: CrmTag[]
  ai_qualified: boolean
  sla_status: ConversationSLAStatus | string
  total_messages: number
  inbound_messages: number
  outbound_messages: number
  first_response_sec: number | null
  avg_response_sec: number | null
  first_inbound_at?: string | null
  last_inbound_at?: string | null
  last_outbound_at?: string | null
  last_message_time: string
  created_at: string
}

export interface ConversationAuditKPIs {
  total_conversations: number
  waiting_reply_count: number
  unanswered_count: number
  avg_first_response_sec: number | null
  won_count: number
  won_revenue: number
}

export interface ConversationOriginCounts {
  all: number
  meta_ads: number
  google_ads: number
  tiktok_ads: number
  bio_link: number
  site: number
}

export interface ConversationAuditResponse {
  items: ConversationAuditItem[]
  total: number
  kpis: ConversationAuditKPIs
  origin_counts: ConversationOriginCounts
  campaigns: string[]
  utms: string[]
}

export interface ConversationAuditQuery {
  from?: string
  to?: string
  pixel?: string
  instance_id?: number | string
  search?: string
  sla_status?: string
  stage_id?: number | string
  tag_id?: number | string
  origin?: string
  campaign?: string
  utm?: string
  sort_by?: string
  sort_dir?: 'asc' | 'desc'
  limit?: number
  offset?: number
}

export interface PublicCrmBoardResponse {
  stages: CrmStage[]
  leads: CrmLead[]
  tags: CrmTag[]
  allow_edit: boolean
}
