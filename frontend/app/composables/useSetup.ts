import type { AdminUser, PlatformBranding } from '#shared/types'
import { useAuthStore } from '~/stores/useAuthStore'
import { useTenantStore } from '~/stores/useTenantStore'
import { setBrowserCookie } from '~/utils/cookies'

export interface SetupInitPayload {
  admin_name: string
  admin_email: string
  admin_phone: string
  admin_password: string
  account_name: string
  branding: PlatformBranding
}

export interface SetupInitResponse {
  access_token: string
  user: AdminUser
  branding: PlatformBranding
}

export interface ServiceNoticeState {
  active: boolean
  badge?: string
  title?: string
  body?: string
  items?: string[]
}

export const useSetup = () => {
  const api = useAPI()
  const needsSetup = useState<boolean | null>('platform_needs_setup', () => null)
  const serviceNotice = useState<ServiceNoticeState | null>('platform_service_notice', () => null)
  const authStore = useAuthStore()
  const tenantStore = useTenantStore()
  const { branding, applyBrandingVisuals } = useBranding()

  const applyServiceNotice = (notice: ServiceNoticeState | null | undefined) => {
    if (!notice?.active) {
      serviceNotice.value = null
      return
    }
    serviceNotice.value = notice
    authStore.clearSession()
  }

  const checkSetupStatus = async (force = false): Promise<boolean> => {
    if (needsSetup.value !== null && !force) {
      return needsSetup.value
    }
    try {
      const res = await api.publicGet<{
        needs_setup: boolean
        service_notice?: ServiceNoticeState | null
      }>('/setup/status')
      if (res?.service_notice?.active) {
        applyServiceNotice(res.service_notice)
        return false
      }
      serviceNotice.value = null
      needsSetup.value = !!res?.needs_setup
      return needsSetup.value
    } catch {
      return false
    }
  }

  const verifySetupEmail = async (email: string): Promise<void> => {
    await api.publicRequest('/setup/verify-email', {
      method: 'POST',
      body: { email },
    })
  }

  const completeSetup = async (payload: SetupInitPayload): Promise<SetupInitResponse> => {
    const response = await api.publicRequest<SetupInitResponse>('/setup/init', {
      method: 'POST',
      body: payload,
    })

    const maxAge = 60 * 60 * 24 * 30
    const accId = response.user?.account_id ? String(response.user.account_id) : '1'

    authStore.token = response.access_token
    authStore.setUser(response.user, maxAge)
    tenantStore.setActiveAccountId(accId, maxAge)
    setBrowserCookie('auth_token', response.access_token, maxAge)
    if (import.meta.client) {
      try {
        localStorage.setItem('auth_token', response.access_token)
      } catch {}
    }

    if (response.branding) {
      branding.value = response.branding
      applyBrandingVisuals(response.branding)
    }

    needsSetup.value = false
    return response
  }

  return {
    needsSetup,
    serviceNotice,
    applyServiceNotice,
    checkSetupStatus,
    verifySetupEmail,
    completeSetup,
  }
}
