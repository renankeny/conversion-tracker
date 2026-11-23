import { setResponseHeader } from 'h3'

interface RemoteBranding {
  platform_name?: string
  tagline?: string
  favicon_url?: string
  theme_config?: {
    dark?: {
      brand?: string
      surface_1?: string
    }
  }
}

export default defineEventHandler(async (event) => {
  const config = useRuntimeConfig(event)
  const internalOrigin = String(config.internalApiUrl || '')
    .trim()
    .replace(/\/+$/, '')
    .replace(/\/api\/v1$/, '')
  const publicOrigin = String(config.public.apiUrl || '')
    .trim()
    .replace(/\/+$/, '')
    .replace(/\/api\/v1$/, '')
  const apiOrigin = internalOrigin || publicOrigin || 'http://127.0.0.1:8080'

  let branding: RemoteBranding = {}
  try {
    const res = await fetch(`${apiOrigin}/api/v1/branding`)
    if (res.ok) {
      branding = (await res.json()) as RemoteBranding
    }
  } catch {
    // Fallback silencioso para valores padrão da plataforma
  }

  const name = (branding.platform_name || String(config.public.appName || 'Painel')).trim()
  const shortName = name.length > 16 ? name.slice(0, 16).trim() : name
  const themeColor = branding.theme_config?.dark?.brand || '#1C87C1'
  const bgColor = branding.theme_config?.dark?.surface_1 || '#121214'

  const icons: Array<{ src: string; sizes: string; type: string; purpose?: string }> = [
    {
      src: '/pwa-icon.svg',
      sizes: '192x192 512x512',
      type: 'image/svg+xml',
      purpose: 'any maskable',
    },
  ]
  if (branding.favicon_url && branding.favicon_url.startsWith('https://')) {
    icons.unshift({
      src: branding.favicon_url,
      sizes: '192x192',
      type: 'image/png',
    })
  }

  setResponseHeader(event, 'content-type', 'application/manifest+json; charset=utf-8')
  setResponseHeader(event, 'cache-control', 'no-cache')

  return {
    name,
    short_name: shortName,
    description: branding.tagline || 'Plataforma de rastreamento e atribuição de conversões',
    start_url: '/',
    scope: '/',
    display: 'standalone',
    orientation: 'portrait-primary',
    background_color: bgColor,
    theme_color: themeColor,
    icons,
  }
})
