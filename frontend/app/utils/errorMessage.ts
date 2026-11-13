export function errorMessage(err: unknown, fallback = 'Ocorreu um erro inesperado.'): string {
  if (!err) return fallback
  if (typeof err === 'string') return err
  if (typeof err === 'object') {
    const rec = err as { data?: { error?: string; message?: string }; message?: string }
    if (rec.data?.error) return rec.data.error
    if (rec.data?.message) return rec.data.message
    if (rec.message) return rec.message
  }
  return fallback
}

export function errorStatusCode(err: unknown): number | undefined {
  if (!err || typeof err !== 'object') return undefined
  const rec = err as { statusCode?: number; status?: number; response?: { status?: number } }
  return rec.statusCode ?? rec.status ?? rec.response?.status
}

