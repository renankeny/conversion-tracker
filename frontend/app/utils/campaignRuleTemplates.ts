import type { CampaignRulePayload } from '#shared/types'
import { CORE_CAMPAIGN_RULE_TEMPLATES } from './campaignRuleTemplatesCore'
import { OPS_CAMPAIGN_RULE_TEMPLATES } from './campaignRuleTemplatesOps'

export type RuleTemplateCategory = 'stop_loss' | 'scale' | 'whatsapp_ai' | 'recovery_schedule'

export interface CampaignRuleTemplate {
  key: string
  category: RuleTemplateCategory
  categoryLabel: string
  title: string
  summary: string
  badge: string
  payload: CampaignRulePayload
}

export const CAMPAIGN_RULE_TEMPLATES: CampaignRuleTemplate[] = [
  ...CORE_CAMPAIGN_RULE_TEMPLATES,
  ...OPS_CAMPAIGN_RULE_TEMPLATES,
]

export const RULE_TEMPLATE_CATEGORIES: Array<{ key: 'all' | RuleTemplateCategory; label: string }> = [
  { key: 'all', label: `Todos (${CAMPAIGN_RULE_TEMPLATES.length})` },
  { key: 'stop_loss', label: 'Proteção de Caixa' },
  { key: 'scale', label: 'Escala Automática' },
  { key: 'whatsapp_ai', label: 'WhatsApp & IA' },
  { key: 'recovery_schedule', label: 'Recuperação & Horários' },
]
