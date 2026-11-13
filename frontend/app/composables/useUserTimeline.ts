import { ref, computed, watch, type Ref, type ComputedRef } from 'vue'
import type { GroupedEvent, TrackedUser, CustomerJourneyResponse } from '#shared/types'
import type { TimelineItem } from '~/components/users/UserJourneyTimeline.vue'
import { formatDateTimeSP } from '~/utils/formatDate'

const formatDate = (iso: string): string => formatDateTimeSP(iso, true)

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
      const [purchasesRes, journeyRes] = await Promise.all([
        eventsApi.listGrouped({ external_id: externalId, event: 'Purchase', limit: 50 }),
        trackedUsers.getJourney(externalId).catch(() => null),
      ])
      if (seq !== timelineSeq || userRef.value?.external_id !== externalId) return
      purchaseHistory.value = purchasesRes.data || []
      journeyData.value = journeyRes

      if (journeyRes && journeyRes.events && journeyRes.events.length) {
        rawEvents.value = journeyRes.events
      } else {
        const fallback = await eventsApi.listGrouped({ external_id: externalId, limit: 100 })
        if (seq !== timelineSeq || userRef.value?.external_id !== externalId) return
        rawEvents.value = fallback.data || []
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

    // 1. Descoberta
    if (firstTouch.created_at) {
      const meta: Array<{ label: string; value: string }> = []
      if (firstTouch.first_utm_source) meta.push({ label: 'Origem', value: firstTouch.first_utm_source })
      if (firstTouch.first_utm_campaign) meta.push({ label: 'Campanha', value: firstTouch.first_utm_campaign })
      if (firstTouch.city || firstTouch.state) meta.push({ label: 'Local', value: [firstTouch.city, firstTouch.state].filter(Boolean).join(', ') })
      if (firstTouch.gclid) meta.push({ label: 'GCLID', value: firstTouch.gclid.slice(0, 10) + '...' })
      if (firstTouch.fbc) meta.push({ label: 'FBC', value: 'Presente' })

      const initialSession = journeyData.value?.sessions?.find(s => s.is_initial)
      if (initialSession) {
        meta.push({ label: 'Dispositivo', value: `${initialSession.device_type} (${initialSession.browser})` })
      }

      items.push({
        badge: 'Descoberta',
        title: 'Primeiro Acesso ao Site',
        description: (firstTouch.first_url || firstTouch.url) ? `Acessou: ${firstTouch.first_url || firstTouch.url}` : undefined,
        date: formatDate(firstTouch.created_at),
        typeClass: 'type-discovery',
        meta,
        timestamp: new Date(firstTouch.created_at).getTime(),
      })
    }

    // 2. Transições
    if (journeyData.value?.sessions) {
      for (const s of journeyData.value.sessions) {
        if (!s.is_initial) {
          const meta: Array<{ label: string; value: string }> = [
            { label: 'Dispositivo', value: s.device_type },
            { label: 'Navegador', value: s.browser },
          ]
          if (s.first_utm_source) meta.push({ label: 'Origem da Sessão', value: s.first_utm_source })

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

    // 3. Eventos
    const seenEvents: Array<{ name: string; timeMs: number }> = []

    for (const ev of rawEvents.value) {
      const evTime = new Date(ev.created_at).getTime()
      if (seenEvents.some(s => s.name === ev.event_name && Math.abs(s.timeMs - evTime) < 3000)) {
        continue
      }
      seenEvents.push({ name: ev.event_name, timeMs: evTime })

      const meta: Array<{ label: string; value: string }> = []
      if (ev.pixel_name) meta.push({ label: 'Pixel', value: ev.pixel_name })

      if (ev.event_name === 'Purchase') {
        const valFormatted = ev.purchase_value > 0
          ? new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(ev.purchase_value)
          : 'Aprovada'
        if (ev.aggregated_count && ev.aggregated_count > 1) {
          meta.push({ label: 'Produtos', value: `${ev.aggregated_count} somados` })
        }
        const orderId = ev.aggregated_items?.[0]?.order_id
        items.push({
          badge: 'Conversão',
          title: `Compra Aprovada (${valFormatted})`,
          description: orderId ? `Pedido: #${orderId}` : undefined,
          date: formatDate(ev.created_at),
          typeClass: 'type-purchase',
          meta,
          timestamp: evTime,
        })
      } else if (ev.event_name === 'InitiateCheckout') {
        items.push({
          badge: 'Checkout',
          title: 'Iniciou Checkout',
          date: formatDate(ev.created_at),
          typeClass: 'type-checkout',
          meta,
          timestamp: evTime,
        })
      } else if (ev.event_name === 'CompleteRegistration') {
        items.push({
          badge: 'Cadastro',
          title: 'Cadastro Concluído',
          date: formatDate(ev.created_at),
          typeClass: 'type-lead',
          meta,
          timestamp: evTime,
        })
      } else if (ev.event_name === 'Lead') {
        items.push({
          badge: 'Lead',
          title: 'Captura de Lead / Formulário',
          date: formatDate(ev.created_at),
          typeClass: 'type-lead',
          meta,
          timestamp: evTime,
        })
      } else if (ev.event_name === 'Contact' || ev.event_name === 'LeadSubmitted') {
        items.push({
          badge: 'Contato',
          title: 'Iniciou Contato / Conversa',
          date: formatDate(ev.created_at),
          typeClass: 'type-contact',
          meta,
          timestamp: evTime,
        })
      }
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
