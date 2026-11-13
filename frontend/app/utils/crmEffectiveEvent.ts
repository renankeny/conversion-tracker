import type { CrmStage, TrackedUser } from '#shared/types'

/**
 * Reproduz exatamente a regra de CTWA do backend (admin_crm_leads.go MoveLead):
 * ctwa_clid || wa_ad_id || (id_wpp && external_id.startsWith('wpp:'))
 */
export function isCtwaTrackedUser(tu?: Partial<TrackedUser> | null): boolean {
  if (!tu) return false
  return Boolean(
    tu.ctwa_clid ||
    tu.wa_ad_id ||
    (tu.id_wpp && tu.external_id && tu.external_id.startsWith('wpp:')),
  )
}

export function resolveStageEffectiveEvent(
  stage?: Pick<CrmStage, 'trigger_event' | 'trigger_event_ctwa'> | null,
  tu?: Partial<TrackedUser> | null,
): string {
  if (!stage) return ''
  if (isCtwaTrackedUser(tu) && stage.trigger_event_ctwa) {
    return stage.trigger_event_ctwa
  }
  return stage.trigger_event || ''
}

