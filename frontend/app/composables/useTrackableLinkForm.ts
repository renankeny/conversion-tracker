import type { Pixel, TrackableLink, TrackableLinkInput, WhatsAppInstance } from '#shared/types'
import { formatOwnerJidPhone } from '~/utils/whatsappInstanceStatus'

export const useTrackableLinkForm = (
  props: { pixel: Pixel; editing?: TrackableLink | null; instances: WhatsAppInstance[] },
) => {
  const config = useRuntimeConfig()

  const form = reactive<TrackableLinkInput>({
    name: '',
    routing_mode: 'auto_pixel',
    whatsapp_instance_id: null,
    fallback_phone: '',
    prefill_message: 'Olá! Gostaria de mais informações.',
    utm_source: 'instagram',
    utm_medium: 'bio',
    utm_campaign: '',
    utm_content: '',
    utm_term: '',
    send_events_enabled: true,
    event_trigger_mode: 'on_first_message',
    event_name: 'Lead',
    send_meta: true,
    send_tiktok: true,
    send_google_ads: true,
    send_ga4: true,
    enabled: true,
  })

  const errors = reactive({
    name: '',
    destination: '',
  })

  const fillForm = (link?: TrackableLink | null) => {
    errors.name = ''
    errors.destination = ''
    if (link) {
      Object.assign(form, {
        name: link.name,
        routing_mode: link.routing_mode || 'auto_pixel',
        whatsapp_instance_id: link.whatsapp_instance_id ?? null,
        fallback_phone: link.fallback_phone || '',
        prefill_message: link.prefill_message || 'Olá! Gostaria de mais informações.',
        utm_source: link.utm_source || '',
        utm_medium: link.utm_medium || '',
        utm_campaign: link.utm_campaign || '',
        utm_content: link.utm_content || '',
        utm_term: link.utm_term || '',
        send_events_enabled: link.send_events_enabled,
        event_trigger_mode: link.event_trigger_mode || 'on_first_message',
        event_name: link.event_name || 'Lead',
        send_meta: link.send_meta,
        send_tiktok: link.send_tiktok,
        send_google_ads: link.send_google_ads,
        send_ga4: link.send_ga4,
        enabled: link.enabled,
      })
      return
    }
    const hasPixelInst = props.instances.length > 0
    Object.assign(form, {
      name: '',
      routing_mode: hasPixelInst ? 'auto_pixel' : 'manual_phone',
      whatsapp_instance_id: props.instances[0]?.id ?? null,
      fallback_phone: '',
      prefill_message: 'Olá! Gostaria de mais informações.',
      utm_source: 'instagram',
      utm_medium: 'bio',
      utm_campaign: '',
      utm_content: '',
      utm_term: '',
      send_events_enabled: true,
      event_trigger_mode: 'on_first_message',
      event_name: 'Lead',
      send_meta: Boolean((props.pixel.pixel_count ?? 0) > 0 || props.pixel.meta_account_id),
      send_tiktok: Boolean((props.pixel.tiktok_pixel_count ?? 0) > 0),
      send_google_ads: Boolean(props.pixel.google_ads_account_id),
      send_ga4: Boolean(props.pixel.ga4_measurement_id),
      enabled: true,
    })
  }

  watch(() => props.editing, (v) => fillForm(v), { immediate: true })

  const previewBaseOrigin = computed(() => {
    const th = (props.pixel.tracking_host || '').trim().replace(/^https?:\/\//, '').replace(/\/+$/, '')
    if (th) return `https://${th}`
    const rawApp = String(config.public.appUrl || '').trim().replace(/\/+$/, '')
    if (rawApp.startsWith('http') && !rawApp.includes('localhost:3000')) return rawApp
    if (import.meta.client) return window.location.origin
    if (rawApp.startsWith('http')) return rawApp
    const rawApi = String(config.public.apiUrl || '').trim().replace(/\/+$/, '').replace(/\/api\/v1$/, '')
    if (rawApi.startsWith('http')) return rawApi
    return ''
  })

  const previewUrl = computed(() => {
    if (props.editing?.slug) {
      return props.editing.public_url || `${previewBaseOrigin.value}/w/${props.editing.slug}`
    }
    const accPrefix = String(props.pixel.account_id || 1)
    const dots = '•'.repeat(Math.max(2, 10 - accPrefix.length))
    return `${previewBaseOrigin.value}/w/${accPrefix}${dots}`
  })

  const validate = (): TrackableLinkInput | null => {
    errors.name = ''
    errors.destination = ''
    if (!form.name.trim()) {
      errors.name = 'Informe um nome para identificar o link (ex: Bio Instagram).'
      return null
    }
    if (form.routing_mode === 'specific_instance' && !form.whatsapp_instance_id) {
      errors.destination = 'Selecione a conexão WhatsApp ou use o modo Automático / Rodízio.'
      return null
    }
    const digits = form.fallback_phone.replace(/\D/g, '')
    if (form.routing_mode === 'manual_phone' && digits.length < 10) {
      errors.destination = 'Informe o número completo com código do país e DDD (ex: 5511999998888).'
      return null
    }
    if (form.routing_mode !== 'manual_phone' && props.instances.length === 0 && digits.length < 10) {
      errors.destination = 'Este pixel ainda não tem WhatsApp conectado. Conecte um aparelho ou preencha o Número Reserva.'
      return null
    }
    if (form.routing_mode === 'specific_instance' && digits.length < 10) {
      const chosen = props.instances.find((i) => i.id === form.whatsapp_instance_id)
      if (chosen && !formatOwnerJidPhone(chosen.owner_jid)) {
        errors.destination = 'Esta conexão ainda não sincronizou o número do aparelho. Envie 1 mensagem de teste ou preencha o Número Reserva.'
        return null
      }
    }
    if (form.routing_mode === 'auto_pixel' && digits.length < 10 && props.instances.length > 0) {
      const anyHasPhone = props.instances.some((i) => Boolean(formatOwnerJidPhone(i.owner_jid)))
      if (!anyHasPhone) {
        errors.destination = 'Nenhuma conexão deste Pixel sincronizou o número ainda. Preencha o Número Reserva abaixo para ativar o redirecionamento.'
        return null
      }
    }
    return {
      ...form,
      name: form.name.trim(),
      whatsapp_instance_id: form.routing_mode === 'specific_instance' ? form.whatsapp_instance_id : null,
      fallback_phone: form.fallback_phone.trim(),
      prefill_message: form.prefill_message.trim() || 'Olá! Gostaria de mais informações.',
    }
  }

  return {
    form,
    errors,
    previewUrl,
    fillForm,
    validate,
  }
}
