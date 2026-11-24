import type { WhatsAppEventTrigger, WhatsAppInstance } from '#shared/types'

export const WHATSAPP_CUSTOM_EVENT_VALUE = '__custom__'

export interface EditableWhatsAppTriggerRow {
  uid: string
  selectedEvent: string
  customEventName: string
  trigger: string
}

export const WHATSAPP_TRIGGER_EVENT_OPTIONS = [
  { value: 'Purchase', label: 'Venda (Purchase)' },
  { value: 'QualifiedLead', label: 'Lead Qualificado (QualifiedLead)' },
  { value: 'Lead', label: 'Lead / Contato (Lead)' },
  { value: 'Schedule', label: 'Agendamento (Schedule)' },
  { value: 'InitiateCheckout', label: 'Início de Checkout / Negociação (InitiateCheckout)' },
  { value: 'AddToCart', label: 'Adicionar ao Carrinho (AddToCart)' },
  { value: 'AddPaymentInfo', label: 'Dados de Pagamento (AddPaymentInfo)' },
  { value: 'AddShippingInfo', label: 'Dados de Entrega (AddShippingInfo)' },
  { value: 'Contact', label: 'Contato Realizado (Contact)' },
  { value: 'CompleteRegistration', label: 'Cadastro Completo (CompleteRegistration)' },
  { value: 'Subscribe', label: 'Assinatura (Subscribe)' },
  { value: 'StartTrial', label: 'Teste Gratuito (StartTrial)' },
  { value: 'ViewContent', label: 'Visualização de Conteúdo (ViewContent)' },
  { value: 'Search', label: 'Pesquisa (Search)' },
  { value: 'AddToWishlist', label: 'Lista de Desejos (AddToWishlist)' },
  { value: 'OrderCreated', label: 'Pedido Criado (OrderCreated)' },
  { value: 'OrderShipped', label: 'Pedido Enviado (OrderShipped)' },
  { value: 'OrderDelivered', label: 'Pedido Entregue (OrderDelivered)' },
  { value: 'OrderCanceled', label: 'Pedido Cancelado (OrderCanceled)' },
  { value: 'OrderReturned', label: 'Pedido Devolvido (OrderReturned)' },
  { value: 'CartAbandoned', label: 'Carrinho Abandonado (CartAbandoned)' },
  { value: WHATSAPP_CUSTOM_EVENT_VALUE, label: 'Evento Personalizado (Digitar nome...)' },
]

const PRESET_VALUES = new Set(
  WHATSAPP_TRIGGER_EVENT_OPTIONS
    .map((o) => o.value)
    .filter((v) => v !== WHATSAPP_CUSTOM_EVENT_VALUE),
)

let uidCounter = 0
export const nextTriggerRowUid = (): string => `wa_trg_${Date.now()}_${++uidCounter}`

export const isPresetWhatsAppEvent = (name: string): boolean => PRESET_VALUES.has(name.trim())

export const getWhatsAppTriggerPlaceholder = (selectedEvent: string): string => {
  switch (selectedEvent) {
    case 'Purchase':
      return 'Ex: pagamento confirmado, pix recebido'
    case 'QualifiedLead':
    case 'Lead':
      return 'Ex: proposta enviada, orçamento enviado'
    case 'Schedule':
      return 'Ex: reunião agendada, horário confirmado'
    case 'InitiateCheckout':
      return 'Ex: link de pagamento enviado, segue o checkout'
    case 'AddToCart':
      return 'Ex: pedido separado, adicionado ao pedido'
    case 'AddPaymentInfo':
      return 'Ex: chave pix enviada, dados bancários'
    default:
      return 'Ex: termo 1, termo 2'
  }
}

export const pickSuggestedNextEvent = (rows: EditableWhatsAppTriggerRow[]): string => {
  const used = new Set(rows.map((r) => r.selectedEvent))
  const candidates = ['Purchase', 'QualifiedLead', 'Schedule', 'InitiateCheckout', 'Contact', 'AddToCart']
  for (const c of candidates) {
    if (!used.has(c)) return c
  }
  return WHATSAPP_CUSTOM_EVENT_VALUE
}

export const toEditableTriggerRows = (inst: WhatsAppInstance | null): EditableWhatsAppTriggerRow[] => {
  const raw: WhatsAppEventTrigger[] = []
  if (inst?.event_triggers?.length) {
    raw.push(...inst.event_triggers)
  } else if (inst) {
    if (inst.purchase_trigger?.trim()) raw.push({ event_name: 'Purchase', trigger: inst.purchase_trigger.trim() })
    if (inst.lead_trigger?.trim()) raw.push({ event_name: 'QualifiedLead', trigger: inst.lead_trigger.trim() })
    if (inst.custom_event_name?.trim() && inst.custom_event_trigger?.trim()) {
      raw.push({ event_name: inst.custom_event_name.trim(), trigger: inst.custom_event_trigger.trim() })
    }
  }
  if (!raw.some((r) => (r.event_name || '').trim().toLowerCase() === 'purchase')) {
    raw.unshift({ event_name: 'Purchase', trigger: '' })
  }
  return raw.map((item) => {
    const ev = (item.event_name || '').trim()
    const preset = isPresetWhatsAppEvent(ev)
    return {
      uid: nextTriggerRowUid(),
      selectedEvent: preset ? ev : WHATSAPP_CUSTOM_EVENT_VALUE,
      customEventName: preset ? '' : ev,
      trigger: item.trigger || '',
    }
  })
}

export const serializeEditableTriggerRows = (
  rows: EditableWhatsAppTriggerRow[],
): { triggers: WhatsAppEventTrigger[]; missingCustomName: boolean } => {
  const triggers: WhatsAppEventTrigger[] = []
  for (const r of rows) {
    const phrase = r.trigger.trim()
    if (!phrase) continue
    const evName = r.selectedEvent === WHATSAPP_CUSTOM_EVENT_VALUE
      ? r.customEventName.trim()
      : r.selectedEvent.trim()
    if (!evName) {
      return { triggers: [], missingCustomName: true }
    }
    triggers.push({ event_name: evName, trigger: phrase })
  }
  return { triggers, missingCustomName: false }
}

export const parseAverageTicketInput = (raw: string): number => {
  const cleaned = (raw || '').replace(/[^\d.,]/g, '').trim()
  if (!cleaned) return 0
  let normalized = cleaned
  const lastComma = cleaned.lastIndexOf(',')
  const lastDot = cleaned.lastIndexOf('.')
  if (lastComma > lastDot) {
    normalized = cleaned.replace(/\./g, '').replace(',', '.')
  } else if (lastDot > lastComma && lastComma !== -1) {
    normalized = cleaned.replace(/,/g, '')
  }
  const n = Number.parseFloat(normalized)
  if (!Number.isFinite(n) || n <= 0) return 0
  return Math.round(n * 100) / 100
}

export const formatAverageTicketInput = (val?: number | null): string => {
  if (!val || val <= 0) return ''
  return val.toFixed(2).replace('.', ',')
}
