/**
 * Extrai o origin (scheme://host:port) da apiUrl configurada.
 */
export const getApiOrigin = (apiUrl: string): string => {
  const trimmed = String(apiUrl || '').trim()
  try {
    const u = new URL(trimmed)
    return `${u.protocol}//${u.host}`
  } catch {
    if (typeof window !== 'undefined' && window.location?.origin) {
      return window.location.origin
    }
    return trimmed.replace(/\/api\/.*$/, '')
  }
}

/**
 * Resolve a URL base pública da API garantindo protocolo e domínio completos
 * mesmo quando NUXT_PUBLIC_API_URL é configurada como caminho relativo (/api/v1).
 */
export const resolvePublicApiBase = (apiUrl: string): string => {
  const raw = String(apiUrl || '').trim() || '/api/v1'
  const clean = raw.replace(/\/+$/, '')
  if (/^https?:\/\//i.test(clean)) {
    return clean
  }
  const path = clean.startsWith('/') ? clean : `/${clean}`
  if (typeof window !== 'undefined' && window.location?.origin) {
    return `${window.location.origin}${path}`
  }
  return `http://localhost:8080${path}`
}

/**
 * Monta o snippet HTML que o usuário deve colar no site.
 * Usa o public_token quando disponível para não expor o nome interno.
 */
export const buildPixelSnippet = (
  apiUrl: string,
  pixelName: string,
  publicToken?: string,
  brandName = 'Sua Plataforma',
): string => {
  const origin = getApiOrigin(apiUrl)
  const identifier = publicToken || pixelName
  return [
    `<!-- ${brandName} Pixel: ${pixelName} -->`,
    `<script async src="${origin}/api/v1/loader.js?pixel=${encodeURIComponent(identifier)}"></script>`,
  ].join('\n')
}
