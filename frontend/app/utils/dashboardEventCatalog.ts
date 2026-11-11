import type { Component } from 'vue'
import {
  Eye, Layers, UserPlus, Phone, ShoppingCart, Heart,
  Lock, CreditCard, CheckCircle, Receipt, AlertTriangle, DollarSign,
  TrendingUp, Target, MousePointerClick, Wallet, BarChart3, Award,
} from 'lucide-vue-next'
import type { AdPlatformLiveInsights, SourceBreakdown } from '#shared/types'

export const FAILED_CARD_ID = '__failed__'
export const FAILED_CARD_ICON = AlertTriangle
export const FAILED_CARD_LABEL = 'Eventos com Falha'

export const REVENUE_CARD_ID = '__revenue__'
export const REVENUE_CARD_ICON = DollarSign
export const REVENUE_CARD_LABEL = 'Valor Vendido no Período'

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
  { id: 'Purchase', label: 'Vendas (Purchase)', icon: Receipt, variant: 'primary', color: 'var(--brand)', category: 'event' },
]

export const EXTRA_CARD_CATALOG: EventCatalogEntry[] = [
  // Geral & Performance Consolidada
  { id: REVENUE_CARD_ID, label: REVENUE_CARD_LABEL, icon: DollarSign, variant: 'success', color: 'var(--success)', format: 'currency', category: 'general' },
  { id: '__total_spend__', label: 'Investimento Total (Ads)', icon: Wallet, variant: 'primary', color: 'var(--brand)', format: 'currency', category: 'general' },
  { id: '__total_cpl__', label: 'Custo por Lead (CPL Geral)', icon: Target, variant: 'primary', color: 'var(--brand)', format: 'currency', category: 'general' },
  { id: '__total_cpa__', label: 'Custo por Compra (CPA Geral)', icon: Award, variant: 'primary', color: 'var(--brand)', format: 'currency', category: 'general' },
  { id: '__total_roas__', label: 'ROAS Geral', icon: TrendingUp, variant: 'success', color: 'var(--success)', format: 'roas', category: 'general' },
  { id: '__total_profit__', label: 'Lucro Bruto (Retorno - Gasto)', icon: DollarSign, variant: 'success', color: 'var(--success)', format: 'currency', category: 'general' },
  { id: FAILED_CARD_ID, label: FAILED_CARD_LABEL, icon: AlertTriangle, variant: 'danger', color: 'var(--danger)', format: 'number', category: 'general' },

  // Meta Ads
  { id: '__meta_spend__', label: 'Valor Gasto (Meta Ads)', icon: Wallet, variant: 'info', color: 'var(--info)', format: 'currency', category: 'meta' },
  { id: '__meta_leads__', label: 'Leads (Meta Ads)', icon: UserPlus, variant: 'info', color: 'var(--info)', format: 'number', category: 'meta' },
  { id: '__meta_cpl__', label: 'Custo por Lead (Meta Ads)', icon: Target, variant: 'info', color: 'var(--info)', format: 'currency', category: 'meta' },
  { id: '__meta_purchases__', label: 'Compras (Meta Ads)', icon: Receipt, variant: 'info', color: 'var(--info)', format: 'number', category: 'meta' },
  { id: '__meta_cpa__', label: 'Custo por Compra (Meta Ads)', icon: Award, variant: 'info', color: 'var(--info)', format: 'currency', category: 'meta' },
  { id: '__meta_revenue__', label: 'Valor Vendido (Meta Ads)', icon: DollarSign, variant: 'success', color: 'var(--success)', format: 'currency', category: 'meta' },
  { id: '__meta_roas__', label: 'ROAS (Meta Ads)', icon: TrendingUp, variant: 'success', color: 'var(--success)', format: 'roas', category: 'meta' },
  { id: '__meta_clicks__', label: 'Cliques (Meta Ads)', icon: MousePointerClick, variant: 'info', color: 'var(--info)', format: 'number', category: 'meta' },
  { id: '__meta_cpc__', label: 'CPC Médio (Meta Ads)', icon: BarChart3, variant: 'info', color: 'var(--info)', format: 'currency', category: 'meta' },

  // Google Ads
  { id: '__google_spend__', label: 'Valor Gasto (Google Ads)', icon: Wallet, variant: 'warning', color: 'var(--warning)', format: 'currency', category: 'google' },
  { id: '__google_leads__', label: 'Leads (Google Ads)', icon: UserPlus, variant: 'warning', color: 'var(--warning)', format: 'number', category: 'google' },
  { id: '__google_cpl__', label: 'Custo por Lead (Google Ads)', icon: Target, variant: 'warning', color: 'var(--warning)', format: 'currency', category: 'google' },
  { id: '__google_purchases__', label: 'Compras (Google Ads)', icon: Receipt, variant: 'warning', color: 'var(--warning)', format: 'number', category: 'google' },
  { id: '__google_cpa__', label: 'Custo por Compra (Google Ads)', icon: Award, variant: 'warning', color: 'var(--warning)', format: 'currency', category: 'google' },
  { id: '__google_revenue__', label: 'Valor Vendido (Google Ads)', icon: DollarSign, variant: 'success', color: 'var(--success)', format: 'currency', category: 'google' },
  { id: '__google_roas__', label: 'ROAS (Google Ads)', icon: TrendingUp, variant: 'success', color: 'var(--success)', format: 'roas', category: 'google' },
  { id: '__google_clicks__', label: 'Cliques (Google Ads)', icon: MousePointerClick, variant: 'warning', color: 'var(--warning)', format: 'number', category: 'google' },
  { id: '__google_cpc__', label: 'CPC Médio (Google Ads)', icon: BarChart3, variant: 'warning', color: 'var(--warning)', format: 'currency', category: 'google' },
]

export const FULL_CARD_CATALOG: EventCatalogEntry[] = [...EVENT_CATALOG, ...EXTRA_CARD_CATALOG]
export const CATALOG_IDS = EVENT_CATALOG.map((e) => e.id)
export const ALL_CARD_IDS = FULL_CARD_CATALOG.map((e) => e.id)
export const DEFAULT_ENABLED_CARDS = [
  'PageView', 'Lead', 'InitiateCheckout', 'Purchase',
  REVENUE_CARD_ID, '__meta_spend__', '__meta_cpl__', '__meta_cpa__',
  '__google_spend__', '__google_cpa__', '__total_roas__', FAILED_CARD_ID,
]
export const DEFAULT_FUNNEL_EVENTS = ['PageView', 'ViewContent', 'AddToCart', 'InitiateCheckout', 'Purchase']

export const catalogById = new Map(FULL_CARD_CATALOG.map((e) => [e.id, e]))

export interface DashboardMetricsContext {
  sumOf: (event: string) => number
  failedTotal: number
  totalRevenue: number
  metaInsights?: AdPlatformLiveInsights | null
  googleInsights?: AdPlatformLiveInsights | null
  sourceBreakdown?: SourceBreakdown | null
}

const safeDiv = (num: number, den: number): number => (den > 0 ? num / den : 0)

export function computeCardMetricValue(id: string, ctx: DashboardMetricsContext): number {
  const metaSpend = ctx.metaInsights?.spend ?? 0
  const metaClicks = ctx.metaInsights?.clicks ?? 0
  const googleSpend = ctx.googleInsights?.spend ?? 0
  const googleClicks = ctx.googleInsights?.clicks ?? 0
  const totalSpend = metaSpend + googleSpend

  const totalLeads = ctx.sumOf('Lead') + ctx.sumOf('LeadSubmitted')
  const totalPurchases = ctx.sumOf('Purchase')
  const metaLeads = ctx.sourceBreakdown?.meta.leads ?? 0
  const metaPurchases = ctx.sourceBreakdown?.meta.purchases ?? 0
  const metaRev = ctx.sourceBreakdown?.meta.revenue ?? 0
  const googleLeads = ctx.sourceBreakdown?.google.leads ?? 0
  const googlePurchases = ctx.sourceBreakdown?.google.purchases ?? 0
  const googleRev = ctx.sourceBreakdown?.google.revenue ?? 0

  switch (id) {
    case FAILED_CARD_ID: return ctx.failedTotal
    case REVENUE_CARD_ID: return ctx.totalRevenue
    case '__total_spend__': return totalSpend
    case '__total_cpl__': return safeDiv(totalSpend, totalLeads)
    case '__total_cpa__': return safeDiv(totalSpend, totalPurchases)
    case '__total_roas__': return safeDiv(ctx.totalRevenue, totalSpend)
    case '__total_profit__': return ctx.totalRevenue - totalSpend
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
    default: return ctx.sumOf(id)
  }
}
