import type { Pixel, PixelInput, TrackerConfig } from '#shared/types'

export type PixelFormTabKey = 'basic' | 'meta' | 'tiktok' | 'gads' | 'ga4' | 'tracker'

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
    test_codes: [] as string[],
    tiktok_pixel_ids: [] as string[],
    tiktok_access_tokens: [] as string[],
    tiktok_test_codes: [] as string[],
    google_ads_account_id: null as number | null,
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
  const showTokens = ref(false)
  const errors = reactive({ name: '', pixel_ids: '' })

  const resetForm = () => {
    Object.assign(form, {
      name: '', label: '', pixel_ids: [], access_tokens: [], test_codes: [],
      tiktok_pixel_ids: [], tiktok_access_tokens: [], tiktok_test_codes: [],
      google_ads_account_id: null, meta_account_id: null, meta_ad_account_id: '', ga4_measurement_id: '', ga4_api_secret: '',
      domains: [], enabled: true, account_id: null, tracker_config: defaultTrackerConfig(),
      aggregation_enabled: false, aggregation_window_minutes: 10,
    })
    domainsInput.value = ''
    errors.name = ''
    errors.pixel_ids = ''
  }

  const fillFromPixel = (pixel: Pixel) => {
    const metaIds = pixel.pixel_ids || []
    const ttIds = pixel.tiktok_pixel_ids || []
    form.name = pixel.name
    form.label = pixel.label || ''
    form.pixel_ids = [...metaIds]
    form.access_tokens = metaIds.map(() => '')
    form.test_codes = [...(pixel.test_codes || metaIds.map(() => ''))]
    form.tiktok_pixel_ids = [...ttIds]
    form.tiktok_access_tokens = ttIds.map(() => '')
    form.tiktok_test_codes = [...(pixel.tiktok_test_codes || ttIds.map(() => ''))]
    form.google_ads_account_id = pixel.google_ads_account_id || null
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
  }

  watch(() => props.pixel, (p) => (p ? fillFromPixel(p) : resetForm()), { immediate: true })

  const validate = (): boolean => {
    errors.name = ''
    errors.pixel_ids = ''
    if (!form.name.trim()) {
      errors.name = 'Nome é obrigatório'
      activeTab.value = 'basic'
      return false
    }
    const metaIdxs = form.pixel_ids.map((id, i) => (id.trim() ? i : -1)).filter((i) => i !== -1)
    const ttIdxs = form.tiktok_pixel_ids.map((id, i) => (id.trim() ? i : -1)).filter((i) => i !== -1)
    const hasGoogleAds = !!(form.google_ads_account_id && form.google_ads_account_id > 0)
    const hasMetaOAuth = !!(form.meta_account_id && form.meta_account_id > 0)
    const hasGA4 = !!form.ga4_measurement_id.trim()
    if (!metaIdxs.length && !hasMetaOAuth && !ttIdxs.length && !hasGoogleAds && !hasGA4) {
      toast.error('Canal obrigatório', 'Configure pelo menos um canal: Meta, TikTok, Google Ads ou GA4.')
      return false
    }
    const oldMeta = new Set(props.pixel?.pixel_ids || [])
    if (!hasMetaOAuth && metaIdxs.some((i) => !(form.access_tokens[i] || '').trim() && !oldMeta.has(form.pixel_ids[i]!.trim()))) {
      errors.pixel_ids = 'Informe o Access Token ou selecione uma Conta Meta Ads (OAuth)'
      activeTab.value = 'meta'
      toast.error('Token Meta obrigatório', errors.pixel_ids)
      return false
    }
    const oldTt = new Set(props.pixel?.tiktok_pixel_ids || [])
    if (ttIdxs.some((i) => !(form.tiktok_access_tokens[i] || '').trim() && !oldTt.has(form.tiktok_pixel_ids[i]!.trim()))) {
      activeTab.value = 'tiktok'
      toast.error('Token TikTok obrigatório', 'Informe o Access Token para cada novo Pixel ID do TikTok.')
      return false
    }
    if (hasGA4 && !form.ga4_api_secret.trim() && !props.pixel?.ga4_measurement_id) {
      activeTab.value = 'ga4'
      toast.error('API Secret obrigatório', 'Informe o API Secret do GA4.')
      return false
    }
    return true
  }

  const buildPayload = (): PixelInput => {
    const indexes = form.pixel_ids.map((id, i) => (id.trim() ? i : -1)).filter((i) => i !== -1)
    const pixel_ids = indexes.map((i) => form.pixel_ids[i]!.trim())
    const test_codes = indexes.map((i) => (form.test_codes[i] || '').trim())
    const domains = domainsInput.value.split(',').map((d) => d.trim().toLowerCase()).filter(Boolean)

    const payload: PixelInput = {
      name: form.name.trim(),
      label: form.label.trim(),
      pixel_ids,
      test_codes,
      domains,
      enabled: form.enabled,
      tracker_config: form.tracker_config,
      aggregation_enabled: form.aggregation_enabled,
      aggregation_window_minutes: form.aggregation_window_minutes,
      google_ads_account_id: form.google_ads_account_id && form.google_ads_account_id > 0 ? form.google_ads_account_id : 0,
      meta_account_id: form.meta_account_id && form.meta_account_id > 0 ? form.meta_account_id : 0,
      meta_ad_account_id: form.meta_ad_account_id.trim(),
      ga4_measurement_id: form.ga4_measurement_id.trim(),
    }
    if (form.account_id) payload.account_id = form.account_id

    const tokens = indexes.map((i) => (form.access_tokens[i] || '').trim())
    if (indexes.length === 0 || tokens.some((t) => t.length > 0) || !isEditing.value) {
      payload.access_tokens = tokens
    }

    const ttIndexes = form.tiktok_pixel_ids.map((id, i) => (id.trim() ? i : -1)).filter((i) => i !== -1)
    payload.tiktok_pixel_ids = ttIndexes.map((i) => form.tiktok_pixel_ids[i]!.trim())
    payload.tiktok_test_codes = ttIndexes.map((i) => (form.tiktok_test_codes[i] || '').trim())

    const ttTokens = ttIndexes.map((i) => (form.tiktok_access_tokens[i] || '').trim())
    if (ttIndexes.length === 0 || ttTokens.some((t) => t.length > 0) || !isEditing.value) {
      payload.tiktok_access_tokens = ttTokens
    }

    if (form.ga4_api_secret || !form.ga4_measurement_id.trim()) {
      payload.ga4_api_secret = form.ga4_api_secret.trim()
    }

    return payload
  }

  return {
    isEditing,
    activeTab,
    form,
    domainsInput,
    showTokens,
    errors,
    validate,
    buildPayload,
  }
}

