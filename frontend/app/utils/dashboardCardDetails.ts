import {
  REVENUE_CARD_ID,
  catalogById,
  resolvePanelLeads,
  resolvePanelPurchases,
  type DashboardMetricsContext,
} from '~/utils/dashboardEventCatalog'

export interface CardComparisonChip {
  label: string
  value: string
  tone: 'meta' | 'google' | 'panel' | 'muted'
}

export interface CardDeliveryAudit {
  sent: number
  notSent: number
  sentLabel: string
  notSentLabel: string
}

export interface CardRichDetails {
  changePercent: number | null
  changeInverse: boolean
  previousLabel?: string
  blurred: boolean
  comparisons: CardComparisonChip[]
  delivery?: CardDeliveryAudit
}

const fmtInt = (n: number) => new Intl.NumberFormat('pt-BR').format(Math.round(n || 0))
export function formatMoney(n: number, currency = 'BRL'): string {
  const code = (currency || 'BRL').trim().toUpperCase()
  try {
    return new Intl.NumberFormat('pt-BR', { style: 'currency', currency: code }).format(n || 0)
  } catch {
    return new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(n || 0)
  }
}
const fmtRoas = (n: number) => `${new Intl.NumberFormat('pt-BR', { minimumFractionDigits: 2, maximumFractionDigits: 2 }).format(n || 0)}x`
const fmtPct = (n: number) => `${new Intl.NumberFormat('pt-BR', { minimumFractionDigits: 1, maximumFractionDigits: 1 }).format((n || 0) * 100)}%`
const safeDiv = (a: number, b: number) => (b > 0 ? a / b : 0)

function pctDelta(curr: number, prev: number | null): number | null {
  if (prev === null || prev <= 0) return null
  return ((curr - prev) / prev) * 100
}

function resolvePreviousSpendAndClicks(ctx: DashboardMetricsContext): { spend: number; clicks: number; hasAdsPrev: boolean } {
  const mConnected = Boolean(ctx.metaInsights?.connected)
  const gConnected = Boolean(ctx.googleInsights?.connected)
  const mPrevSpend = ctx.metaInsights?.prev_spend ?? 0
  const mPrevClicks = ctx.metaInsights?.prev_clicks ?? 0
  const gPrevSpend = ctx.googleInsights?.prev_spend ?? 0
  const gPrevClicks = ctx.googleInsights?.prev_clicks ?? 0

  if (ctx.trafficSource === 'meta') return { spend: mPrevSpend, clicks: mPrevClicks, hasAdsPrev: mConnected }
  if (ctx.trafficSource === 'google') return { spend: gPrevSpend, clicks: gPrevClicks, hasAdsPrev: gConnected }
  return { spend: mPrevSpend + gPrevSpend, clicks: mPrevClicks + gPrevClicks, hasAdsPrev: mConnected || gConnected }
}

function computePreviousMetric(id: string, ctx: DashboardMetricsContext): { val: number | null; formatted?: string } {
  const prev = ctx.previousPeriod
  const { spend: pSpend, clicks: pClicks, hasAdsPrev } = resolvePreviousSpendAndClicks(ctx)
  const money = (n: number) => formatMoney(n, ctx.currency)
  if (!prev) return { val: null }

  switch (id) {
    case 'PageView': return { val: prev.page_views, formatted: fmtInt(prev.page_views) }
    case '__ad_messages__': return { val: prev.ad_messages, formatted: fmtInt(prev.ad_messages) }
    case 'Lead': return { val: prev.leads, formatted: fmtInt(prev.leads) }
    case 'Purchase': return { val: prev.purchases, formatted: fmtInt(prev.purchases) }
    case REVENUE_CARD_ID: return { val: prev.revenue, formatted: money(prev.revenue) }
    case '__total_spend__': return hasAdsPrev ? { val: pSpend, formatted: money(pSpend) } : { val: null }
    case '__ad_clicks__': return hasAdsPrev ? { val: pClicks, formatted: fmtInt(pClicks) } : { val: null }
    case '__avg_cpc__': {
      if (!hasAdsPrev || pClicks <= 0) return { val: null }
      const v = safeDiv(pSpend, pClicks)
      return { val: v, formatted: money(v) }
    }
    case '__total_cpl__': {
      if (!hasAdsPrev || prev.leads <= 0) return { val: null }
      const v = safeDiv(pSpend, prev.leads)
      return { val: v, formatted: money(v) }
    }
    case '__total_cpa__': {
      if (!hasAdsPrev || prev.purchases <= 0) return { val: null }
      const v = safeDiv(pSpend, prev.purchases)
      return { val: v, formatted: money(v) }
    }
    case '__total_roas__': {
      if (!hasAdsPrev || pSpend <= 0) return { val: null }
      const v = safeDiv(prev.revenue, pSpend)
      return { val: v, formatted: fmtRoas(v) }
    }
    case '__avg_ticket__': {
      if (prev.purchases <= 0) return { val: null }
      const v = safeDiv(prev.revenue, prev.purchases)
      return { val: v, formatted: money(v) }
    }
    case '__conv_rate__': {
      if (prev.leads <= 0) return { val: null }
      const v = safeDiv(prev.purchases, prev.leads)
      return { val: v, formatted: fmtPct(v) }
    }
    default:
      return { val: null }
  }
}

export function computeCardRichDetails(id: string, currentVal: number, ctx: DashboardMetricsContext): CardRichDetails {
  const entry = catalogById.get(id)
  const blurred = Boolean(ctx.privacyMode && entry?.privateValue)
  const changeInverse = Boolean(entry?.changeInverse)
  const money = (n: number) => formatMoney(n, ctx.currency)
  const { val: prevVal, formatted: prevFmt } = computePreviousMetric(id, ctx)
  const changePercent = pctDelta(currentVal, prevVal)

  const src = ctx.trafficSource || 'all'
  const mConn = Boolean(ctx.metaInsights?.connected)
  const gConn = Boolean(ctx.googleInsights?.connected)
  const mSpend = ctx.metaInsights?.spend ?? 0
  const gSpend = ctx.googleInsights?.spend ?? 0
  const mClicks = ctx.metaInsights?.clicks ?? 0
  const gClicks = ctx.googleInsights?.clicks ?? 0
  const mMsgs = ctx.metaInsights?.messages ?? 0
  const mMgrLeads = ctx.metaInsights?.leads ?? 0
  const gMgrLeads = ctx.googleInsights?.leads ?? 0
  const mMgrPur = ctx.metaInsights?.purchases ?? 0
  const gMgrPur = ctx.googleInsights?.purchases ?? 0
  const mMgrRev = ctx.metaInsights?.revenue ?? 0
  const gMgrRev = ctx.googleInsights?.revenue ?? 0

  const panelLeads = resolvePanelLeads(ctx)
  const panelPurchases = resolvePanelPurchases(ctx)
  const audit = ctx.panelAudit

  const comparisons: CardComparisonChip[] = []
  let delivery: CardDeliveryAudit | undefined

  if (id === '__ad_messages__') {
    const c = audit?.ad_messages
    if (src === 'meta') {
      comparisons.push({ label: 'Meta', value: fmtInt(mMsgs), tone: 'meta' })
      comparisons.push({ label: 'Painel', value: fmtInt(c?.meta_total ?? currentVal), tone: 'panel' })
    } else if (src === 'google') {
      comparisons.push({ label: 'Google (Cliques)', value: fmtInt(gClicks), tone: 'google' })
      comparisons.push({ label: 'Painel', value: fmtInt(c?.google_total ?? currentVal), tone: 'panel' })
    } else {
      if (mConn || mMsgs > 0 || (c?.meta_total ?? 0) > 0) comparisons.push({ label: 'Meta', value: fmtInt(mMsgs), tone: 'meta' })
      if ((c?.google_total ?? 0) > 0) comparisons.push({ label: 'Google (WPP)', value: fmtInt(c?.google_total ?? 0), tone: 'google' })
      comparisons.push({ label: 'Painel', value: fmtInt(c?.total ?? currentVal), tone: 'panel' })
    }
    if (c) {
      delivery = { sent: c.sent, notSent: c.not_sent, sentLabel: 'Enviados', notSentLabel: 'Não enviados' }
    }
  } else if (id === 'Lead' || id === '__meta_leads__' || id === '__google_leads__') {
    const c = audit?.leads
    if (id === '__meta_leads__' || src === 'meta') {
      comparisons.push({ label: 'Meta', value: fmtInt(mMgrLeads), tone: 'meta' })
      comparisons.push({ label: 'Painel', value: fmtInt(c?.meta_total ?? panelLeads), tone: 'panel' })
      if (c) delivery = { sent: c.meta_sent, notSent: c.meta_not_sent, sentLabel: 'Enviados', notSentLabel: 'Não enviados' }
    } else if (id === '__google_leads__' || src === 'google') {
      comparisons.push({ label: 'Google', value: fmtInt(gMgrLeads), tone: 'google' })
      comparisons.push({ label: 'Painel', value: fmtInt(c?.google_total ?? panelLeads), tone: 'panel' })
      if (c) delivery = { sent: c.google_sent, notSent: c.google_not_sent, sentLabel: 'Enviados', notSentLabel: 'Não enviados' }
    } else {
      if (mConn || mMgrLeads > 0 || (c?.meta_total ?? 0) > 0) comparisons.push({ label: 'Meta', value: fmtInt(mMgrLeads), tone: 'meta' })
      if (gConn || gMgrLeads > 0 || (c?.google_total ?? 0) > 0) comparisons.push({ label: 'Google', value: fmtInt(gMgrLeads), tone: 'google' })
      comparisons.push({ label: 'Painel', value: fmtInt(panelLeads), tone: 'panel' })
      if (c) delivery = { sent: c.sent, notSent: c.not_sent, sentLabel: 'Enviados', notSentLabel: 'Não enviados' }
    }
  } else if (id === 'Purchase' || id === '__meta_purchases__' || id === '__google_purchases__') {
    const c = audit?.purchases
    if (id === '__meta_purchases__' || src === 'meta') {
      comparisons.push({ label: 'Meta', value: fmtInt(mMgrPur), tone: 'meta' })
      comparisons.push({ label: 'Painel', value: fmtInt(c?.meta_total ?? panelPurchases), tone: 'panel' })
      if (c) delivery = { sent: c.meta_sent, notSent: c.meta_not_sent, sentLabel: 'Enviadas', notSentLabel: 'Não enviadas' }
    } else if (id === '__google_purchases__' || src === 'google') {
      comparisons.push({ label: 'Google', value: fmtInt(gMgrPur), tone: 'google' })
      comparisons.push({ label: 'Painel', value: fmtInt(c?.google_total ?? panelPurchases), tone: 'panel' })
      if (c) delivery = { sent: c.google_sent, notSent: c.google_not_sent, sentLabel: 'Enviadas', notSentLabel: 'Não enviadas' }
    } else {
      if (mConn || mMgrPur > 0 || (c?.meta_total ?? 0) > 0) comparisons.push({ label: 'Meta', value: fmtInt(mMgrPur), tone: 'meta' })
      if (gConn || gMgrPur > 0 || (c?.google_total ?? 0) > 0) comparisons.push({ label: 'Google', value: fmtInt(gMgrPur), tone: 'google' })
      comparisons.push({ label: 'Painel', value: fmtInt(panelPurchases), tone: 'panel' })
      if (c) delivery = { sent: c.sent, notSent: c.not_sent, sentLabel: 'Enviadas', notSentLabel: 'Não enviadas' }
    }
  } else if (id === REVENUE_CARD_ID && !blurred) {
    const sbMeta = ctx.sourceBreakdown?.meta.revenue ?? 0
    const sbGoogle = ctx.sourceBreakdown?.google.revenue ?? 0
    if (src === 'meta') {
      comparisons.push({ label: 'Meta', value: money(mMgrRev), tone: 'meta' })
      comparisons.push({ label: 'Painel', value: money(ctx.totalRevenue), tone: 'panel' })
    } else if (src === 'google') {
      comparisons.push({ label: 'Google', value: money(gMgrRev), tone: 'google' })
      comparisons.push({ label: 'Painel', value: money(ctx.totalRevenue), tone: 'panel' })
    } else {
      if (mConn || sbMeta > 0 || mMgrRev > 0) comparisons.push({ label: 'Meta', value: money(sbMeta), tone: 'meta' })
      if (gConn || sbGoogle > 0 || gMgrRev > 0) comparisons.push({ label: 'Google', value: money(sbGoogle), tone: 'google' })
    }
  } else if (id === '__total_spend__' && src === 'all' && !blurred) {
    if (mConn || mSpend > 0) comparisons.push({ label: 'Meta', value: money(mSpend), tone: 'meta' })
    if (gConn || gSpend > 0) comparisons.push({ label: 'Google', value: money(gSpend), tone: 'google' })
  } else if (id === '__ad_clicks__' && src === 'all') {
    if (mConn || mClicks > 0) comparisons.push({ label: 'Meta', value: fmtInt(mClicks), tone: 'meta' })
    if (gConn || gClicks > 0) comparisons.push({ label: 'Google', value: fmtInt(gClicks), tone: 'google' })
  } else if (id === '__total_roas__' && src === 'all') {
    const mRev = ctx.sourceBreakdown?.meta.revenue ?? 0
    const gRev = ctx.sourceBreakdown?.google.revenue ?? 0
    if (mSpend > 0) comparisons.push({ label: 'Meta', value: fmtRoas(safeDiv(mRev, mSpend)), tone: 'meta' })
    if (gSpend > 0) comparisons.push({ label: 'Google', value: fmtRoas(safeDiv(gRev, gSpend)), tone: 'google' })
  } else if (id === '__conv_rate__') {
    comparisons.push({ label: 'Base', value: `${fmtInt(panelPurchases)} de ${fmtInt(panelLeads)} leads`, tone: 'muted' })
  }

  return {
    changePercent,
    changeInverse,
    previousLabel: blurred ? undefined : prevFmt,
    blurred,
    comparisons,
    delivery,
  }
}

