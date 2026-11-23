import { setResponseHeader } from 'h3'

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

  let name = String(config.public.appName || 'T')
  let brand = '#1C87C1'
  let bg = '#121214'

  try {
    const res = await fetch(`${apiOrigin}/api/v1/branding`)
    if (res.ok) {
      const b = await res.json()
      if (b?.platform_name) name = String(b.platform_name)
      if (/^#[0-9a-fA-F]{6}$/.test(b?.theme_config?.dark?.brand || '')) brand = b.theme_config.dark.brand
      if (/^#[0-9a-fA-F]{6}$/.test(b?.theme_config?.dark?.surface_1 || '')) bg = b.theme_config.dark.surface_1
    }
  } catch {
    // Mantém fallback Whitelabel
  }

  const initial = (name.trim().charAt(0) || 'T').toUpperCase().replace(/[^A-Z0-9]/g, 'T')
  setResponseHeader(event, 'content-type', 'image/svg+xml; charset=utf-8')
  setResponseHeader(event, 'cache-control', 'public, max-age=300')

  return `<svg xmlns="http://www.w3.org/2000/svg" width="512" height="512" viewBox="0 0 512 512">
  <rect width="512" height="512" rx="112" fill="${bg}"/>
  <rect x="36" y="36" width="440" height="440" rx="92" fill="${brand}" fill-opacity="0.16" stroke="${brand}" stroke-width="12"/>
  <text x="256" y="328" text-anchor="middle" font-family="system-ui,-apple-system,sans-serif" font-size="224" font-weight="700" fill="${brand}">${initial}</text>
</svg>`
})
