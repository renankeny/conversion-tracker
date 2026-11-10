export interface CustomMappingConfig {
  first_name: string
  last_name: string
  email: string
  phone: string
  transaction_id: string
  amount: string
  status: string
  utm_source: string
  utm_medium: string
  utm_campaign: string
  pixel: string
  status_map: Record<string, string>
  _example_payload?: string
}

export interface CustomWebhook {
  id: string
  account_id: number | null
  name: string
  mapping_config: CustomMappingConfig
  created_at: string
  updated_at: string
}

export interface CreateCustomWebhookInput {
  name: string
  mapping_config: CustomMappingConfig
  is_global: boolean
}

export interface UpdateCustomWebhookInput {
  name?: string
  mapping_config?: CustomMappingConfig
}

