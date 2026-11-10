import type { AdminUser } from '#shared/types'

export function setBrowserCookie(
  name: string,
  value: string | null | undefined,
  maxAgeSeconds = 60 * 60 * 24 * 30,
): void {
  if (!import.meta.client) return
  const isHttps = typeof window !== 'undefined' && window.location.protocol === 'https:'
  const secureSuffix = isHttps ? '; Secure' : ''
  if (value === null || value === undefined || value === '') {
    document.cookie = `${name}=; Path=/; Max-Age=-1; SameSite=Lax${secureSuffix}`
    return
  }
  document.cookie = `${name}=${encodeURIComponent(value)}; Path=/; Max-Age=${maxAgeSeconds}; SameSite=Lax${secureSuffix}`
}

export function getBrowserCookie(name: string): string | null {
  if (!import.meta.client || typeof document === 'undefined') return null
  const match = document.cookie.match(new RegExp(`(?:^|;\\s*)${name}=([^;]+)`))
  if (!match || !match[1]) return null
  try {
    return decodeURIComponent(match[1]).trim() || null
  } catch {
    return match[1].trim() || null
  }
}

export function sanitizeUserForStorage(user: AdminUser | null | undefined): AdminUser | null {
  if (!user) return null
  const prefs = user.preferences ? { ...user.preferences } : undefined
  if (prefs && typeof prefs.openai_api_key === 'string') {
    if (prefs.openai_api_key.trim()) {
      prefs.has_openai_key = true
    }
    delete prefs.openai_api_key
  }
  return {
    ...user,
    preferences: prefs,
  }
}

