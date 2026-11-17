import type { Pixel, PixelInput, TrackerConfig } from '#shared/types'

export type PixelFormTabKey = 'basic' | 'meta' | 'tiktok' | 'gads' | 'ga4' | 'tracker'

export const CLEAR_TOKEN_SENTINEL = '__CLEAR__'
const SLUG_REGEX = /^[a-zA-Z0-9_-]{1,64}$/

export const defaultTrackerConfig = (): TrackerConfig => ({
  product_value: [],
  product_currency: 'BRL',
  product_checkout: '',
  auto_page_view: true,
  auto_checkout_buttons: true,
  auto_form_scraping: true,
  auto_form_lead: true,
  whatsapp_protocol: true,
  whatsapp_lead: true,
  whatsapp_lead_event: 'Lead',
  send_scrolls: false,
  send_timer_1min: false,
  send_view_video: false,
  page_rules: [],
  form_mappings: {
    email: ['email', 'form_fields[email]'],
    phone: ['tel', 'telefone', 'phone', 'celular', 'cellphone', 'whatsapp', 'fone', 'form_fields[phone]'],
    first_name: ['nome', 'name', 'first_name', 'form_fields[name]', 'form_fields[firstname]'],
    last_name: ['sobrenome', 'lastname', 'last_name', 'form_fields[lastname]'],
  },
  meta_init_pixel_id: '',
  utm_origin_param: 'utm_source',
})

export const usePixelForm = (props: { pixel?: Pixel | null; initialTab?: PixelFormTabKey }) => {
  const toast = useToast()
  const isEditing = computed(() => !!props.pixel)
  const activeTab = ref<PixelFormTabKey>(props.initialTab || 'basic')

  watch(() => props.initialTab, (tab) => {
    if (tab) activeTab.value = tab
  })

  const form = reactive({
    name: '',
    label: '',
    pixel_ids: [] as string[],
    access_tokens: [] as string[],
    tiktok_pixel_ids: [] as string[],
    tiktok_access_tokens: [] as string[],
    google_ads_account_id: null as number | null,
    google_ads_customer_id: '',
    google_ads_customer_name: '',
    google_ads_login_customer_id: '',
    meta_account_id: null as number | null,
    meta_ad_account_id: '',
    ga4_measurement_id: '',
    ga4_api_secret: '',
    domains: [] as string[],
    enabled: true,
    account_id: null as number | null,
    tracker_config: defaultTrackerConfig(),
    aggregation_enabled: false,
    aggregation_window_minutes: 10,
  })

  const domainsInput = ref('')
  const errors = reactive({ name: '', pixel_ids: '' })
  const savedMetaTokens = reactive<Record<string, boolean>>({})
  const clearedMetaTokens = reactive<Record<string, boolean>>({})
  const savedTikTokTokens = reactive<Record<string, boolean>>({})

  const clearTokenMaps = () => {
    for (const k of Object.keys(savedMetaTokens)) delete savedMetaTokens[k]
    for (const k of Object.keys(clearedMetaTokens)) delete clearedMetaTokens[k]
    for (const k of Object.keys(savedTikTokTokens)) delete savedTikTokTokens[k]
  }

  const hasSavedManualMetaToken = (pixelId: string): boolean => {
    const pid = (pixelId || '').trim()
    return Boolean(pid && savedMetaTokens[pid] && !clearedMetaTokens[pid])
  }

  const clearManualMetaToken = (idx: number) => {
    const pid = (form.pixel_ids[idx] || '').trim()
    if (pid) clearedMetaTokens[pid] = true
    form.access_tokens[idx] = ''
  }

  const resetForm = () => {
    clearTokenMaps()
    Object.assign(form, {
      name: '', label: '', pixel_ids: [], access_tokens: [],
      tiktok_pixel_ids: [], tiktok_access_tokens: [],
      google_ads_account_id: null, google_ads_customer_id: '', google_ads_customer_name: '', google_ads_login_customer_id: '',
      meta_account_id: null, meta_ad_account_id: '', ga4_measurement_id: '', ga4_api_secret: '',
      domains: [], enabled: true, account_id: null, tracker_config: defaultTrackerConfig(),
      aggregation_enabled: false, aggregation_window_minutes: 10,
    })
    domainsInput.value = ''
    errors.name = ''
    errors.pixel_ids = ''
  }

  const fillFromPixel = (pixel: Pixel) => {
    clearTokenMaps()
    const metaIds = pixel.pixel_ids || []
    const ttIds = pixel.tiktok_pixel_ids || []
    metaIds.forEach((pid, i) => {
      if ((pixel.access_tokens?.[i] || '').trim()) savedMetaTokens[pid.trim()] = true
    })
    ttIds.forEach((pid, i) => {
      if ((pixel.tiktok_access_tokens?.[i] || '').trim()) savedTikTokTokens[pid.trim()] = true
    })
    form.name = pixel.name
    form.label = pixel.label || ''
    form.pixel_ids = [...metaIds]
    form.access_tokens = metaIds.map(() => '')
    form.tiktok_pixel_ids = [...ttIds]
    form.tiktok_access_tokens = ttIds.map(() => '')
    form.google_ads_account_id = pixel.google_ads_account_id || null
    form.google_ads_customer_id = pixel.google_ads_customer_id || ''
    form.google_ads_customer_name = pixel.google_ads_customer_name || ''
    form.google_ads_login_customer_id = pixel.google_ads_login_customer_id || ''
    form.meta_account_id = pixel.meta_account_id || null
    form.meta_ad_account_id = pixel.meta_ad_account_id || ''
    form.ga4_measurement_id = pixel.ga4_measurement_id || ''
    form.ga4_api_secret = ''
    form.domains = [...(pixel.domains || [])]
    form.enabled = pixel.enabled
    form.account_id = pixel.account_id || null
    form.aggregation_enabled = pixel.aggregation_enabled ?? false
    form.aggregation_window_minutes = pixel.aggregation_window_minutes || 10
    form.tracker_config = { ...defaultTrackerConfig(), ...(pixel.tracker_config || {}) }
    domainsInput.value = (pixel.domains || []).join(', ')
    errors.name = ''
    errors.pixel_ids = ''
  }

  watch(() => props.pixel, (p) => (p ? fillFromPixel(p) : resetForm()), { immediate: true })

  const validate = (): boolean => {
    errors.name = ''
    errors.pixel_ids = ''
    const cleanName = form.name.trim()
    if (!cleanName) {
      errors.name = 'Nome é obrigatório'
      activeTab.value = 'basic'
      return false
    }
    if (!isEditing.value && !SLUG_REGEX.test(cleanName)) {
      errors.name = 'Use apenas letras, números, hífen (-) ou underline (_), sem espaços'
      activeTab.value = 'basic'
      return false
    }
    const metaIdxs = form.pixel_ids.map((id, i) => (id.trim() ? i : -1)).filter((i) => i !== -1)
    const ttIdxs = form.tiktok_pixel_ids.map((id, i) => (id.trim() ? i : -1)).filter((i) => i !== -1)
    const hasGoogleAds = !!(form.google_ads_account_id && form.google_ads_account_id > 0)
    const hasMetaOAuth = !!(form.meta_account_id && form.meta_account_id > 0)
    const hasGA4 = !!form.ga4_measurement_id.trim()
    const hasOtherChannels = ttIdxs.length > 0 || hasGoogleAds || hasGA4

    if (hasMetaOAuth && !metaIdxs.length && !hasOtherChannels) {
      errors.pixel_ids = 'Selecione pelo menos um Pixel ID da Meta para disparar eventos via OAuth'
      activeTab.value = 'meta'
      toast.error('Pixel ID da Meta obrigatório', errors.pixel_ids)
      return false
    }
    if (!metaIdxs.length && !hasMetaOAuth && !hasOtherChannels) {
      toast.error('Canal obrigatório', 'Configure pelo menos um canal: Meta, TikTok, Google Ads ou GA4.')
      return false
    }
    if (!hasMetaOAuth && metaIdxs.some((i) => !(form.access_tokens[i] || '').trim() && !hasSavedManualMetaToken(form.pixel_ids[i]!))) {
      errors.pixel_ids = 'Informe o Access Token ou selecione uma Conta Meta Ads (OAuth)'
      activeTab.value = 'meta'
      toast.error('Token Meta obrigatório', errors.pixel_ids)
      return false
    }
    if (ttIdxs.some((i) => !(form.tiktok_access_tokens[i] || '').trim() && !savedTikTokTokens[form.tiktok_pixel_ids[i]!.trim()])) {
      activeTab.value = 'tiktok'
      toast.error('Token TikTok obrigatório', 'Informe o Access Token para cada novo Pixel ID do TikTok.')
      return false
    }
    if (hasGoogleAds && !form.google_ads_customer_id.trim()) {
      activeTab.value = 'gads'
      toast.error('Conta Google Ads obrigatória', 'Escolha qual Conta de Anúncios Google Ads deseja vincular a este Pixel.')
      return false
    }
    if (hasGA4 && !form.ga4_api_secret.trim() && !props.pixel?.ga4_api_secret) {
      activeTab.value = 'ga4'
      toast.error('API Secret obrigatório', 'Informe o API Secret do GA4.')
      return false
    }
    return true
  }

  const buildPayload = (): PixelInput => {
    const indexes = form.pixel_ids.map((id, i) => (id.trim() ? i : -1)).filter((i) => i !== -1)
    const pixel_ids = indexes.map((i) => form.pixel_ids[i]!.trim())
    const access_tokens = indexes.map((i) => {
      const typed = (form.access_tokens[i] || '').trim()
      if (typed) return typed
      const pid = form.pixel_ids[i]!.trim()
      return clearedMetaTokens[pid] ? CLEAR_TOKEN_SENTINEL : ''
    })
    const ttIndexes = form.tiktok_pixel_ids.map((id, i) => (id.trim() ? i : -1)).filter((i) => i !== -1)
    const tiktok_pixel_ids = ttIndexes.map((i) => form.tiktok_pixel_ids[i]!.trim())
    const cleanPrices = (form.tracker_config.product_value || []).filter((n) => typeof n === 'number' && Number.isFinite(n) && n > 0)

    const hasGads = Boolean(form.google_ads_account_id && form.google_ads_account_id > 0)
    const payload: PixelInput = {
      name: form.name.trim(),
      label: form.label.trim(),
      pixel_ids,
      access_tokens,
      test_codes: pixel_ids.map(() => ''),
      domains: domainsInput.value.split(',').map((d) => d.trim().toLowerCase()).filter(Boolean),
      enabled: form.enabled,
      tracker_config: { ...form.tracker_config, product_value: cleanPrices },
      aggregation_enabled: form.aggregation_enabled,
      aggregation_window_minutes: form.aggregation_window_minutes,
      google_ads_account_id: hasGads ? form.google_ads_account_id : 0,
      google_ads_customer_id: hasGads ? form.google_ads_customer_id.trim() : '',
      google_ads_customer_name: hasGads ? form.google_ads_customer_name.trim() : '',
      google_ads_login_customer_id: hasGads ? form.google_ads_login_customer_id.trim() : '',
      meta_account_id: form.meta_account_id && form.meta_account_id > 0 ? form.meta_account_id : 0,
      meta_ad_account_id: form.meta_ad_account_id.trim(),
      ga4_measurement_id: form.ga4_measurement_id.trim(),
      tiktok_pixel_ids,
      tiktok_access_tokens: ttIndexes.map((i) => (form.tiktok_access_tokens[i] || '').trim()),
      tiktok_test_codes: tiktok_pixel_ids.map(() => ''),
    }
    if (!isEditing.value && form.account_id) payload.account_id = form.account_id
    if (!form.ga4_measurement_id.trim()) {
      payload.ga4_api_secret = CLEAR_TOKEN_SENTINEL
    } else if (form.ga4_api_secret.trim()) {
      payload.ga4_api_secret = form.ga4_api_secret.trim()
    }
    return payload
  }

  return {
    isEditing, activeTab, form, domainsInput, errors,
    hasSavedManualMetaToken, clearManualMetaToken, validate, buildPayload,
  }
}
