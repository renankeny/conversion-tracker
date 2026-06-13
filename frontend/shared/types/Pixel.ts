export interface TrackerConfig {
  enabled?: boolean
  auto_form_scraping?: boolean
  whatsapp_protocol?: boolean

  events?: {
    page_view?: boolean
    view_content?: boolean
    add_to_cart?: boolean
    initiate_checkout?: boolean
    purchase?: boolean
    lead?: boolean
    [key: string]: boolean | undefined
  }

  scroll_tracking?: {
    enabled?: boolean
    thresholds?: number[]
  }

  timer_tracking?: {
    enabled?: boolean
    intervals?: number[]
  }

  video_tracking?: {
    enabled?: boolean
  }

  page_rules?: Array<{
    path?: string
    event?: string
    enabled?: boolean
  }>

  product_values?: {
    value?: number
    currency?: string
  }

  meta_init_pixel_id?: string
  utm_origin_param?: string

  [key: string]: any
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
  pixel_count: number
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
