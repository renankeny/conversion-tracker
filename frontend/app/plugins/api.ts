import { watch } from 'vue'
import type { $Fetch } from 'ofetch'
import type { AdminUser } from '#shared/types'
import type { ServiceNoticeState } from '~/composables/useSetup'
import { useAuthStore } from '~/stores/useAuthStore'
import { useDashboardStore, type DashboardTrafficSource, type PeriodValue } from '~/stores/useDashboardStore'
import { useTenantStore } from '~/stores/useTenantStore'
import { useThemeStore } from '~/stores/useThemeStore'
import { setBrowserCookie, getBrowserCookie } from '~/utils/cookies'

function buildNormalizedError(status: number | undefined, data: unknown, fallback: string): Error & { statusCode?: number; data?: unknown } {
  const errField = data && typeof data === 'object' ? (data as Record<string, unknown>).error : undefined
  const message =
    typeof data === 'string' && data.trim()
      ? data
      : (typeof errField === 'string' && errField ? errField : (fallback || 'Erro desconhecido'))
  const err = new Error(message) as Error & { statusCode?: number; data?: unknown }
  err.statusCode = status
  err.data = data
  return err
}

export default defineNuxtPlugin({
  name: 'api',
  dependsOn: ['pinia'],
  setup() {
    const config = useRuntimeConfig()
    const apiBase = config.public.apiUrl as string
    const serviceNotice = useState<ServiceNoticeState | null>('platform_service_notice', () => null)

    const authStore = useAuthStore()
    const tenantStore = useTenantStore()
    const themeStore = useThemeStore()
    const dashboardStore = useDashboardStore()

    const authTokenCookie = useCookie<string | null>('auth_token')
    const authUserCookie = useCookie<AdminUser | null>('auth_user')
    const activeAccountCookie = useCookie<string | null>('active_account_id')
    const tfThemeCookie = useCookie<string | null>('tf_theme', { default: () => null, sameSite: 'lax' })
    const legacyThemeCookie = useCookie<string | null>('theme', { default: () => 'dark', sameSite: 'lax' })
    const dashPeriodCookie = useCookie<PeriodValue>('tf_dashboard_period', { default: () => 'today' })
    const dashFromCookie = useCookie<string>('tf_dashboard_custom_from', { default: () => '' })
    const dashToCookie = useCookie<string>('tf_dashboard_custom_to', { default: () => '' })
    const dashPixelCookie = useCookie<string>('tf_dashboard_selected_pixel', { default: () => '' })
    const dashSourceCookie = useCookie<DashboardTrafficSource>('tf_dashboard_source', { default: () => 'all' })
    const dashPrivacyCookie = useCookie<string | null>('tf_dashboard_privacy', { default: () => null })

    authStore.hydrate(authTokenCookie.value, authUserCookie.value)
    tenantStore.hydrate(activeAccountCookie.value)

    const resolvedTheme = tfThemeCookie.value || legacyThemeCookie.value || 'dark'
    themeStore.hydrate(resolvedTheme)
    if (import.meta.client && !tfThemeCookie.value) {
      setBrowserCookie('tf_theme', themeStore.theme, 60 * 60 * 24 * 365)
    }

    dashboardStore.hydrate({
      period: dashPeriodCookie.value,
      customFrom: dashFromCookie.value,
      customTo: dashToCookie.value,
      selectedPixel: dashPixelCookie.value,
      trafficSource: dashSourceCookie.value,
      privacyMode: dashPrivacyCookie.value === '1',
    })

    // Mantém os refs do useCookie sincronizados com as stores para middlewares Nuxt
    watch(() => authStore.token, (val) => { authTokenCookie.value = val })
    watch(() => authStore.user, (val) => { authUserCookie.value = val })
    watch(() => tenantStore.activeAccountId, (val) => { activeAccountCookie.value = val })
    watch(() => themeStore.theme, (val) => {
      tfThemeCookie.value = val
      legacyThemeCookie.value = val
    })

    function getResolvedBearerToken(): string | null {
      if (authStore.token && authStore.token.trim()) {
        return authStore.token.trim()
      }
      if (authTokenCookie.value && authTokenCookie.value.trim()) {
        return authTokenCookie.value.trim()
      }
      if (import.meta.client) {
        try {
          const local = localStorage.getItem('auth_token')
          if (local && local.trim()) {
            authStore.token = local.trim()
            return local.trim()
          }
        } catch {}
        try {
          const cookieVal = getBrowserCookie('auth_token')
          if (cookieVal && cookieVal.trim()) {
            authStore.token = cookieVal.trim()
            return cookieVal.trim()
          }
        } catch {}
      }
      return null
    }

    const publicApi = $fetch.create({
      baseURL: apiBase,
      credentials: 'omit',
      onRequest({ options }) {
        const headers = new Headers(options.headers as HeadersInit | undefined)
        if (!headers.has('Content-Type')) {
          headers.set('Content-Type', 'application/json')
        }
        headers.delete('Authorization')
        headers.delete('X-Account-ID')
        options.headers = headers
      },
      onResponseError({ response }) {
        throw buildNormalizedError(response.status, response._data, response.statusText)
      },
    })

    const api = $fetch.create({
      baseURL: apiBase,
      onRequest({ request, options }) {
        const urlStr = typeof request === 'string' ? request : String(request)
        const cleanPath = urlStr.split('?')[0] || ''
        const isPublicRoute =
          !cleanPath.includes('/admin/') &&
          (cleanPath.includes('/public/') || cleanPath === '/branding' || cleanPath.endsWith('/api/v1/branding'))

        const headers = new Headers(options.headers as HeadersInit | undefined)
        if (!headers.has('Content-Type') && !(options.body instanceof FormData)) {
          headers.set('Content-Type', 'application/json')
        }

        if (!isPublicRoute) {
          const bearerToken = getResolvedBearerToken()
          if (bearerToken && !headers.has('Authorization')) {
            headers.set('Authorization', `Bearer ${bearerToken}`)
          }
          if (tenantStore.activeAccountId && !headers.has('X-Account-ID')) {
            headers.set('X-Account-ID', tenantStore.activeAccountId)
          }
        }
        options.headers = headers
      },
      onResponseError({ request, response }) {
        const status = response.status
        const data = response._data
        const urlStr = typeof request === 'string' ? request : String(request)

        if (status === 423 && data?.service_notice?.active) {
          serviceNotice.value = data.service_notice
          authStore.clearSession()
        } else if (
          status === 401 &&
          !urlStr.includes('/admin/login') &&
          !urlStr.includes('/admin/logout')
        ) {
          const isMissingHeaderErr =
            typeof data === 'string'
              ? data.includes('missing bearer token')
              : (data?.error?.includes('missing bearer token') || false)
          const fallbackToken = getResolvedBearerToken()

          if (!isMissingHeaderErr || !fallbackToken) {
            authStore.clearSession()
            if (import.meta.client) {
              const path = window.location.pathname
              if (!path.startsWith('/login') && !path.startsWith('/setup') && !path.startsWith('/cliente')) {
                void navigateTo('/login', { replace: true })
              }
            }
          }
        }

        throw buildNormalizedError(status, data, response.statusText)
      },
    })

    return {
      provide: {
        api: api as unknown as $Fetch,
        publicApi: publicApi as unknown as $Fetch,
      },
    }
  },
})

