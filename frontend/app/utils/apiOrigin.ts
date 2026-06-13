/**
 * Extrai o origin (scheme://host:port) da apiUrl configurada.
 * Ex: "http://localhost:8080/api/v1" → "http://localhost:8080"
 */
export const getApiOrigin = (apiUrl: string): string => {
  try {
    const u = new URL(apiUrl)
    return `${u.protocol}//${u.host}`
  } catch {
    return apiUrl.replace(/\/api\/.*$/, '')
  }
}

/**
 * Monta o origin (scheme://host) preferindo o tracking_host do pixel quando definido.
 * Em produção sempre usa HTTPS para o subdomínio do cliente.
 */
export const getPixelOrigin = (apiUrl: string, trackingHost?: string): string => {
  if (trackingHost && trackingHost.trim()) {
    return `https://${trackingHost.trim()}`
  }
  return getApiOrigin(apiUrl)
}

/**
 * Monta o snippet HTML que o usuário deve colar no site.
 * Quando o pixel tem tracking_host configurado, o snippet usa o subdomínio do cliente.
 */
export const buildPixelSnippet = (
  apiUrl: string,
  pixelName: string,
  trackingHost?: string
): string => {
  const origin = getPixelOrigin(apiUrl, trackingHost)
  return [
    `<!-- TrackFind Pixel: ${pixelName} -->`,
    `<script async src="${origin}/api/v1/loader.js?pixel=${encodeURIComponent(pixelName)}"></script>`,
  ].join('\n')
}
