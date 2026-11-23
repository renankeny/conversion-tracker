import type { WhatsAppInstance } from '#shared/types'

export type InstanceConnectionState = 'connected' | 'connecting' | 'disconnected'

export const isWhatsAppStatusConnected = (status?: string | null, enabled = true): boolean => {
  if (!enabled) return false
  const norm = (status || '').trim().toUpperCase()
  return norm === 'OPEN' || norm === 'CONNECTED' || norm === 'ONLINE'
}

export const getWhatsAppStatusState = (status?: string | null, enabled = true): InstanceConnectionState => {
  if (!enabled) return 'disconnected'
  const norm = (status || '').trim().toUpperCase()
  if (norm === 'OPEN' || norm === 'CONNECTED' || norm === 'ONLINE') return 'connected'
  if (norm === 'CONNECTING') return 'connecting'
  return 'disconnected'
}

export const getWhatsAppProviderLabel = (provider?: string | null): string => {
  const norm = (provider || '').trim().toLowerCase()
  if (norm === 'api_premium') return 'API Premium'
  if (norm === 'uazapi') return 'Uazapi'
  return 'Conexão Direta'
}

export const getWhatsAppStatusLabel = (status?: string | null, provider?: string | null, enabled = true): string => {
  if (!enabled) return 'Pausada'
  const state = getWhatsAppStatusState(status, enabled)
  if (state === 'connected') return 'Conectado'
  if (state === 'connecting') {
    const p = (provider || '').trim().toLowerCase()
    return p === 'api_premium' || p === 'uazapi' ? 'Aguardando Webhook' : 'Pareando'
  }
  return 'Desconectado'
}

export const formatOwnerJidPhone = (ownerJid?: string | null): string => {
  const bare = (ownerJid || '').split('@')[0]?.split(':')[0] || ''
  const digits = bare.replace(/\D/g, '')
  if (!digits) return ''
  if (digits.startsWith('55') && (digits.length === 12 || digits.length === 13)) {
    const ddd = digits.slice(2, 4)
    const rest = digits.slice(4)
    if (rest.length === 9) return `+55 (${ddd}) ${rest.slice(0, 5)}-${rest.slice(5)}`
    return `+55 (${ddd}) ${rest.slice(0, 4)}-${rest.slice(4)}`
  }
  return `+${digits}`
}

export const formatInstanceOptionLabel = (inst: WhatsAppInstance): string => {
  const provider = getWhatsAppProviderLabel(inst.provider)
  const phone = formatOwnerJidPhone(inst.owner_jid)
  const status = getWhatsAppStatusLabel(inst.connection_status, inst.provider, inst.enabled)
  return phone
    ? `${inst.name} · ${provider} · ${phone} (${status})`
    : `${inst.name} · ${provider} (${status})`
}
