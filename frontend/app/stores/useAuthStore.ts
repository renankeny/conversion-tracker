import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import type { AdminUser, LoginRequest, LoginResponse } from '#shared/types'
import { sanitizeUserForStorage, setBrowserCookie, getBrowserCookie } from '~/utils/cookies'
import { errorStatusCode } from '~/utils/errorMessage'
import { useTenantStore } from '~/stores/useTenantStore'
import { useLiveEventsStore } from '~/stores/useLiveEventsStore'
import { usePixelStore } from '~/stores/usePixelStore'
import { useGoogleAdsStore } from '~/stores/useGoogleAdsStore'
import { useCustomWebhooksStore } from '~/stores/customWebhooks'
import { useSystemAlertsStore } from '~/stores/useSystemAlertsStore'

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string | null>(null)
  const user = ref<AdminUser | null>(null)
  const hydrated = ref(false)

  const isAuthenticated = computed(() => !!token.value)
  const isOwner = computed(() => {
    const role = user.value?.role
    return role === 'owner' || role === 'superadmin'
  })
  const isAdmin = computed(() => {
    const role = user.value?.role
    return role === 'owner' || role === 'superadmin' || role === 'admin'
  })
  const canManageAccounts = computed(() => {
    const role = user.value?.role
    return role === 'owner' || role === 'superadmin' || role === 'admin' || role === 'manager'
  })
  const canManagePixels = computed(() => {
    if (!user.value) return false
    if (user.value.role === 'employee') return user.value.permissions?.edit_pixels !== false
    return true
  })
  const hasOpenAiKey = computed(() => {
    const prefs = user.value?.preferences
    return !!(prefs?.has_openai_key || prefs?.openai_api_key)
  })

  function hydrate(initialToken?: string | null, initialUser?: AdminUser | null): void {
    let resolvedToken = initialToken || null
    let resolvedUser = initialUser || null

    if (!resolvedToken && import.meta.client) {
      resolvedToken = getBrowserCookie('auth_token')
      if (!resolvedToken) {
        try {
          const local = localStorage.getItem('auth_token')
          if (local && local.trim()) resolvedToken = local.trim()
        } catch {}
      }
    }

    if (!resolvedUser && import.meta.client) {
      try {
        const localUser = localStorage.getItem('auth_user')
        if (localUser) resolvedUser = JSON.parse(localUser)
      } catch {}
    }

    token.value = resolvedToken
    user.value = sanitizeUserForStorage(resolvedUser)
    hydrated.value = true
  }

  function setUser(nextUser: AdminUser | null, maxAge = 60 * 60 * 24 * 30): void {
    const clean = sanitizeUserForStorage(nextUser)
    user.value = clean
    setBrowserCookie('auth_user', clean ? JSON.stringify(clean) : null, maxAge)
    if (import.meta.client) {
      try {
        if (clean) localStorage.setItem('auth_user', JSON.stringify(clean))
        else localStorage.removeItem('auth_user')
      } catch {}
    }
  }

  function clearSession(): void {
    token.value = null
    user.value = null
    setBrowserCookie('auth_token', null)
    setBrowserCookie('auth_user', null)
    setBrowserCookie('active_account_id', null)
    if (import.meta.client) {
      try {
        localStorage.removeItem('auth_token')
        localStorage.removeItem('auth_user')
        localStorage.removeItem('trackfind_selected_pixel')
      } catch {}
    }
    useLiveEventsStore().disconnect()
    useTenantStore().reset()
    usePixelStore().invalidate()
    useGoogleAdsStore().invalidate()
    useCustomWebhooksStore().invalidate()
    useSystemAlertsStore().invalidate()
  }

  async function login(credentials: LoginRequest): Promise<LoginResponse> {
    const api = useAPI()
    const response = await api.post<LoginResponse>('/admin/login', credentials)
    const maxAge = credentials.remember_me ? 60 * 60 * 24 * 30 : 60 * 60 * 24
    const cleanUser = sanitizeUserForStorage(response.user)
    const accId = cleanUser?.account_id ? String(cleanUser.account_id) : '1'

    token.value = response.access_token
    user.value = cleanUser
    useTenantStore().setActiveAccountId(accId, maxAge)

    setBrowserCookie('auth_token', response.access_token, maxAge)
    setBrowserCookie('auth_user', cleanUser ? JSON.stringify(cleanUser) : null, maxAge)

    if (import.meta.client) {
      try {
        localStorage.setItem('auth_token', response.access_token)
        if (cleanUser) localStorage.setItem('auth_user', JSON.stringify(cleanUser))
      } catch {}
      if (credentials.remember_me) {
        localStorage.setItem('remember_me', 'true')
        localStorage.setItem('remember_email', credentials.email)
      } else {
        localStorage.removeItem('remember_me')
        localStorage.removeItem('remember_email')
      }
    }

    return { ...response, user: cleanUser || response.user }
  }

  async function logout(): Promise<void> {
    const api = useAPI()
    try {
      if (token.value) {
        await api.post('/admin/logout', {}).catch(() => {})
      }
    } finally {
      clearSession()
      if (import.meta.client) {
        localStorage.removeItem('remember_me')
        await navigateTo('/login', { replace: true })
      }
    }
  }

  async function fetchMe(): Promise<boolean> {
    if (!token.value) return false
    const api = useAPI()
    try {
      const me = await api.get<AdminUser>('/admin/me')
      setUser(me)
      return true
    } catch (err: unknown) {
      if (errorStatusCode(err) === 401) {
        clearSession()
      }
      return false
    }
  }

  async function updatePreferences(preferences: Record<string, unknown>): Promise<void> {
    if (!token.value || !user.value) return
    const api = useAPI()
    const response = await api.put<{ ok: boolean; preferences: Record<string, unknown> }>(
      '/admin/me/preferences',
      preferences,
    )
    setUser({ ...user.value, preferences: response.preferences })
  }

  return {
    token,
    user,
    hydrated,
    isAuthenticated,
    isOwner,
    isAdmin,
    canManageAccounts,
    canManagePixels,
    hasOpenAiKey,
    hydrate,
    setUser,
    clearSession,
    login,
    logout,
    fetchMe,
    updatePreferences,
  }
})
