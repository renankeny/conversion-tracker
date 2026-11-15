import { computed, onMounted, ref, watch } from 'vue'
import { PLATFORMS } from '#shared/constants/platforms'
import type { Pixel, PlatformSummaryItem, PlatformsSummaryResponse } from '#shared/types'
import { useAuth } from '~/composables/useAuth'
import { usePlatformTokenModal } from '~/composables/usePlatformTokenModal'
import { useTenantStore } from '~/stores/useTenantStore'

export type PlatformStatusFilter = 'all' | 'active'

export const usePlatformsPage = () => {
  const auth = useAuth()
  const api = useAPI()
  const tenantStore = useTenantStore()

  const searchQuery = ref('')
  const statusFilter = ref<PlatformStatusFilter>('all')
  const localAccountId = ref('')

  const summaryItems = ref<Record<string, PlatformSummaryItem>>({})
  const loadingSummary = ref(false)
  const accountPixels = ref<Pixel[]>([])

  const isGlobalTenantMode = computed(() => Boolean(auth.isOwner.value && !tenantStore.activeAccountId))

  const effectiveAccountId = computed(() => {
    if (tenantStore.activeAccountId) return String(tenantStore.activeAccountId)
    if (localAccountId.value) return localAccountId.value
    if (tenantStore.accounts[0]?.id) return String(tenantStore.accounts[0].id)
    if (auth.user.value?.account_id) return String(auth.user.value.account_id)
    return ''
  })

  const accountHeaders = computed<Record<string, string>>(() => {
    const headers: Record<string, string> = {}
    const acc = effectiveAccountId.value
    if (acc) headers['X-Account-ID'] = acc
    return headers
  })

  const modal = usePlatformTokenModal({
    effectiveAccountId,
    accountHeaders,
    summaryItems,
  })

  const isPlatformActive = (id: string): boolean => {
    const st = summaryItems.value[id]
    return Boolean(st && (st.has_token || st.total_accepted > 0))
  }

  const activeCount = computed(() => PLATFORMS.filter((p) => isPlatformActive(p.id)).length)

  const filteredPlatforms = computed(() => {
    const q = searchQuery.value.trim().toLowerCase()
    return PLATFORMS.filter((p) => {
      if (statusFilter.value === 'active' && !isPlatformActive(p.id)) {
        return false
      }
      if (!q) return true
      return (
        p.name.toLowerCase().includes(q) ||
        p.id.toLowerCase().includes(q) ||
        p.description.toLowerCase().includes(q)
      )
    })
  })

  const enabledPixels = computed(() => accountPixels.value.filter((p) => p.enabled))
  const hasMultiplePixels = computed(() => enabledPixels.value.length > 1)

  const pixelOptions = computed(() => [
    { value: '', label: 'Automático da empresa (identifica pelo lead)' },
    ...enabledPixels.value.map((p) => ({
      value: p.name,
      label: p.label && p.label !== p.name ? `${p.label} (${p.name})` : p.name,
    })),
  ])

  const fetchSummaryAndPixels = async () => {
    loadingSummary.value = true
    const headers = accountHeaders.value
    const acc = effectiveAccountId.value
    const accQuery = acc ? `?account_id=${encodeURIComponent(acc)}` : ''
    try {
      const [sumRes, pixRes] = await Promise.all([
        api.get<PlatformsSummaryResponse>(`/admin/integrations/platforms/summary${accQuery}`, { headers }),
        acc
          ? api.get<{ data: Pixel[] }>('/admin/pixels', { headers }).catch(() => ({ data: [] as Pixel[] }))
          : Promise.resolve({ data: [] as Pixel[] }),
      ])
      summaryItems.value = sumRes.items || {}
      accountPixels.value = pixRes.data || []
    } catch {
      summaryItems.value = {}
    } finally {
      loadingSummary.value = false
    }
  }

  watch(() => tenantStore.switchVersion, () => {
    modal.showModal.value = false
    modal.selectedPixelSlug.value = ''
    fetchSummaryAndPixels()
  })

  watch(effectiveAccountId, (next, prev) => {
    if (next && next !== prev) {
      modal.selectedPixelSlug.value = ''
      fetchSummaryAndPixels()
      if (modal.showModal.value && modal.selectedPlatform.value) {
        modal.loadSelectedPlatformToken(modal.selectedPlatform.value.id)
      }
    }
  })

  onMounted(async () => {
    if (auth.canManageAccounts.value) {
      await tenantStore.fetchAccounts()
    }
    if (!tenantStore.activeAccountId && tenantStore.accounts[0]?.id) {
      localAccountId.value = String(tenantStore.accounts[0].id)
    }
    await fetchSummaryAndPixels()
  })

  return {
    canManagePixels: auth.canManagePixels,
    accountOptions: computed(() => tenantStore.accountOptions),
    isGlobalTenantMode,
    localAccountId,
    effectiveAccountId,
    searchQuery,
    statusFilter,
    summaryItems,
    loadingSummary,
    activeCount,
    filteredPlatforms,
    pixelOptions,
    hasMultiplePixels,
    selectedPixelSlug: modal.selectedPixelSlug,
    showModal: modal.showModal,
    selectedPlatform: modal.selectedPlatform,
    modalWebhookUrl: modal.modalWebhookUrl,
    tokenInput: modal.tokenInput,
    hasSavedToken: modal.hasSavedToken,
    maskedToken: modal.maskedToken,
    loadingToken: modal.loadingToken,
    savingToken: modal.savingToken,
    copiedKey: modal.copiedKey,
    openModal: modal.openModal,
    saveToken: modal.saveToken,
    clearToken: modal.clearToken,
    copyCardUrl: modal.copyCardUrl,
    copyModalUrl: modal.copyModalUrl,
  }
}
