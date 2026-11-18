import { ref, computed, watch, type Ref, type ComputedRef } from 'vue'
import type { GroupedEvent, TrackedUser, CustomerJourneyResponse } from '#shared/types'
import type { TimelineItem } from '~/components/users/UserJourneyTimeline.vue'
import { decodeUtm, formatCurrencyBRL } from '~/components/users/userFormatters'
import { formatDateTimeSP } from '~/utils/formatDate'

const formatDate = (iso: string): string => formatDateTimeSP(iso, true)

const isSentPurchase = (ev: GroupedEvent): boolean =>
  ev.event_name === 'Purchase' &&
  Array.isArray(ev.channels) &&
  ev.channels.some((c) => c.status === 'sent' && !c.has_error)

interface EventPresentation {
  badge: string
  title: string
  description?: string
  typeClass: string
}

const FUNNEL_EVENT_MAP: Record<string, EventPresentation> = {
  InitiateCheckout: { badge: 'Checkout', title: 'Iniciou Checkout', typeClass: 'type-checkout' },
  AddPaymentInfo: { badge: 'Pagamento', title: 'Gerou Pagamento (Pix / Boleto)', typeClass: 'type-checkout' },
  AddToCart: { badge: 'Carrinho', title: 'Adicionou ao Carrinho', typeClass: 'type-checkout' },
  CompleteRegistration: { badge: 'Cadastro', title: 'Cadastro Concluído', typeClass: 'type-lead' },
  QualifiedLead: { badge: 'Qualificado', title: 'Lead Qualificado', typeClass: 'type-lead' },
  Lead: { badge: 'Lead', title: 'Captura de Lead / Formulário', typeClass: 'type-lead' },
  Contact: { badge: 'Contato', title: 'Iniciou Contato / Conversa', typeClass: 'type-contact' },
  LeadSubmitted: { badge: 'Contato', title: 'Iniciou Contato / Conversa', typeClass: 'type-contact' },
  Schedule: { badge: 'Agenda', title: 'Agendamento Realizado', typeClass: 'type-contact' },
  Subscribe: { badge: 'Assinatura', title: 'Assinatura / Trial Iniciado', typeClass: 'type-purchase' },
  StartTrial: { badge: 'Assinatura', title: 'Assinatura / Trial Iniciado', typeClass: 'type-purchase' },
}

const buildEventPresentation = (
  ev: GroupedEvent,
  meta: Array<{ label: string; value: string }>
): EventPresentation | null => {
  const name = ev.event_name?.trim()
  if (!name) return null

  if (name === 'Purchase') {
    const hasSent = ev.channels?.some((c) => c.status === 'sent' && !c.has_error)
    const allFailed =
      Array.isArray(ev.channels) &&
      ev.channels.length > 0 &&
      ev.channels.every((c) => c.status === 'failed' || c.has_error)

    if (ev.aggregated_count && ev.aggregated_count > 1) {
      meta.push({ label: 'Produtos', value: `${ev.aggregated_count} somados` })
    }
    const orderId = ev.aggregated_items?.[0]?.order_id
    const desc = orderId ? `Pedido: #${orderId}` : undefined

    if (hasSent) {
      const valText = ev.purchase_value > 0 ? ` (${formatCurrencyBRL(ev.purchase_value)})` : ''
      return { badge: 'Conversão', title: `Compra Aprovada${valText}`, description: desc, typeClass: 'type-purchase' }
    }
    if (allFailed) {
      if (ev.purchase_value > 0) meta.push({ label: 'Valor', value: formatCurrencyBRL(ev.purchase_value) })
      return { badge: 'Falha', title: 'Tentativa de Envio de Compra (Falhou)', description: desc, typeClass: 'type-failed' }
    }
    return { badge: 'Pendente', title: 'Compra em Processamento', description: desc, typeClass: 'type-checkout' }
  }

  return FUNNEL_EVENT_MAP[name] || { badge: 'Evento', title: name, typeClass: 'type-default' }
}

export const useUserTimeline = (
  userRef: Ref<TrackedUser | null> | ComputedRef<TrackedUser | null>
) => {
  const eventsApi = useEvents()
  const trackedUsers = useTrackedUsers()

  const purchaseHistory = ref<GroupedEvent[]>([])
  const loadingPurchases = ref(false)
  const rawEvents = ref<GroupedEvent[]>([])
  const loadingTimeline = ref(false)
  const journeyData = ref<CustomerJourneyResponse | null>(null)
  let timelineSeq = 0

  const fetchPurchaseHistory = async (externalId: string) => {
    const seq = ++timelineSeq
    loadingPurchases.value = true
    loadingTimeline.value = true
    try {
      const journeyRes = await trackedUsers.getJourney(externalId).catch(() => null)
      if (seq !== timelineSeq || userRef.value?.external_id !== externalId) return
      journeyData.value = journeyRes

      if (journeyRes && Array.isArray(journeyRes.events)) {
        rawEvents.value = journeyRes.events
        purchaseHistory.value = journeyRes.events.filter(isSentPurchase)
      } else {
        const [purchasesRes, fallbackRes] = await Promise.all([
          eventsApi.listGrouped({ external_id: externalId, event: 'Purchase', status: 'sent', limit: 50 }),
          eventsApi.listGrouped({ external_id: externalId, limit: 100 }),
        ])
        if (seq !== timelineSeq || userRef.value?.external_id !== externalId) return
        rawEvents.value = fallbackRes.data || []
        purchaseHistory.value = (purchasesRes.data || []).filter(isSentPurchase)
      }
    } catch {
      if (seq !== timelineSeq) return
      purchaseHistory.value = []
      rawEvents.value = []
      journeyData.value = null
    } finally {
      if (seq === timelineSeq) {
        loadingPurchases.value = false
        loadingTimeline.value = false
      }
    }
  }

  watch(
    () => userRef.value?.external_id,
    (externalId) => {
      if (externalId) {
        void fetchPurchaseHistory(externalId)
      } else {
        timelineSeq++
        loadingPurchases.value = false
        loadingTimeline.value = false
        purchaseHistory.value = []
        rawEvents.value = []
        journeyData.value = null
      }
    },
    { immediate: true }
  )

  const timelineEvents = computed<TimelineItem[]>(() => {
    const items: Array<TimelineItem & { timestamp: number }> = []
    const u = userRef.value
    if (!u) return []

    const firstTouch = journeyData.value?.first_touch || u

    if (firstTouch.created_at) {
      const meta: Array<{ label: string; value: string }> = []
      const utmSource = decodeUtm(firstTouch.first_utm_source)
      const utmCampaign = decodeUtm(firstTouch.first_utm_campaign)
      if (utmSource) meta.push({ label: 'Origem', value: utmSource })
      if (utmCampaign) meta.push({ label: 'Campanha', value: utmCampaign })
      if (firstTouch.city || firstTouch.state) {
        meta.push({ label: 'Local', value: [firstTouch.city, firstTouch.state].filter(Boolean).join(', ') })
      }
      if (firstTouch.gclid) meta.push({ label: 'GCLID', value: firstTouch.gclid.slice(0, 10) + '...' })
      if (firstTouch.fbc) meta.push({ label: 'FBC', value: 'Presente' })
      if (firstTouch.ctwa_clid) meta.push({ label: 'CTWA CLID', value: 'Presente' })

      const isCtwa = Boolean(firstTouch.ctwa_clid || firstTouch.wa_ad_id)
      const isWpp = isCtwa || Boolean(firstTouch.external_id?.startsWith('wpp:'))
      const isOfflineOrigin = (firstTouch.pageview_count === 0) && !firstTouch.first_url && !firstTouch.url
      const initialSession = journeyData.value?.sessions?.find((s) => s.is_initial)
      if (initialSession && (!isOfflineOrigin || firstTouch.client_user_agent)) {
        meta.push({ label: 'Dispositivo', value: `${initialSession.device_type} (${initialSession.browser})` })
      }

      const firstUrl = firstTouch.first_url || firstTouch.url
      const originBadge = isOfflineOrigin ? (isCtwa ? 'Meta CTWA' : (isWpp ? 'WhatsApp' : 'Origem')) : 'Descoberta'
      const originTitle = isOfflineOrigin
        ? (isCtwa ? 'Entrada via Anúncio Click-to-WhatsApp (Meta CTWA)' : (isWpp ? 'Entrada via Conversa no WhatsApp' : 'Registro Inicial (Webhook / Integração)'))
        : 'Primeiro Acesso ao Site'

      items.push({
        badge: originBadge,
        title: originTitle,
        description: !isOfflineOrigin && firstUrl ? `Acessou: ${firstUrl}` : undefined,
        date: formatDate(firstTouch.created_at),
        typeClass: 'type-discovery',
        meta,
        timestamp: new Date(firstTouch.created_at).getTime(),
      })
    }

    if (journeyData.value?.sessions) {
      for (const s of journeyData.value.sessions) {
        if (!s.is_initial) {
          const meta: Array<{ label: string; value: string }> = [
            { label: 'Dispositivo', value: s.device_type },
            { label: 'Navegador', value: s.browser },
          ]
          const sessionSource = decodeUtm(s.first_utm_source)
          if (sessionSource) meta.push({ label: 'Origem da Sessão', value: sessionSource })

          items.push({
            badge: 'Transição',
            title: 'Novo Dispositivo / Navegador Detectado',
            description: `Sessão iniciada via ${s.device_type} (${s.browser}). Identidade unificada por dados cadastrais.`,
            date: formatDate(s.created_at),
            typeClass: 'type-device',
            meta,
            timestamp: new Date(s.created_at).getTime(),
          })
        }
      }
    }

    const seenEvents: Array<{ key: string; timeMs: number }> = []
    for (const ev of rawEvents.value) {
      const evTime = new Date(ev.created_at).getTime()
      const meta: Array<{ label: string; value: string }> = []
      if (ev.pixel_name) meta.push({ label: 'Pixel', value: ev.pixel_name })

      const presentation = buildEventPresentation(ev, meta)
      if (!presentation) continue

      const dedupKey = `${ev.event_name}:${presentation.typeClass}`
      if (seenEvents.some((s) => s.key === dedupKey && Math.abs(s.timeMs - evTime) < 3000)) {
        continue
      }
      seenEvents.push({ key: dedupKey, timeMs: evTime })

      items.push({
        badge: presentation.badge,
        title: presentation.title,
        description: presentation.description,
        date: formatDate(ev.created_at),
        typeClass: presentation.typeClass,
        meta,
        timestamp: evTime,
      })
    }

    return items.sort((a, b) => b.timestamp - a.timestamp)
  })

  return {
    purchaseHistory,
    loadingPurchases,
    rawEvents,
    loadingTimeline,
    journeyData,
    timelineEvents,
    fetchPurchaseHistory,
  }
}
