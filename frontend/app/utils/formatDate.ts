export const SP_TIMEZONE = 'America/Sao_Paulo'

function toValidDate(input?: string | number | Date | null): Date | null {
  if (input === null || input === undefined || input === '') return null
  const d = input instanceof Date ? input : new Date(input)
  return Number.isNaN(d.getTime()) ? null : d
}

export function formatDateSP(
  input?: string | number | Date | null,
  options?: Intl.DateTimeFormatOptions,
): string {
  const d = toValidDate(input)
  if (!d) return ''
  return d.toLocaleDateString('pt-BR', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    ...options,
    timeZone: SP_TIMEZONE,
  })
}

function resolveDateTimeExtra(
  arg: boolean | Intl.DateTimeFormatOptions,
): Intl.DateTimeFormatOptions {
  if (typeof arg === 'boolean') {
    return arg ? { second: '2-digit' } : {}
  }
  return arg
}

export function formatTimeSP(
  input?: string | number | Date | null,
  withSecondsOrOptions: boolean | Intl.DateTimeFormatOptions = false,
): string {
  const d = toValidDate(input)
  if (!d) return ''
  return d.toLocaleTimeString('pt-BR', {
    hour: '2-digit',
    minute: '2-digit',
    ...resolveDateTimeExtra(withSecondsOrOptions),
    timeZone: SP_TIMEZONE,
  })
}

export function formatDateTimeSP(
  input?: string | number | Date | null,
  withSecondsOrOptions: boolean | Intl.DateTimeFormatOptions = false,
): string {
  const d = toValidDate(input)
  if (!d) return ''
  return d.toLocaleString('pt-BR', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    ...resolveDateTimeExtra(withSecondsOrOptions),
    timeZone: SP_TIMEZONE,
  })
}

export function formatShortDateTimeSP(
  input?: string | number | Date | null,
  withSecondsOrOptions: boolean | Intl.DateTimeFormatOptions = false,
): string {
  const d = toValidDate(input)
  if (!d) return ''
  return d.toLocaleString('pt-BR', {
    day: '2-digit',
    month: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    ...resolveDateTimeExtra(withSecondsOrOptions),
    timeZone: SP_TIMEZONE,
  })
}

export function dateKeySP(input?: string | number | Date | null): string {
  const d = toValidDate(input)
  if (!d) return ''
  return d.toLocaleDateString('en-CA', { timeZone: SP_TIMEZONE })
}
