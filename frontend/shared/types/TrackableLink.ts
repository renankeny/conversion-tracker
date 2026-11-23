export type LinkRoutingMode = 'auto_pixel' | 'specific_instance' | 'manual_phone'
export type LinkEventTriggerMode = 'on_first_message' | 'on_click'

export interface TrackableLink {
  id: number
  account_id: number
  pixel_id: number
  pixel_name?: string
  pixel_label?: string
  tracking_host?: string
  slug: string
  name: string
  public_url?: string
  routing_mode: LinkRoutingMode
  whatsapp_instance_id: number | null
  instance_name?: string
  instance_provider?: string
  instance_status?: string
  fallback_phone: string
  prefill_message: string
  utm_source: string
  utm_medium: string
  utm_campaign: string
  utm_content: string
  utm_term: string
  send_events_enabled: boolean
  event_trigger_mode: LinkEventTriggerMode
  event_name: string
  send_meta: boolean
  send_tiktok: boolean
  send_google_ads: boolean
  send_ga4: boolean
  enabled: boolean
  clicks_count: number
  chats_count: number
  sales_count: number
  sales_revenue: number
  created_at: string
  updated_at: string
}

export interface TrackableLinkInput {
  name: string
  routing_mode: LinkRoutingMode
  whatsapp_instance_id: number | null
  fallback_phone: string
  prefill_message: string
  utm_source: string
  utm_medium: string
  utm_campaign: string
  utm_content: string
  utm_term: string
  send_events_enabled: boolean
  event_trigger_mode: LinkEventTriggerMode
  event_name: string
  send_meta: boolean
  send_tiktok: boolean
  send_google_ads: boolean
  send_ga4: boolean
  enabled: boolean
}

export interface TrackableLinksOverview {
  total_links: number
  total_clicks: number
  total_chats: number
  total_sales: number
  total_revenue: number
  links: TrackableLink[]
}
