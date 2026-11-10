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
  /** Evento disparado no clique do WhatsApp quando whatsapp_lead=true: "Lead" ou "Contact". */
  whatsapp_lead_event: string
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

export interface DashboardConfig {
  enabled_cards: string[]
  card_order: string[]
  enabled_funnel_events: string[]
  funnel_order: string[]
  event_labels: Record<string, string>
}

export interface Pixel {
  id: number
  name: string
  label: string
  /** Token opaco de 32 chars hex — usar no snippet HTML em vez do name */
  public_token: string
  pixel_ids: string[]
  access_tokens: string[]
  test_codes: string[]
  domains: string[]
  tracker_config: TrackerConfig
  enabled: boolean
  account_id: number
  pixel_count: number

  /** Config da dashboard web (cards/ordem/funil/nomes) — por pixel. */
  dashboard_config?: DashboardConfig
  /** Token do link público view-only. Vazio = link não gerado. */
  dashboard_share_token?: string

  // TikTok Events API (opcional)
  tiktok_pixel_ids: string[]
  tiktok_access_tokens: string[]
  tiktok_test_codes: string[]
  tiktok_pixel_count: number

  /**
   * Vinculo opcional a uma conta Google Ads (entidade propria, gerenciada
   * em /integracoes/google-ads). Quando definido, eventos elegiveis
   * desse pixel sao tambem despachados pra essa conta como conversoes
   * offline.
   */
  google_ads_account_id?: number | null
  meta_account_id?: number | null
  meta_ad_account_id?: string

  ga4_measurement_id?: string
  ga4_api_secret?: string

  /** Agrupamento de Conversões (Order Bump & Upsell) */
  aggregation_enabled?: boolean
  aggregation_window_minutes?: number

  created_at: string
  updated_at: string
}

export interface PixelInput {
  name: string
  label?: string
  pixel_ids?: string[]
  access_tokens?: string[]
  test_codes?: string[]
  domains?: string[]
  tracker_config?: TrackerConfig
  enabled?: boolean
  // Multi-tenant: ownership da empresa que recebe o pixel.
  // Owner pode definir; member é ignorado (backend força a própria conta).
  account_id?: number

  // TikTok (opcional)
  tiktok_pixel_ids?: string[]
  tiktok_access_tokens?: string[]
  tiktok_test_codes?: string[]

  /** Vinculo a uma conta Google Ads. 0 ou null desfaz o vinculo. */
  google_ads_account_id?: number | null
  /** Vinculo opcional a uma conta Meta Ads OAuth. 0 ou null desfaz o vinculo. */
  meta_account_id?: number | null
  meta_ad_account_id?: string

  ga4_measurement_id?: string
  ga4_api_secret?: string

  /** Agrupamento de Conversões (Order Bump & Upsell) */
  aggregation_enabled?: boolean
  aggregation_window_minutes?: number
}

export type CreatePixelInput = PixelInput
export type UpdatePixelInput = Partial<PixelInput>

export interface PixelsListResponse {
  data: Pixel[]
}

export interface PixelDefaults {
  tracker_config: TrackerConfig
}

export interface TestEventInput {
  event_name?: string
  email?: string
  phone?: string
  first_name?: string
  last_name?: string
  value?: number
  currency?: string
  // 'meta' (default), 'tiktok', 'google_ads' ou 'all'
  channel?: 'meta' | 'tiktok' | 'google_ads' | 'all'
}

export interface TestEventPixelResult {
  channel?: 'meta' | 'tiktok' | 'google_ads'
  pixel_id: string
  status_code: number
  response?: string
  error?: string
}

export interface TestEventResponse {
  event_id: string
  event_name: string
  channel?: 'meta' | 'tiktok' | 'google_ads' | 'all'
  results: TestEventPixelResult[]
}
