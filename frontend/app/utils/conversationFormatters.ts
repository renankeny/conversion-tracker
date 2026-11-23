import type { ConversationAuditItem } from '#shared/types'

export const formatDurationSLA = (sec?: number | null): string => {
  if (sec === null || sec === undefined || sec < 0) return '-'
  if (sec < 60) return `${sec}s`
  const totalMin = Math.floor(sec / 60)
  if (totalMin < 60) return `${totalMin}min`
  const hours = Math.floor(totalMin / 60)
  const mins = totalMin % 60
  return mins > 0 ? `${hours}h${String(mins).padStart(2, '0')}m` : `${hours}h`
}

export const formatRelativeElapsed = (iso?: string | null): string => {
  if (!iso) return '-'
  const diffSec = Math.max(0, Math.floor((Date.now() - new Date(iso).getTime()) / 1000))
  if (diffSec < 60) return 'agora'
  const mins = Math.floor(diffSec / 60)
  if (mins < 60) return `${mins}min`
  const hours = Math.floor(mins / 60)
  if (hours < 24) return `${hours}h`
  const days = Math.floor(hours / 24)
  return `${days}d`
}

export const formatAuditDateTime = (iso?: string | null): string => {
  if (!iso) return '-'
  const d = new Date(iso)
  return new Intl.DateTimeFormat('pt-BR', {
    timeZone: 'America/Sao_Paulo',
    day: '2-digit',
    month: '2-digit',
    year: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(d)
}

export const formatAuditMoney = (val: number, currency = 'BRL'): string => {
  try {
    return new Intl.NumberFormat('pt-BR', { style: 'currency', currency: currency || 'BRL' }).format(val || 0)
  } catch {
    return `R$ ${(val || 0).toFixed(2)}`
  }
}

export const responseToneClass = (sec?: number | null): string => {
  if (sec === null || sec === undefined) return 'tone-muted'
  if (sec <= 300) return 'tone-good'
  if (sec <= 1800) return 'tone-warn'
  return 'tone-bad'
}

export const inboundWaitToneClass = (item: ConversationAuditItem): string => {
  if (!item.last_inbound_at) return 'tone-muted'
  if (item.sla_status === 'replied') return 'tone-neutral'
  const diffMin = Math.max(0, Math.floor((Date.now() - new Date(item.last_inbound_at).getTime()) / 60000))
  if (diffMin <= 10) return 'tone-good'
  if (diffMin <= 45) return 'tone-warn'
  return 'tone-bad'
}

export const originMetaOf = (cat: string): { label: string; badgeClass: string } => {
  switch (cat) {
    case 'meta_ads':
      return { label: 'Meta Ads', badgeClass: 'origin-meta' }
    case 'google_ads':
      return { label: 'Google Ads', badgeClass: 'origin-google' }
    case 'tiktok_ads':
      return { label: 'TikTok Ads', badgeClass: 'origin-tiktok' }
    case 'bio_link':
      return { label: 'Link na Bio', badgeClass: 'origin-bio' }
    default:
      return { label: 'Site', badgeClass: 'origin-site' }
  }
}

export const slaStatusMetaOf = (status: string): { label: string; statusClass: string } => {
  switch (status) {
    case 'replied':
      return { label: 'Respondido', statusClass: 'sla-replied' }
    case 'waiting_reply':
      return { label: 'Aguardando', statusClass: 'sla-waiting' }
    default:
      return { label: 'Sem resposta', statusClass: 'sla-unanswered' }
  }
}

const csvEscape = (v: unknown): string => {
  const s = v === null || v === undefined ? '' : String(v)
  if (/[;"\n\r]/.test(s)) return `"${s.replace(/"/g, '""')}"`
  return s
}

export const downloadConversationsCSV = (items: ConversationAuditItem[], filename = 'conversas-auditoria.csv') => {
  if (!import.meta.client) return
  const headers = [
    'Data Ultima Mensagem', 'Lead', 'Telefone', 'E-mail', 'Origem', 'Campanha', 'Anuncio', 'UTM Source',
    'Link Rastreavel', 'Etapa CRM', 'Valor', 'Venda Convertida', 'Status Atendimento',
    '1a Resposta', 'Tempo Medio Resposta', 'Total Mensagens', 'Qualificado IA',
  ]
  const lines = [headers.join(';')]
  for (const it of items) {
    lines.push([
      formatAuditDateTime(it.last_message_time),
      it.lead_name,
      it.phone,
      it.email || '',
      originMetaOf(it.origin_category).label,
      it.campaign_name || '',
      it.ad_name || '',
      it.utm_source || '',
      it.trackable_link_name || '',
      it.stage_name || 'Sem etapa',
      it.lead_value > 0 ? String(it.lead_value).replace('.', ',') : '',
      it.is_won ? 'Sim' : 'Nao',
      slaStatusMetaOf(it.sla_status).label,
      formatDurationSLA(it.first_response_sec),
      formatDurationSLA(it.avg_response_sec),
      it.total_messages,
      it.ai_qualified ? 'Sim' : 'Nao',
    ].map(csvEscape).join(';'))
  }
  const blob = new Blob(['\uFEFF' + lines.join('\r\n')], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}
