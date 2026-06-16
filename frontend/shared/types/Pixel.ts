export interface PageRule {
  page: string
  event: string
  value?: number
  currency?: string
}

export interface FormMappings {
  email: string[]
  phone: string[]
  first_name: string[]
  last_name: string[]
}

export interface TrackerConfig {
  // Produto / checkout
  product_value: number[]
  product_currency: string
  product_checkout: string

  // Auto-triggers
  auto_page_view: boolean
  auto_checkout_buttons: boolean
  auto_form_scraping: boolean
  auto_form_lead: boolean
  whatsapp_protocol: boolean
  whatsapp_lead: boolean
  send_scrolls: boolean
  send_timer_1min: boolean
  send_view_video: boolean

  // Page rules
  page_rules: PageRule[]

  // Form mappings
  form_mappings: FormMappings

  // Meta
  meta_init_pixel_id: string

  // UTM
  utm_origin_param: string
}

export interface Pixel {
  id: number
  name: string
  label: string
  pixel_ids: string[]
  access_tokens: string[]
  test_codes: string[]
  domains: string[]
  tracker_config: TrackerConfig
  enabled: boolean
  tracking_host: string
  account_id: number
  pixel_count: number

  // TikTok Events API (opcional)
  tiktok_pixel_ids: string[]
  tiktok_access_tokens: string[]
  tiktok_test_codes: string[]
  tiktok_pixel_count: number

  created_at: string
  updated_at: string
}

export interface PixelInput {
  name: string
  label?: string
  pixel_ids: string[]
  access_tokens?: string[]
  test_codes?: string[]
  domains?: string[]
  tracker_config?: TrackerConfig
  enabled?: boolean
  tracking_host?: string
  // Multi-tenant: ownership da empresa que recebe o pixel.
  // Owner pode definir; member é ignorado (backend força a própria conta).
  account_id?: number

  // TikTok (opcional)
  tiktok_pixel_ids?: string[]
  tiktok_access_tokens?: string[]
  tiktok_test_codes?: string[]
}

export type CreatePixelInput = PixelInput
export type UpdatePixelInput = Partial<PixelInput>

export interface PixelsListResponse {
  data: Pixel[]
}

export interface PixelDefaults {
  tracker_config: TrackerConfig
}

export interface DNSVerifyResponse {
  status: 'ok' | 'wrong_target' | 'not_found' | 'no_host'
  host?: string
  cname?: string
  a?: string[]
  target?: string
  hint: string
}


export interface TestEventInput {
  event_name?: string
  email?: string
  phone?: string
  first_name?: string
  last_name?: string
  value?: number
  currency?: string
  // 'meta' (default), 'tiktok' ou 'all'
  channel?: 'meta' | 'tiktok' | 'all'
}

export interface TestEventPixelResult {
  channel?: 'meta' | 'tiktok'
  pixel_id: string
  status_code: number
  response?: string
  error?: string
}

export interface TestEventResponse {
  event_id: string
  event_name: string
  channel?: 'meta' | 'tiktok' | 'all'
  results: TestEventPixelResult[]
}
