import type { AdminUser } from '#shared/types'

export function setBrowserCookie(
  name: string,
  value: string | null | undefined,
  maxAgeSeconds = 60 * 60 * 24 * 30,
): void {
  if (!import.meta.client) return
  if (value === null || value === undefined || value === '') {
    document.cookie = `${name}=; Path=/; Max-Age=-1; SameSite=Lax`
    return
  }
  document.cookie = `${name}=${encodeURIComponent(value)}; Path=/; Max-Age=${maxAgeSeconds}; SameSite=Lax`
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

