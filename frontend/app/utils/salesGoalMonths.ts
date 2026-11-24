import type { PeriodValue } from '~/stores/useDashboardStore'

const PT_MONTH_NAMES = [
  'Janeiro', 'Fevereiro', 'Março', 'Abril', 'Maio', 'Junho',
  'Julho', 'Agosto', 'Setembro', 'Outubro', 'Novembro', 'Dezembro',
]

export function getCurrentSaoPauloYM(now: Date = new Date()): string {
  const parts = new Intl.DateTimeFormat('en-CA', {
    timeZone: 'America/Sao_Paulo',
    year: 'numeric',
    month: '2-digit',
  }).formatToParts(now)
  const y = parts.find((p) => p.type === 'year')?.value || '1970'
  const m = parts.find((p) => p.type === 'month')?.value || '01'
  return `${y}-${m}`
}

export function shiftYearMonth(ym: string, deltaMonths: number): string {
  const [y = 1970, m = 1] = ym.split('-').map(Number)
  const d = new Date(Date.UTC(y, m - 1 + deltaMonths, 1))
  const yy = d.getUTCFullYear()
  const mm = String(d.getUTCMonth() + 1).padStart(2, '0')
  return `${yy}-${mm}`
}

export function resolveDashboardGoalMonth(
  period: PeriodValue,
  customFrom = '',
  customTo = '',
): string {
  const curYM = getCurrentSaoPauloYM()
  if (period === 'last_month') {
    return shiftYearMonth(curYM, -1)
  }
  if (period === 'custom') {
    const refDate = /^\d{4}-\d{2}-\d{2}$/.test(customTo)
      ? customTo
      : (/^\d{4}-\d{2}-\d{2}$/.test(customFrom) ? customFrom : '')
    if (refDate) {
      return refDate.slice(0, 7)
    }
  }
  return curYM
}

export function formatGoalMonthLabel(ym: string, includeRelativeTag = true): string {
  const [y = 0, m = 0] = ym.split('-').map(Number)
  const name = PT_MONTH_NAMES[m - 1]
  if (!name || !y) return ym
  const base = `${name}/${y}`
  if (!includeRelativeTag) return base
  const curYM = getCurrentSaoPauloYM()
  if (ym === curYM) return `${base} (Mês atual)`
  if (ym === shiftYearMonth(curYM, -1)) return `${base} (Mês passado)`
  if (ym === shiftYearMonth(curYM, 1)) return `${base} (Próximo mês)`
  return base
}

export function resolveSalesGoalTargetForMonth(
  defaultTarget: number | undefined,
  months: Record<string, number> | undefined,
  targetYM: string,
): number {
  const map = months || {}
  const exact = Number(map[targetYM])
  if (Number.isFinite(exact) && exact > 0) return exact

  const validKeys = Object.keys(map)
    .filter((k) => /^\d{4}-\d{2}$/.test(k) && Number(map[k]) > 0)
    .sort()

  const priorOrEqual = validKeys.filter((k) => k <= targetYM)
  if (priorOrEqual.length > 0) {
    const bestKey = priorOrEqual[priorOrEqual.length - 1]!
    return Number(map[bestKey]) || 0
  }

  const fallback = Number(defaultTarget) || 0
  if (fallback > 0) return fallback

  if (validKeys.length > 0) {
    return Number(map[validKeys[0]!]) || 0
  }
  return 0
}

export function buildEditableGoalMonthOptions(existingMonths?: Record<string, number>): Array<{ value: string; label: string }> {
  const curYM = getCurrentSaoPauloYM()
  const set = new Set<string>([
    shiftYearMonth(curYM, 1),
    curYM,
    shiftYearMonth(curYM, -1),
    shiftYearMonth(curYM, -2),
    shiftYearMonth(curYM, -3),
    shiftYearMonth(curYM, -4),
    shiftYearMonth(curYM, -5),
  ])
  for (const k of Object.keys(existingMonths || {})) {
    if (/^\d{4}-\d{2}$/.test(k)) set.add(k)
  }
  return Array.from(set)
    .sort((a, b) => b.localeCompare(a))
    .map((ym) => ({
      value: ym,
      label: formatGoalMonthLabel(ym, true),
    }))
}
