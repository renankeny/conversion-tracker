import { defineStore } from 'pinia'
import { computed, ref, watch } from 'vue'
import { setBrowserCookie } from '~/utils/cookies'

export type PeriodValue = 'today' | 'yesterday' | '7d' | '30d' | '60d' | '90d' | 'custom'
export type DashboardTrafficSource = 'all' | 'meta' | 'google'

const COOKIE_MAX_AGE = 60 * 60 * 24 * 365
const SAO_PAULO_OFFSET = '-03:00'

function getTodaySaoPauloYMD(now: Date = new Date()): string {
  const parts = new Intl.DateTimeFormat('en-CA', {
    timeZone: 'America/Sao_Paulo',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).formatToParts(now)
  const y = parts.find((p) => p.type === 'year')?.value || '1970'
  const m = parts.find((p) => p.type === 'month')?.value || '01'
  const d = parts.find((p) => p.type === 'day')?.value || '01'
  return `${y}-${m}-${d}`
}

function shiftYMD(ymd: string, deltaDays: number): string {
  const [y = 1970, m = 1, d = 1] = ymd.split('-').map(Number)
  const utc = new Date(Date.UTC(y, m - 1, d + deltaDays))
  const yy = utc.getUTCFullYear()
  const mm = String(utc.getUTCMonth() + 1).padStart(2, '0')
  const dd = String(utc.getUTCDate()).padStart(2, '0')
  return `${yy}-${mm}-${dd}`
}

function spStartOfDayISO(ymd: string): string {
  return new Date(`${ymd}T00:00:00.000${SAO_PAULO_OFFSET}`).toISOString()
}

function spEndOfDayISO(ymd: string): string {
  return new Date(`${ymd}T23:59:59.999${SAO_PAULO_OFFSET}`).toISOString()
}

export function computeSaoPauloRange(
  periodVal: PeriodValue,
  customFromVal = '',
  customToVal = '',
): { from: string; to: string } {
  const todayYMD = getTodaySaoPauloYMD()
  let fromYMD = todayYMD
  let toYMD = todayYMD

  switch (periodVal) {
    case 'today':
      break
    case 'yesterday':
      fromYMD = shiftYMD(todayYMD, -1)
      toYMD = fromYMD
      break
    case '7d':
      // 7 dias exatos incluindo hoje (hoje - 6 até hoje)
      fromYMD = shiftYMD(todayYMD, -6)
      break
    case '30d':
      fromYMD = shiftYMD(todayYMD, -29)
      break
    case '60d':
      fromYMD = shiftYMD(todayYMD, -59)
      break
    case '90d':
      fromYMD = shiftYMD(todayYMD, -89)
      break
    case 'custom':
      fromYMD = /^\d{4}-\d{2}-\d{2}$/.test(customFromVal) ? customFromVal : todayYMD
      toYMD = /^\d{4}-\d{2}-\d{2}$/.test(customToVal) ? customToVal : todayYMD
      break
  }

  return {
    from: spStartOfDayISO(fromYMD),
    to: spEndOfDayISO(toYMD),
  }
}

export const useDashboardStore = defineStore('dashboard', () => {
  const period = ref<PeriodValue>('today')
  const customFrom = ref('')
  const customTo = ref('')
  const selectedPixel = ref('')
  const trafficSource = ref<DashboardTrafficSource>('all')
  const privacyMode = ref(false)
  const hydrated = ref(false)

  function hydrate(init: {
    period?: PeriodValue | null
    customFrom?: string | null
    customTo?: string | null
    selectedPixel?: string | null
  }): void {
    if (init.period) period.value = init.period
    if (typeof init.customFrom === 'string') customFrom.value = init.customFrom
    if (typeof init.customTo === 'string') customTo.value = init.customTo
    if (typeof init.selectedPixel === 'string') selectedPixel.value = init.selectedPixel
    hydrated.value = true
  }

  function setSelectedPixel(val: string): void {
    selectedPixel.value = val
    setBrowserCookie('tf_dashboard_selected_pixel', val || null, COOKIE_MAX_AGE)
  }

  if (import.meta.client) {
    watch(period, (v) => setBrowserCookie('tf_dashboard_period', v, COOKIE_MAX_AGE))
    watch(customFrom, (v) => setBrowserCookie('tf_dashboard_custom_from', v || null, COOKIE_MAX_AGE))
    watch(customTo, (v) => setBrowserCookie('tf_dashboard_custom_to', v || null, COOKIE_MAX_AGE))
    watch(selectedPixel, (v) => setBrowserCookie('tf_dashboard_selected_pixel', v || null, COOKIE_MAX_AGE))
  }

  function getDateRange(): { from: string; to: string } {
    return computeSaoPauloRange(period.value, customFrom.value, customTo.value)
  }

  const granularityFor = computed<'hour' | 'day'>(() =>
    period.value === 'today' || period.value === 'yesterday' ? 'hour' : 'day',
  )

  const periodIncludesToday = computed<boolean>(() => {
    if (period.value === 'yesterday') return false
    if (period.value === 'custom') {
      if (!customTo.value) return true
      return customTo.value >= getTodaySaoPauloYMD()
    }
    return true
  })

  function togglePrivacyMode(): void {
    privacyMode.value = !privacyMode.value
    setBrowserCookie('tf_dashboard_privacy', privacyMode.value ? '1' : null, COOKIE_MAX_AGE)
  }

  return {
    period,
    customFrom,
    customTo,
    selectedPixel,
    trafficSource,
    privacyMode,
    togglePrivacyMode,
    hydrated,
    hydrate,
    setSelectedPixel,
    getDateRange,
    granularityFor,
    periodIncludesToday,
  }
})
