import type { TrackedUser } from '#shared/types'

export const decodeUtm = (val?: string | null): string => {
  if (!val) return ''
  try {
    return decodeURIComponent(val.replace(/\+/g, ' '))
  } catch {
    return val.replace(/\+/g, ' ')
  }
}

export const formatCurrencyBRL = (val?: number | null): string => {
  return new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(val || 0)
}

export const formatBrPhone = (raw?: string | null): string => {
  if (!raw) return ''
  const trimmed = raw.trim()
  const digits = trimmed.replace(/\D/g, '')
  if (!digits) return trimmed

  if (digits.startsWith('55') && (digits.length === 12 || digits.length === 13)) {
    const ddd = digits.slice(2, 4)
    const rest = digits.slice(4)
    const splitAt = rest.length === 9 ? 5 : 4
    return `+55 (${ddd}) ${rest.slice(0, splitAt)}-${rest.slice(splitAt)}`
  }

  if (digits.length === 10 || digits.length === 11) {
    const ddd = digits.slice(0, 2)
    const rest = digits.slice(2)
    const splitAt = rest.length === 9 ? 5 : 4
    return `(${ddd}) ${rest.slice(0, splitAt)}-${rest.slice(splitAt)}`
  }

  if (digits.length > 11) {
    return `+${digits}`
  }

  return trimmed
}

export const truncateText = (val: string | null | undefined, max: number): string => {
  if (!val) return ''
  return val.length > max ? `${val.slice(0, max)}...` : val
}

export const userFullName = (u?: TrackedUser | null): string => {
  if (!u) return ''
  const full = [u.first_name, u.last_name].filter(Boolean).join(' ').trim()
  if (full) return full
  return u.wa_name?.trim() || ''
}

export const isAnonymousUser = (u?: TrackedUser | null): boolean => {
  if (!u) return true
  return !userFullName(u) && !u.email?.trim() && !u.phone?.trim()
}

export const userPrimaryLabel = (u?: TrackedUser | null): string => {
  if (!u) return 'Visitante anônimo'
  const name = userFullName(u)
  if (name) return name
  if (u.phone?.trim()) return formatBrPhone(u.phone)
  if (u.email?.trim()) return u.email.trim()
  return 'Visitante anônimo'
}

export const userSecondaryLine = (u: TrackedUser): string => {
  const name = userFullName(u)
  const email = u.email?.trim() || ''
  const phone = u.phone?.trim() ? formatBrPhone(u.phone) : ''

  if (name) {
    return [email, phone].filter(Boolean).join(' · ')
  }
  if (phone && email) {
    return email
  }
  return ''
}

export const userQuickCopyTarget = (u: TrackedUser): { value: string; label: string } => {
  if (u.phone?.trim()) return { value: u.phone.trim(), label: 'Copiar telefone' }
  if (u.email?.trim()) return { value: u.email.trim(), label: 'Copiar e-mail' }
  return { value: u.external_id, label: 'Copiar ID' }
}

export const userInitials = (u?: TrackedUser | null): string => {
  if (!u) return '?'
  const name = userFullName(u)
  if (name) {
    const parts = name.split(/\s+/).filter(Boolean)
    if (parts.length === 1) return (parts[0]?.slice(0, 2) || '?').toUpperCase()
    const first = parts[0]?.charAt(0) || ''
    const last = parts[parts.length - 1]?.charAt(0) || ''
    return (first + last).toUpperCase()
  }
  if (u.email?.trim()) return u.email.trim().slice(0, 2).toUpperCase()
  if (u.phone?.trim()) {
    const digits = u.phone.replace(/\D/g, '')
    if (digits.startsWith('55') && digits.length >= 4) return digits.slice(2, 4)
    return digits.slice(0, 2) || 'BR'
  }
  return 'VA'
}

export const hasUserGeo = (u?: TrackedUser | null): boolean => {
  return Boolean(u?.city?.trim() || u?.state?.trim() || u?.country?.trim())
}

export const formatUserGeo = (u?: TrackedUser | null): string => {
  if (!u) return '-'
  const parts = [
    u.city?.trim(),
    u.state?.trim().toUpperCase(),
    u.country?.trim().toUpperCase(),
  ].filter(Boolean)
  return parts.length ? parts.join(', ') : '-'
}

export const countryFlag = (country?: string | null): string => {
  const code = country?.trim().toUpperCase() || ''
  if (!/^[A-Z]{2}$/.test(code)) return '🌐'
  const offset = 0x1f1a5
  return String.fromCodePoint(code.charCodeAt(0) + offset, code.charCodeAt(1) + offset)
}

export const trafficPillLabel = (type?: TrackedUser['traffic_type']): string => {
  if (type === 'bot') return 'Bot'
  if (type === 'inconclusive') return 'Inconclusivo'
  return 'Humano'
}

export const trafficPillClass = (type?: TrackedUser['traffic_type']): string => {
  if (type === 'bot') return 'is-bot'
  if (type === 'inconclusive') return 'is-inconclusive'
  return 'is-human'
}

export interface UserOriginDisplay {
  primary: string
  campaign: string
  isDirect: boolean
  isEmpty: boolean
}

export const resolveUserOrigin = (u: TrackedUser): UserOriginDisplay => {
  const lastSrc = decodeUtm(u.last_utm_source).trim()
  const firstSrc = decodeUtm(u.first_utm_source).trim()
  const src = lastSrc || firstSrc
  const rawCamp = lastSrc ? u.last_utm_campaign : (u.last_utm_campaign || u.first_utm_campaign)
  const campaign = truncateText(decodeUtm(rawCamp).trim(), 22)

  if (src || campaign) {
    return { primary: src || 'UTM', campaign, isDirect: false, isEmpty: false }
  }
  if (u.ctwa_clid) {
    return { primary: 'WhatsApp CTWA', campaign: '', isDirect: false, isEmpty: false }
  }
  if (u.first_url || u.url || (u.pageview_count ?? 0) > 0) {
    return { primary: 'Direto', campaign: '', isDirect: true, isEmpty: false }
  }
  return { primary: '-', campaign: '', isDirect: false, isEmpty: true }
}
