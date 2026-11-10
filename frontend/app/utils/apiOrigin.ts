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
 * Monta o snippet HTML que o usuário deve colar no site.
 * Usa o public_token quando disponível (mais seguro — não expõe o nome interno).
 */
export const buildPixelSnippet = (
  apiUrl: string,
  pixelName: string,
  publicToken?: string,
  brandName = 'Sua Plataforma'
): string => {
  const origin = getApiOrigin(apiUrl)
  const identifier = publicToken || pixelName
  return [
    `<!-- ${brandName} Pixel: ${pixelName} -->`,
    `<script async src="${origin}/api/v1/loader.js?pixel=${encodeURIComponent(identifier)}"></script>`,
  ].join('\n')
}

