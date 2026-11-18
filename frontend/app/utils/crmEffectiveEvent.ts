import type { CrmStage, TrackedUser } from '#shared/types'

export const CRM_TRIGGER_NONE = 'none'

/**
 * Espelha domain.IsWhatsAppOrCTWAUser do backend:
 * ctwa_clid || wa_ad_id || external_id.startsWith('wpp:')
 */
export function isCtwaTrackedUser(tu?: Partial<TrackedUser> | null): boolean {
  if (!tu) return false
  return Boolean(
    tu.ctwa_clid?.trim() ||
    tu.wa_ad_id?.trim() ||
    (tu.external_id && tu.external_id.trim().startsWith('wpp:')),
  )
}

/**
 * Espelha domain.EffectiveStageTrigger do backend, incluindo suporte a 'none'
 * ("Não disparar evento em CTWA") e herança automática de trigger_event.
 */
export function resolveStageEffectiveEvent(
  stage?: Pick<CrmStage, 'trigger_event' | 'trigger_event_ctwa'> | null,
  tu?: Partial<TrackedUser> | null,
): string {
  if (!stage) return ''
  const webEv = (stage.trigger_event || '').trim()
  if (isCtwaTrackedUser(tu)) {
    const ctwaEv = (stage.trigger_event_ctwa || '').trim()
    if (ctwaEv.toLowerCase() === CRM_TRIGGER_NONE) return ''
    if (ctwaEv) return ctwaEv
    if (webEv === 'Lead') return 'LeadSubmitted'
    return webEv
  }
  if (webEv.toLowerCase() === CRM_TRIGGER_NONE) return ''
  return webEv
}

export function resolveStageCtwaBadge(
  stage?: Pick<CrmStage, 'trigger_event' | 'trigger_event_ctwa'> | null,
): { label: string; disabled: boolean } | null {
  if (!stage) return null
  const ctwaEv = (stage.trigger_event_ctwa || '').trim()
  if (ctwaEv.toLowerCase() === CRM_TRIGGER_NONE) {
    return { label: 'WPP: Desativado', disabled: true }
  }
  if (ctwaEv) {
    return { label: `WPP: ${ctwaEv}`, disabled: false }
  }
  const webEv = (stage.trigger_event || '').trim()
  if (webEv && webEv.toLowerCase() !== CRM_TRIGGER_NONE) {
    const mapped = webEv === 'Lead' ? 'LeadSubmitted' : webEv
    return { label: `WPP: ${mapped}`, disabled: false }
  }
  return null
}

