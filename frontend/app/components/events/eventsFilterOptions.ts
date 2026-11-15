export type PeriodPresetId = 'today' | '24h' | '7d' | '30d'

export const EVENT_FILTER_OPTIONS = [
  { value: 'Purchase', label: 'Purchase' },
  { value: 'InitiateCheckout', label: 'InitiateCheckout' },
  { value: 'Lead', label: 'Lead' },
  { value: 'QualifiedLead', label: 'QualifiedLead' },
  { value: 'LeadSubmitted', label: 'LeadSubmitted' },
  { value: 'CompleteRegistration', label: 'CompleteRegistration' },
  { value: 'Contact', label: 'Contact' },
  { value: 'AddPaymentInfo', label: 'AddPaymentInfo' },
  { value: 'AddToCart', label: 'AddToCart' },
  { value: 'AddToWishlist', label: 'AddToWishlist' },
  { value: 'Search', label: 'Search' },
  { value: 'Subscribe', label: 'Subscribe' },
  { value: 'FindLocation', label: 'FindLocation' },
  { value: 'Schedule', label: 'Schedule' },
  { value: 'StartTrial', label: 'StartTrial' },
  { value: 'SubmitApplication', label: 'SubmitApplication' },
  { value: 'Donate', label: 'Donate' },
  { value: 'CustomizeProduct', label: 'CustomizeProduct' },
  { value: 'Scroll_25', label: 'Scroll_25' },
  { value: 'Scroll_50', label: 'Scroll_50' },
  { value: 'Scroll_75', label: 'Scroll_75' },
  { value: 'Scroll_90', label: 'Scroll_90' },
  { value: 'Timer_1min', label: 'Timer_1min' },
  { value: 'PlayVideo', label: 'PlayVideo' },
  { value: 'ViewVideo_25', label: 'ViewVideo_25' },
  { value: 'ViewVideo_50', label: 'ViewVideo_50' },
  { value: 'ViewVideo_75', label: 'ViewVideo_75' },
  { value: 'ViewVideo_90', label: 'ViewVideo_90' },
]

export const CHANNEL_FILTER_OPTIONS = [
  { value: '', label: 'Todos os canais' },
  { value: 'meta', label: 'Meta CAPI' },
  { value: 'google_ads', label: 'Google Ads' },
  { value: 'tiktok', label: 'TikTok Events' },
  { value: 'ga4', label: 'Google Analytics 4' },
]

export const STATUS_FILTER_OPTIONS = [
  { value: '', label: 'Todos os status' },
  { value: 'sent', label: 'Enviado (sem erros)' },
  { value: 'partial', label: 'Parcial (falha em canal)' },
  { value: 'failed', label: 'Com falha' },
]

export const PERIOD_PRESETS: Array<{ id: PeriodPresetId; label: string }> = [
  { id: 'today', label: 'Hoje' },
  { id: '24h', label: '24h' },
  { id: '7d', label: '7d' },
  { id: '30d', label: '30d' },
]

export function buildPresetFromISO(preset: PeriodPresetId): string {
  const now = new Date()
  if (preset === 'today') {
    const start = new Date(now.getFullYear(), now.getMonth(), now.getDate(), 0, 0, 0, 0)
    return start.toISOString()
  }
  const hoursMap: Record<Exclude<PeriodPresetId, 'today'>, number> = {
    '24h': 24,
    '7d': 24 * 7,
    '30d': 24 * 30,
  }
  return new Date(now.getTime() - hoursMap[preset] * 60 * 60 * 1000).toISOString()
}

export function isoToLocalInput(iso?: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  const pad = (n: number) => String(n).padStart(2, '0')
  return (
    d.getFullYear() +
    '-' +
    pad(d.getMonth() + 1) +
    '-' +
    pad(d.getDate()) +
    'T' +
    pad(d.getHours()) +
    ':' +
    pad(d.getMinutes())
  )
}
