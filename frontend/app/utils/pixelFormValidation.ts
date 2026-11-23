import type { PixelFormTabKey } from '~/composables/usePixelForm'

const SLUG_REGEX = /^[a-zA-Z0-9_-]{1,64}$/

export interface PixelValidationContext {
  isEditing: boolean
  hasExistingGa4Secret: boolean
  form: {
    name: string
    pixel_ids: string[]
    access_tokens: string[]
    tiktok_pixel_ids: string[]
    tiktok_access_tokens: string[]
    google_ads_account_id: number | null
    google_ads_customer_id: string
    meta_account_id: number | null
    ga4_measurement_id: string
    ga4_api_secret: string
    sales_goal_enabled: boolean
    sales_goal_target: number
  }
  errors: { name: string; pixel_ids: string; sales_goal: string }
  hasSavedManualMetaToken: (pixelId: string) => boolean
  savedTikTokTokens: Record<string, boolean>
  setActiveTab: (tab: PixelFormTabKey) => void
  notifyError: (title: string, msg: string) => void
}

export const validatePixelFormState = (ctx: PixelValidationContext): boolean => {
  const { form, errors } = ctx
  errors.name = ''
  errors.pixel_ids = ''
  errors.sales_goal = ''

  const cleanName = form.name.trim()
  if (!cleanName) {
    errors.name = 'Nome é obrigatório'
    ctx.setActiveTab('basic')
    return false
  }
  if (!ctx.isEditing && !SLUG_REGEX.test(cleanName)) {
    errors.name = 'Use apenas letras, números, hífen (-) ou underline (_), sem espaços'
    ctx.setActiveTab('basic')
    return false
  }
  if (form.sales_goal_enabled && (!Number.isFinite(Number(form.sales_goal_target)) || Number(form.sales_goal_target) <= 0)) {
    errors.sales_goal = 'Informe uma meta maior que zero ou desative a meta de vendas'
    ctx.setActiveTab('basic')
    ctx.notifyError('Meta inválida', errors.sales_goal)
    return false
  }

  const metaIdxs = form.pixel_ids.map((id, i) => (id.trim() ? i : -1)).filter((i) => i !== -1)
  const ttIdxs = form.tiktok_pixel_ids.map((id, i) => (id.trim() ? i : -1)).filter((i) => i !== -1)
  const hasGoogleAds = Boolean(form.google_ads_account_id && form.google_ads_account_id > 0)
  const hasMetaOAuth = Boolean(form.meta_account_id && form.meta_account_id > 0)
  const hasGA4 = Boolean(form.ga4_measurement_id.trim())
  const hasOtherChannels = ttIdxs.length > 0 || hasGoogleAds || hasGA4

  if (hasMetaOAuth && !metaIdxs.length && !hasOtherChannels) {
    errors.pixel_ids = 'Selecione pelo menos um Pixel ID da Meta para disparar eventos via OAuth'
    ctx.setActiveTab('meta')
    ctx.notifyError('Pixel ID da Meta obrigatório', errors.pixel_ids)
    return false
  }
  if (!metaIdxs.length && !hasMetaOAuth && !hasOtherChannels) {
    ctx.notifyError('Canal obrigatório', 'Configure pelo menos um canal: Meta, TikTok, Google Ads ou GA4.')
    return false
  }
  if (!hasMetaOAuth && metaIdxs.some((i) => !(form.access_tokens[i] || '').trim() && !ctx.hasSavedManualMetaToken(form.pixel_ids[i]!))) {
    errors.pixel_ids = 'Informe o Access Token ou selecione uma Conta Meta Ads (OAuth)'
    ctx.setActiveTab('meta')
    ctx.notifyError('Token Meta obrigatório', errors.pixel_ids)
    return false
  }
  if (ttIdxs.some((i) => !(form.tiktok_access_tokens[i] || '').trim() && !ctx.savedTikTokTokens[form.tiktok_pixel_ids[i]!.trim()])) {
    ctx.setActiveTab('tiktok')
    ctx.notifyError('Token TikTok obrigatório', 'Informe o Access Token para cada novo Pixel ID do TikTok.')
    return false
  }
  if (hasGoogleAds && !form.google_ads_customer_id.trim()) {
    ctx.setActiveTab('gads')
    ctx.notifyError('Conta Google Ads obrigatória', 'Escolha qual Conta de Anúncios Google Ads deseja vincular a este Pixel.')
    return false
  }
  if (hasGA4 && !form.ga4_api_secret.trim() && !ctx.hasExistingGa4Secret) {
    ctx.setActiveTab('ga4')
    ctx.notifyError('API Secret obrigatório', 'Informe o API Secret do GA4.')
    return false
  }
  return true
}
