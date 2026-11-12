import type { Component } from 'vue'
import {
  Eye, Layers, UserPlus, Phone, ShoppingCart, Heart,
  Lock, CreditCard, CheckCircle, Receipt, AlertTriangle, DollarSign,
  TrendingUp, Target, MousePointerClick, Wallet, BarChart3, Award,
  MessageCircle, Percent,
} from 'lucide-vue-next'
import { MetaAdsIcon, GoogleAdsIcon } from '~/utils/adIcons'
import type { AdPlatformLiveInsights, PanelAudit, PreviousPeriodMetrics, SourceBreakdown } from '#shared/types'
import type { DashboardTrafficSource } from '~/stores/useDashboardStore'

export const FAILED_CARD_ID = '__failed__'
export const FAILED_CARD_ICON = AlertTriangle
export const FAILED_CARD_LABEL = 'Eventos com Falha'

export const REVENUE_CARD_ID = '__revenue__'
export const REVENUE_CARD_ICON = DollarSign
export const REVENUE_CARD_LABEL = 'Total Vendido'

export type CardFormat = 'number' | 'currency' | 'percent' | 'roas'
export type CardCategory = 'event' | 'general' | 'meta' | 'google'

export interface EventCatalogEntry {
  id: string
  label: string
  icon: Component
  variant: 'primary' | 'success' | 'warning' | 'danger' | 'info'
  color: string
  format?: CardFormat
  category?: CardCategory
  privateValue?: boolean
  changeInverse?: boolean
}

export const EVENT_CATALOG: EventCatalogEntry[] = [
  { id: 'PageView', label: 'Visitas (PageView)', icon: Eye, variant: 'primary', color: 'var(--brand)', category: 'event' },
  { id: 'ViewContent', label: 'View Content', icon: Layers, variant: 'primary', color: 'var(--brand-strong)', category: 'event' },
  { id: 'Lead', label: 'Leads', icon: UserPlus, variant: 'primary', color: 'var(--brand)', category: 'event' },
  { id: 'Contact', label: 'Contato', icon: Phone, variant: 'primary', color: 'var(--brand)', category: 'event' },
  { id: 'AddToCart', label: 'Add to Cart', icon: ShoppingCart, variant: 'primary', color: 'var(--brand)', category: 'event' },
  { id: 'AddToWishlist', label: 'Lista de Desejos', icon: Heart, variant: 'primary', color: 'var(--brand)', category: 'event' },
  { id: 'InitiateCheckout', label: 'Initiate Checkout', icon: Lock, variant: 'primary', color: 'var(--brand)', category: 'event' },
  { id: 'AddPaymentInfo', label: 'Add Payment Info', icon: CreditCard, variant: 'primary', color: 'var(--brand)', category: 'event' },
  { id: 'CompleteRegistration', label: 'Cadastro Completo', icon: CheckCircle, variant: 'primary', color: 'var(--brand)', category: 'event' },
  { id: 'Purchase', label: 'Vendas', icon: Receipt, variant: 'success', color: 'var(--success)', category: 'event' },
]

export const EXTRA_CARD_CATALOG: EventCatalogEntry[] = [
  { id: '__total_spend__', label: 'Investimento Total', icon: Wallet, variant: 'primary', color: 'var(--brand)', format: 'currency', category: 'general', privateValue: true },
  { id: '__total_cpa__', label: 'Custo por Venda (CPA)', icon: Award, variant: 'primary', color: 'var(--brand)', format: 'currency', category: 'general', changeInverse: true },
  { id: REVENUE_CARD_ID, label: REVENUE_CARD_LABEL, icon: DollarSign, variant: 'success', color: 'var(--success)', format: 'currency', category: 'general', privateValue: true },
  { id: '__ad_messages__', label: 'Mensagens de Anúncios', icon: MessageCircle, variant: 'info', color: 'var(--info)', format: 'number', category: 'general' },
  { id: '__total_cpl__', label: 'Custo por Lead (CPL)', icon: Target, variant: 'primary', color: 'var(--brand)', format: 'currency', category: 'general', changeInverse: true },
  { id: '__conv_rate__', label: 'Taxa de Conversão', icon: Percent, variant: 'primary', color: 'var(--brand)', format: 'percent', category: 'general' },
  { id: '__total_roas__', label: 'ROAS', icon: TrendingUp, variant: 'success', color: 'var(--success)', format: 'roas', category: 'general' },
  { id: '__avg_ticket__', label: 'Ticket Médio', icon: Receipt, variant: 'primary', color: 'var(--brand)', format: 'currency', category: 'general', privateValue: true },
  { id: '__ad_clicks__', label: 'Cliques nos Anúncios', icon: MousePointerClick, variant: 'primary', color: 'var(--brand)', format: 'number', category: 'general' },
  { id: '__avg_cpc__', label: 'CPC Médio', icon: BarChart3, variant: 'primary', color: 'var(--brand)', format: 'currency', category: 'general', changeInverse: true },
  { id: '__total_profit__', label: 'Lucro Bruto (Retorno - Gasto)', icon: DollarSign, variant: 'success', color: 'var(--success)', format: 'currency', category: 'general', privateValue: true },
  { id: FAILED_CARD_ID, label: FAILED_CARD_LABEL, icon: AlertTriangle, variant: 'danger', color: 'var(--danger)', format: 'number', category: 'general', changeInverse: true },

  // Meta Ads específicos
  { id: '__meta_spend__', label: 'Valor Gasto (Meta Ads)', icon: MetaAdsIcon, variant: 'primary', color: 'var(--brand)', format: 'currency', category: 'meta', privateValue: true },
  { id: '__meta_leads__', label: 'Leads (Meta Ads)', icon: MetaAdsIcon, variant: 'primary', color: 'var(--brand)', format: 'number', category: 'meta' },
  { id: '__meta_cpl__', label: 'Custo por Lead (Meta Ads)', icon: MetaAdsIcon, variant: 'primary', color: 'var(--brand)', format: 'currency', category: 'meta', changeInverse: true },
  { id: '__meta_purchases__', label: 'Compras (Meta Ads)', icon: MetaAdsIcon, variant: 'primary', color: 'var(--brand)', format: 'number', category: 'meta' },
  { id: '__meta_cpa__', label: 'Custo por Compra (Meta Ads)', icon: MetaAdsIcon, variant: 'primary', color: 'var(--brand)', format: 'currency', category: 'meta', changeInverse: true },
  { id: '__meta_revenue__', label: 'Valor Vendido (Meta Ads)', icon: MetaAdsIcon, variant: 'primary', color: 'var(--brand)', format: 'currency', category: 'meta', privateValue: true },
  { id: '__meta_roas__', label: 'ROAS (Meta Ads)', icon: MetaAdsIcon, variant: 'primary', color: 'var(--brand)', format: 'roas', category: 'meta' },
  { id: '__meta_clicks__', label: 'Cliques (Meta Ads)', icon: MetaAdsIcon, variant: 'primary', color: 'var(--brand)', format: 'number', category: 'meta' },
  { id: '__meta_cpc__', label: 'CPC Médio (Meta Ads)', icon: MetaAdsIcon, variant: 'primary', color: 'var(--brand)', format: 'currency', category: 'meta', changeInverse: true },

  // Google Ads específicos
  { id: '__google_spend__', label: 'Valor Gasto (Google Ads)', icon: GoogleAdsIcon, variant: 'primary', color: 'var(--brand)', format: 'currency', category: 'google', privateValue: true },
  { id: '__google_leads__', label: 'Leads (Google Ads)', icon: GoogleAdsIcon, variant: 'primary', color: 'var(--brand)', format: 'number', category: 'google' },
  { id: '__google_cpl__', label: 'Custo por Lead (Google Ads)', icon: GoogleAdsIcon, variant: 'primary', color: 'var(--brand)', format: 'currency', category: 'google', changeInverse: true },
  { id: '__google_purchases__', label: 'Compras (Google Ads)', icon: GoogleAdsIcon, variant: 'primary', color: 'var(--brand)', format: 'number', category: 'google' },
  { id: '__google_cpa__', label: 'Custo por Compra (Google Ads)', icon: GoogleAdsIcon, variant: 'primary', color: 'var(--brand)', format: 'currency', category: 'google', changeInverse: true },
  { id: '__google_revenue__', label: 'Valor Vendido (Google Ads)', icon: GoogleAdsIcon, variant: 'primary', color: 'var(--brand)', format: 'currency', category: 'google', privateValue: true },
  { id: '__google_roas__', label: 'ROAS (Google Ads)', icon: GoogleAdsIcon, variant: 'primary', color: 'var(--brand)', format: 'roas', category: 'google' },
  { id: '__google_clicks__', label: 'Cliques (Google Ads)', icon: GoogleAdsIcon, variant: 'primary', color: 'var(--brand)', format: 'number', category: 'google' },
  { id: '__google_cpc__', label: 'CPC Médio (Google Ads)', icon: GoogleAdsIcon, variant: 'primary', color: 'var(--brand)', format: 'currency', category: 'google', changeInverse: true },
]

export const FULL_CARD_CATALOG: EventCatalogEntry[] = [...EVENT_CATALOG, ...EXTRA_CARD_CATALOG]
export const CATALOG_IDS = EVENT_CATALOG.map((e) => e.id)
export const ALL_CARD_IDS = FULL_CARD_CATALOG.map((e) => e.id)

export const DEFAULT_ENABLED_CARDS = [
  '__total_spend__', 'Purchase', '__total_cpa__', REVENUE_CARD_ID,
  '__ad_messages__', 'Lead', '__total_cpl__', '__conv_rate__',
  '__total_roas__', '__avg_ticket__', '__ad_clicks__', '__avg_cpc__',
]

export const DEFAULT_FUNNEL_EVENTS = ['PageView', 'ViewContent', 'AddToCart', 'InitiateCheckout', 'Purchase']
export const catalogById = new Map(FULL_CARD_CATALOG.map((e) => [e.id, e]))

const ADS_METRIC_CARD_IDS = new Set(['__total_spend__', '__ad_clicks__', '__avg_cpc__', '__total_cpl__', '__total_cpa__', '__total_roas__'])

export function resolveCardIcon(entry: EventCatalogEntry, trafficSource?: DashboardTrafficSource): Component {
  if (entry.category === 'meta' || entry.id.startsWith('__meta_')) return MetaAdsIcon
  if (entry.category === 'google' || entry.id.startsWith('__google_')) return GoogleAdsIcon
  if (trafficSource === 'meta' && ADS_METRIC_CARD_IDS.has(entry.id)) return MetaAdsIcon
  if (trafficSource === 'google' && ADS_METRIC_CARD_IDS.has(entry.id)) return GoogleAdsIcon
  return entry.icon
}

export interface DashboardMetricsContext {
  sumOf: (event: string) => number
  failedTotal: number
  totalRevenue: number
  trafficSource?: DashboardTrafficSource
  privacyMode?: boolean
  metaInsights?: AdPlatformLiveInsights | null
  googleInsights?: AdPlatformLiveInsights | null
  sourceBreakdown?: SourceBreakdown | null
  panelAudit?: PanelAudit | null
  previousPeriod?: PreviousPeriodMetrics | null
}

const safeDiv = (num: number, den: number): number => (den > 0 ? num / den : 0)

export function resolveActiveSpendAndClicks(ctx: DashboardMetricsContext): { spend: number; clicks: number } {
  const mSpend = ctx.metaInsights?.spend ?? 0
  const mClicks = ctx.metaInsights?.clicks ?? 0
  const gSpend = ctx.googleInsights?.spend ?? 0
  const gClicks = ctx.googleInsights?.clicks ?? 0
  if (ctx.trafficSource === 'meta') return { spend: mSpend, clicks: mClicks }
  if (ctx.trafficSource === 'google') return { spend: gSpend, clicks: gClicks }
  return { spend: mSpend + gSpend, clicks: mClicks + gClicks }
}

export function resolvePanelLeads(ctx: DashboardMetricsContext): number {
  const evLeads = ctx.sumOf('Lead') + ctx.sumOf('LeadSubmitted') + ctx.sumOf('QualifiedLead')
  const audited = ctx.panelAudit?.leads.total ?? 0
  return Math.max(audited, evLeads)
}

export function resolvePanelPurchases(ctx: DashboardMetricsContext): number {
  return Math.max(ctx.panelAudit?.purchases.total ?? 0, ctx.sumOf('Purchase'))
}

export function computeCardMetricValue(id: string, ctx: DashboardMetricsContext): number {
  const { spend: activeSpend, clicks: activeClicks } = resolveActiveSpendAndClicks(ctx)
  const metaSpend = ctx.metaInsights?.spend ?? 0
  const metaClicks = ctx.metaInsights?.clicks ?? 0
  const googleSpend = ctx.googleInsights?.spend ?? 0
  const googleClicks = ctx.googleInsights?.clicks ?? 0

  const totalLeads = resolvePanelLeads(ctx)
  const totalPurchases = resolvePanelPurchases(ctx)
  const metaLeads = ctx.panelAudit?.leads.meta_total ?? ctx.sourceBreakdown?.meta.leads ?? 0
  const metaPurchases = ctx.panelAudit?.purchases.meta_total ?? ctx.sourceBreakdown?.meta.purchases ?? 0
  const metaRev = ctx.sourceBreakdown?.meta.revenue ?? 0
  const googleLeads = ctx.panelAudit?.leads.google_total ?? ctx.sourceBreakdown?.google.leads ?? 0
  const googlePurchases = ctx.panelAudit?.purchases.google_total ?? ctx.sourceBreakdown?.google.purchases ?? 0
  const googleRev = ctx.sourceBreakdown?.google.revenue ?? 0

  switch (id) {
    case FAILED_CARD_ID: return ctx.failedTotal
    case REVENUE_CARD_ID: return ctx.totalRevenue
    case '__ad_messages__': return ctx.panelAudit?.ad_messages.total ?? 0
    case '__total_spend__': return activeSpend
    case '__total_cpl__': return safeDiv(activeSpend, totalLeads)
    case '__total_cpa__': return safeDiv(activeSpend, totalPurchases)
    case '__conv_rate__': return safeDiv(totalPurchases, totalLeads)
    case '__total_roas__': return safeDiv(ctx.totalRevenue, activeSpend)
    case '__avg_ticket__': return safeDiv(ctx.totalRevenue, totalPurchases)
    case '__ad_clicks__': return activeClicks
    case '__avg_cpc__': return safeDiv(activeSpend, activeClicks)
    case '__total_profit__': return ctx.totalRevenue - activeSpend
    case '__meta_spend__': return metaSpend
    case '__meta_leads__': return metaLeads
    case '__meta_cpl__': return safeDiv(metaSpend, metaLeads || totalLeads)
    case '__meta_purchases__': return metaPurchases
    case '__meta_cpa__': return safeDiv(metaSpend, metaPurchases || totalPurchases)
    case '__meta_revenue__': return metaRev
    case '__meta_roas__': return safeDiv(metaRev || ctx.totalRevenue, metaSpend)
    case '__meta_clicks__': return metaClicks
    case '__meta_cpc__': return safeDiv(metaSpend, metaClicks)
    case '__google_spend__': return googleSpend
    case '__google_leads__': return googleLeads
    case '__google_cpl__': return safeDiv(googleSpend, googleLeads || totalLeads)
    case '__google_purchases__': return googlePurchases
    case '__google_cpa__': return safeDiv(googleSpend, googlePurchases || totalPurchases)
    case '__google_revenue__': return googleRev
    case '__google_roas__': return safeDiv(googleRev || ctx.totalRevenue, googleSpend)
    case '__google_clicks__': return googleClicks
    case '__google_cpc__': return safeDiv(googleSpend, googleClicks)
    case 'Lead': return totalLeads
    case 'Purchase': return totalPurchases
    default: return ctx.sumOf(id)
  }
}
