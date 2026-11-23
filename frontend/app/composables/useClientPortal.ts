import { computed } from 'vue'
import type { DashboardConfig } from '#shared/types'
import { errorMessage } from '~/utils/errorMessage'

interface PortalMetaState {
  loadingMeta: boolean
  metaLoaded: boolean
  notFound: boolean
  loadError: string | null
  pixelLabel: string
  currency: string
  dashboardConfig: DashboardConfig | null
}

export const useClientPortal = (token: string) => {
  const api = usePublicDashboard(token)
  const { fetchBranding } = useBranding()

  const state = useState<PortalMetaState>(`client-portal-meta:${token}`, () => ({
    loadingMeta: false,
    metaLoaded: false,
    notFound: false,
    loadError: null,
    pixelLabel: '',
    currency: 'BRL',
    dashboardConfig: null,
  }))

  const privacyMode = useState<boolean>(`client-portal-privacy:${token}`, () => false)

  const showCrm = computed(() => state.value.dashboardConfig?.public_show_crm !== false)
  const allowCrmEdit = computed(() => showCrm.value && Boolean(state.value.dashboardConfig?.public_allow_crm_edit))
  const showConversations = computed(() => state.value.dashboardConfig?.public_show_conversations !== false)

  const loadPortalMeta = async (force = false) => {
    if (state.value.loadingMeta) return
    if (state.value.metaLoaded && !force) return
    state.value.loadingMeta = true
    try {
      void fetchBranding()
      const res = await api.meta()
      state.value.pixelLabel = res.pixel_label || 'Projeto'
      state.value.currency = res.currency || 'BRL'
      state.value.dashboardConfig = res.dashboard_config || null
      state.value.metaLoaded = true
      state.value.notFound = false
      state.value.loadError = null
    } catch (err: unknown) {
      const code = err && typeof err === 'object' ? (err as { statusCode?: number }).statusCode : undefined
      if (code === 404 || code === 410) {
        state.value.notFound = true
        state.value.loadError = null
      } else {
        state.value.loadError = errorMessage(err, 'Não foi possível carregar o portal no momento.')
      }
    } finally {
      state.value.loadingMeta = false
    }
  }

  return {
    state,
    privacyMode,
    showCrm,
    allowCrmEdit,
    showConversations,
    loadPortalMeta,
  }
}
