import type {
  CampaignRule,
  CampaignRuleActionType,
  CampaignRuleCondition,
  CampaignRulePayload,
} from '#shared/types'
import type { DropdownOption } from '~/components/BaseDropdown.vue'

export const RULE_PLATFORM_OPTIONS: DropdownOption[] = [
  { value: 'all', label: 'Meta Ads + Google Ads' },
  { value: 'meta', label: 'Somente Meta Ads' },
  { value: 'google', label: 'Somente Google Ads' },
]

export const RULE_LEVEL_OPTIONS: DropdownOption[] = [
  { value: 'campaign', label: 'Campanha' },
  { value: 'adset', label: 'Conjunto / Grupo de anúncios' },
  { value: 'ad', label: 'Anúncio individual' },
]

export const RULE_KIND_OPTIONS: DropdownOption[] = [
  { value: 'metric', label: 'Por métricas reais (Stop Loss / Escala / WhatsApp)' },
  { value: 'reactivate_late_sale', label: 'Reativar por venda atrasada (pausados hoje)' },
  { value: 'schedule', label: 'Agendamento por horário (Dayparting / Reset)' },
]

export const RULE_PERIOD_OPTIONS: DropdownOption[] = [
  { value: 'today', label: 'Hoje' },
  { value: 'yesterday', label: 'Ontem' },
  { value: '3d', label: 'Últimos 3 dias' },
  { value: '7d', label: 'Últimos 7 dias' },
  { value: '14d', label: 'Últimos 14 dias' },
  { value: '30d', label: 'Últimos 30 dias' },
]

export const RULE_ATTRIBUTION_OPTIONS: DropdownOption[] = [
  { value: 'last_click', label: 'Último clique (padrão)' },
  { value: 'first_click', label: 'Primeiro clique' },
]

export const RULE_TARGET_STATUS_OPTIONS: DropdownOption[] = [
  { value: 'ACTIVE', label: 'Somente ativos' },
  { value: 'PAUSED', label: 'Somente pausados' },
  { value: 'ALL', label: 'Ativos e pausados' },
]

export const RULE_METRIC_OPTIONS: DropdownOption[] = [
  { value: 'spend', label: 'Gasto (R$)' },
  { value: 'net_sales', label: 'Vendas líquidas (reais)' },
  { value: 'gross_sales', label: 'Vendas brutas' },
  { value: 'net_roas', label: 'ROAS líquido (x)' },
  { value: 'gross_roas', label: 'ROAS bruto (x)' },
  { value: 'net_profit', label: 'Lucro líquido (R$)' },
  { value: 'net_revenue', label: 'Receita líquida (R$)' },
  { value: 'real_cpa', label: 'CPA real (R$)' },
  { value: 'roi', label: 'ROI (%)' },
  { value: 'refund_count', label: 'Reembolsos (qtd)' },
  { value: 'refund_rate', label: 'Taxa de reembolso (%)' },
  { value: 'leads', label: 'Leads (qtd)' },
  { value: 'cost_per_lead', label: 'Custo por Lead (R$)' },
  { value: 'conversations', label: 'Conversas WhatsApp (qtd)' },
  { value: 'ai_qualified', label: 'Qualificados IA (qtd)' },
  { value: 'cost_per_qualified', label: 'Custo por Qualificado IA (R$)' },
  { value: 'impressions', label: 'Impressões' },
  { value: 'clicks', label: 'Cliques' },
  { value: 'ctr', label: 'CTR (%)' },
  { value: 'cpc', label: 'CPC (R$)' },
  { value: 'cpm', label: 'CPM (R$)' },
  { value: 'daily_budget', label: 'Orçamento diário atual (R$)' },
]

export const RULE_OPERATOR_OPTIONS: DropdownOption[] = [
  { value: '>', label: 'Maior que (>)' },
  { value: '>=', label: 'Maior ou igual (>=)' },
  { value: '<', label: 'Menor que (<)' },
  { value: '<=', label: 'Menor ou igual (<=)' },
  { value: '==', label: 'Igual a (=)' },
  { value: '!=', label: 'Diferente de (!=)' },
]

export const RULE_ACTION_OPTIONS: DropdownOption[] = [
  { value: 'pause', label: 'Pausar' },
  { value: 'activate', label: 'Ativar / Reativar' },
  { value: 'increase_budget_pct', label: 'Aumentar orçamento (%)' },
  { value: 'increase_budget_fixed', label: 'Aumentar orçamento (+ R$)' },
  { value: 'decrease_budget_pct', label: 'Reduzir orçamento (%)' },
  { value: 'decrease_budget_fixed', label: 'Reduzir orçamento (- R$)' },
  { value: 'set_budget', label: 'Definir orçamento fixo (R$)' },
  { value: 'notify_only', label: 'Apenas avisar (sem alterar)' },
]

export const RULE_FREQUENCY_OPTIONS: DropdownOption[] = [
  { value: 15, label: 'A cada 15 minutos' },
  { value: 30, label: 'A cada 30 minutos' },
  { value: 60, label: 'A cada 1 hora' },
  { value: 180, label: 'A cada 3 horas' },
  { value: 1440, label: '1 vez por dia' },
]

export const RULE_WEEKDAYS = [
  { value: 0, short: 'Dom' },
  { value: 1, short: 'Seg' },
  { value: 2, short: 'Ter' },
  { value: 3, short: 'Qua' },
  { value: 4, short: 'Qui' },
  { value: 5, short: 'Sex' },
  { value: 6, short: 'Sáb' },
]

export function isBudgetAction(action: CampaignRuleActionType): boolean {
  return [
    'increase_budget_pct',
    'increase_budget_fixed',
    'decrease_budget_pct',
    'decrease_budget_fixed',
    'set_budget',
  ].includes(action)
}

export function formatRuleMetric(metric: string): string {
  const found = RULE_METRIC_OPTIONS.find((o) => o.value === metric)
  return found ? found.label.replace(/\s*\([^)]*\)$/, '') : metric
}

export function formatMetricValue(metric: string, val: number): string {
  if (['spend', 'cpc', 'cpm', 'net_revenue', 'net_profit', 'real_cpa', 'cost_per_lead', 'cost_per_qualified', 'daily_budget'].includes(metric)) {
    return `R$ ${Number(val || 0).toFixed(2).replace('.', ',')}`
  }
  if (['ctr', 'refund_rate', 'roi'].includes(metric)) return `${Number(val || 0).toFixed(1)}%`
  if (['net_roas', 'gross_roas'].includes(metric)) return `${Number(val || 0).toFixed(2)}x`
  return String(Math.round(Number(val || 0)))
}

export function formatRuleConditionsSummary(conditions: CampaignRuleCondition[], scheduleTime?: string, ruleKind?: string): string {
  if (ruleKind === 'schedule' && (!conditions || conditions.length === 0)) {
    return `Horário programado às ${scheduleTime || '08:00'} (Brasília)`
  }
  if (!conditions || conditions.length === 0) return 'Sem condições'
  return conditions.map((c) => `${formatRuleMetric(c.metric)} ${c.operator} ${formatMetricValue(c.metric, c.value)}`).join(' E ')
}

export function formatRuleActionSummary(rule: Pick<CampaignRule, 'action_type' | 'action_value' | 'min_budget' | 'max_budget' | 'dry_run'>): string {
  const val = Number(rule.action_value || 0)
  const cap = rule.max_budget > 0 ? ` (teto R$ ${rule.max_budget.toFixed(0)})` : ''
  const floor = rule.min_budget > 0 ? ` (piso R$ ${rule.min_budget.toFixed(0)})` : ''
  const prefix = rule.dry_run && rule.action_type !== 'notify_only' ? '[Simular] ' : ''
  switch (rule.action_type) {
    case 'pause': return `${prefix}Pausar`
    case 'activate': return `${prefix}Reativar`
    case 'increase_budget_pct': return `${prefix}+${val}% orçamento${cap}`
    case 'increase_budget_fixed': return `${prefix}+R$ ${val.toFixed(2)} orçamento${cap}`
    case 'decrease_budget_pct': return `${prefix}-${val}% orçamento${floor}`
    case 'decrease_budget_fixed': return `${prefix}-R$ ${val.toFixed(2)} orçamento${floor}`
    case 'set_budget': return `${prefix}Orçamento fixo R$ ${val.toFixed(2)}`
    default: return 'Apenas avisar'
  }
}

export function defaultRulePayload(): CampaignRulePayload {
  return {
    name: '',
    description: '',
    template_key: '',
    enabled: true,
    dry_run: true,
    platform: 'all',
    pixel_name: '',
    level: 'adset',
    attribution_mode: 'last_click',
    rule_kind: 'metric',
    period_preset: 'today',
    conditions: [
      { metric: 'spend', operator: '>', value: 150 },
      { metric: 'net_sales', operator: '==', value: 0 },
    ],
    action_type: 'pause',
    action_value: 0,
    min_budget: 0,
    max_budget: 0,
    name_contains: '',
    name_excludes: '',
    target_status: 'ACTIVE',
    frequency_minutes: 30,
    min_hour_sp: 11,
    schedule_time: '',
    schedule_days: [],
    cooldown_minutes: 360,
    notify_push: true,
    notify_whatsapp: false,
    whatsapp_phone: '',
  }
}

export function cloneRulePayload(src?: Partial<CampaignRulePayload> | null): CampaignRulePayload {
  const base = defaultRulePayload()
  if (!src) return base
  return {
    ...base,
    ...src,
    template_key: src.template_key || '',
    conditions: (src.conditions || []).map((c) => ({ ...c })),
    schedule_days: [...(src.schedule_days || [])],
  }
}
