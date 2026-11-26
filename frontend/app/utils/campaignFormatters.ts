export function formatCampaignMoney(value: number, currency = 'BRL'): string {
  const num = Number.isFinite(value) ? value : 0
  const cur = (currency || 'BRL').toUpperCase()
  try {
    return new Intl.NumberFormat('pt-BR', {
      style: 'currency',
      currency: cur,
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    }).format(num)
  } catch {
    return `${cur} ${num.toFixed(2)}`
  }
}

export function formatCampaignPct(value: number): string {
  const num = Number.isFinite(value) ? value : 0
  return `${num.toFixed(1).replace('.', ',')}%`
}

export function formatRoas(value: number): string {
  const num = Number.isFinite(value) ? value : 0
  return `${num.toFixed(2).replace('.', ',')}x`
}

export function formatCompactInt(value: number): string {
  const num = Math.round(Number.isFinite(value) ? value : 0)
  return new Intl.NumberFormat('pt-BR').format(num)
}

export function formatSPDateISO(date: Date): string {
  const parts = new Intl.DateTimeFormat('en-CA', {
    timeZone: 'America/Sao_Paulo',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).formatToParts(date)
  const y = parts.find((p) => p.type === 'year')?.value || '2026'
  const m = parts.find((p) => p.type === 'month')?.value || '01'
  const d = parts.find((p) => p.type === 'day')?.value || '01'
  return `${y}-${m}-${d}`
}

export function resolvePeriodDates(preset: string, customSince: string, customUntil: string): { since: string; until: string } {
  const now = new Date()
  const today = formatSPDateISO(now)
  const shift = (days: number) => {
    const dt = new Date(now.getTime() + days * 86400000)
    return formatSPDateISO(dt)
  }
  switch (preset) {
    case 'today':
      return { since: today, until: today }
    case 'yesterday': {
      const y = shift(-1)
      return { since: y, until: y }
    }
    case '30d':
      return { since: shift(-29), until: today }
    case 'month':
      return { since: `${today.slice(0, 7)}-01`, until: today }
    case 'custom':
      return {
        since: customSince || shift(-6),
        until: customUntil || today,
      }
    default:
      return { since: shift(-6), until: today }
  }
}
