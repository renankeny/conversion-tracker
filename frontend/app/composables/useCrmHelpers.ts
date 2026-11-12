import type { CrmLead, CrmOriginBadge, CrmOriginCategory, CrmStage, TrackedUser } from '#shared/types'

const WON_STAGE_REGEX = /\b(ganh[oa]s?|fechad[oa]s?|conclu[ií]d[oa]s?|comprou|pago|venda\s+(?:ganha|fechada|conclu[ií]da|realizada))\b/i
const PRE_POST_SALE_REGEX = /\b(pr[eé][\s-]*venda|p[oó]s[\s-]*venda)\b/i
const LOST_STAGE_REGEX = /\b(perdid[oa]s?|descartad[oa]s?|arquivad[oa]s?|cancelad[oa]s?|sem\s+interesse)\b/i

export const useCrmHelpers = () => {
  const cleanUtmValue = (raw?: string | null): string => {
    if (!raw) return ''
    return String(raw)
      .replace(/\|(?:userId|uid)[A-Za-z0-9_-]+\|/gi, '')
      .replace(/^\|+|\|+$/g, '')
      .trim()
  }

  const formatCurrency = (val: number): string => {
    return new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(val || 0)
  }

  const formatDateTime = (dateString?: string): string => {
    if (!dateString) return ''
    const d = new Date(dateString)
    return new Intl.DateTimeFormat('pt-BR', {
      day: '2-digit',
      month: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
    }).format(d)
  }

  const timeAgo = (dateStr?: string): string => {
    if (!dateStr) return ''
    const diff = Date.now() - new Date(dateStr).getTime()
    const mins = Math.floor(diff / 60000)
    if (mins < 1) return 'agora'
    if (mins < 60) return `há ${mins}m`
    const hours = Math.floor(mins / 60)
    if (hours < 24) return `há ${hours}h`
    const days = Math.floor(hours / 24)
    if (days === 1) return 'ontem'
    if (days < 30) return `há ${days}d`
    return new Intl.DateTimeFormat('pt-BR', { day: '2-digit', month: '2-digit' }).format(new Date(dateStr))
  }

  const getOriginBadge = (tu?: Partial<TrackedUser> | null): CrmOriginBadge | null => {
    if (!tu) return null
    const src = cleanUtmValue(tu.last_utm_source || tu.first_utm_source).toLowerCase()

    if (tu.ctwa_clid) {
      return { label: 'Meta CTWA', class: 'origin-meta' }
    }
    if (
      src.includes('facebook') ||
      src.includes('instagram') ||
      src.includes('meta') ||
      src === 'fb' ||
      src === 'ig' ||
      tu.fbc
    ) {
      return { label: 'Meta Ads', class: 'origin-meta' }
    }
    if (
      src.includes('google') ||
      src.includes('adwords') ||
      src.includes('youtube') ||
      tu.gclid ||
      tu.wbraid ||
      tu.gbraid
    ) {
      return { label: 'Google Ads', class: 'origin-google' }
    }
    if (src.includes('tiktok') || src === 'tt' || tu.ttclid) {
      return { label: 'TikTok', class: 'origin-tiktok' }
    }
    if (src.includes('direto') || src.includes('direct') || src.includes('organic')) {
      return { label: 'Direto', class: 'origin-direct' }
    }
    if (src) {
      return { label: src.slice(0, 14), class: 'origin-other' }
    }
    return null
  }

  const getOriginCategory = (tu?: Partial<TrackedUser> | null): CrmOriginCategory => {
    const badge = getOriginBadge(tu)
    if (!badge || badge.class === 'origin-direct') return 'direct'
    if (badge.class === 'origin-meta') return 'meta'
    if (badge.class === 'origin-google') return 'google'
    if (badge.class === 'origin-tiktok') return 'tiktok'
    return 'other'
  }

  const isWonStage = (stage?: Pick<CrmStage, 'name' | 'trigger_event' | 'trigger_event_ctwa'> | null): boolean => {
    if (!stage) return false
    if (stage.trigger_event === 'Purchase' || stage.trigger_event_ctwa === 'Purchase') return true
    const name = stage.name || ''
    if (PRE_POST_SALE_REGEX.test(name)) return false
    return WON_STAGE_REGEX.test(name)
  }

  const isLostStage = (stage?: Pick<CrmStage, 'name' | 'trigger_event'> | null): boolean => {
    if (!stage) return false
    return LOST_STAGE_REGEX.test(stage.name || '')
  }

  const getStagnantDays = (dateStr?: string): number => {
    if (!dateStr) return 0
    const diff = Date.now() - new Date(dateStr).getTime()
    return Math.floor(diff / (1000 * 60 * 60 * 24))
  }

  const getLeadFullName = (lead: CrmLead): string => {
    const tu = lead.tracked_user
    if (tu?.first_name || tu?.last_name) {
      return `${tu.first_name || ''} ${tu.last_name || ''}`.trim()
    }
    return tu?.phone || tu?.email || `Lead #${lead.id}`
  }

  const getLeadInitials = (lead: CrmLead): string => {
    const tu = lead.tracked_user
    const name = tu?.first_name || tu?.last_name || tu?.email || 'L'
    return name.slice(0, 2).toUpperCase()
  }

  const normalizeWhatsAppPhone = (rawPhone?: string | null): string => {
    const digits = (rawPhone || '').replace(/\D/g, '').replace(/^0+/, '')
    if (!digits) return ''
    if (digits.length === 10 || digits.length === 11) {
      return `55${digits}`
    }
    return digits
  }

  const getWhatsAppLink = (leadOrPhone: CrmLead | string, firstName = ''): string => {
    const rawPhone = typeof leadOrPhone === 'string' ? leadOrPhone : leadOrPhone.tracked_user?.phone
    const name = typeof leadOrPhone === 'string' ? firstName : (leadOrPhone.tracked_user?.first_name || '')
    const phone = normalizeWhatsAppPhone(rawPhone)
    if (!phone) return '#'
    const greeting = name ? `Olá ${name}, tudo bem?` : 'Olá, tudo bem?'
    return `https://wa.me/${phone}?text=${encodeURIComponent(greeting)}`
  }

  return {
    cleanUtmValue,
    formatCurrency,
    formatDateTime,
    timeAgo,
    getOriginBadge,
    getOriginCategory,
    isWonStage,
    isLostStage,
    getStagnantDays,
    getLeadFullName,
    getLeadInitials,
    normalizeWhatsAppPhone,
    getWhatsAppLink,
  }
}
