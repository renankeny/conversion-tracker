import { storeToRefs } from 'pinia'
import type { PixelStatusFilter } from '~/components/pixels/PixelsToolbar.vue'
import type { GoogleAdsAccount, MetaAccount, Pixel, PixelInput } from '#shared/types'
import { usePixelStore } from '~/stores/usePixelStore'
import { useDashboardDataStore } from '~/stores/useDashboardDataStore'
import { useTenantStore } from '~/stores/useTenantStore'
import { errorMessage } from '~/utils/errorMessage'

export const usePixelsPage = () => {
  const pixelsApi = usePixels()
  const accountsApi = useAccounts()
  const metaApi = useMetaAccounts()
  const gadsApi = useGoogleAdsAccounts()
  const pixelStore = usePixelStore()
  const dashboardDataStore = useDashboardDataStore()
  const tenantStore = useTenantStore()
  const { pixels, loading } = storeToRefs(pixelStore)
  const toast = useToast()
  const { user, canManagePixels } = useAuth()

  const searchQuery = ref('')
  const statusFilter = ref<PixelStatusFilter>('all')
  const accountNamesById = ref<Record<number, string>>({})
  const metaById = ref<Record<number, MetaAccount>>({})
  const gadsById = ref<Record<number, GoogleAdsAccount>>({})

  const formModalOpen = ref(false)
  const editing = ref<Pixel | null>(null)
  const saving = ref(false)
  const formRef = ref<{ submit: () => void } | null>(null)
  const formInitialTab = ref<'basic' | 'tracker'>('basic')

  const deleteModalOpen = ref(false)
  const deletingPixel = ref<Pixel | null>(null)
  const deleting = ref(false)
  const installModalOpen = ref(false)
  const installingPixel = ref<Pixel | null>(null)
  const transferModalOpen = ref(false)
  const transferringPixel = ref<Pixel | null>(null)
  const transferring = ref(false)

  const canTransferPixels = computed(() => {
    const role = user.value?.role
    const allowedRole = role === 'superadmin' || role === 'owner' || role === 'admin'
    return allowedRole && Object.keys(accountNamesById.value).length > 1
  })

  const showCompanyBadge = computed(
    () => !tenantStore.activeAccountId && Object.keys(accountNamesById.value).length > 1,
  )

  const filteredPixels = computed(() => {
    const q = searchQuery.value.trim().toLowerCase()
    return pixels.value.filter((p) => {
      if (statusFilter.value === 'active' && !p.enabled) return false
      if (statusFilter.value === 'paused' && p.enabled) return false
      if (!q) return true
      const hay = [
        p.name,
        p.label,
        ...(p.domains || []),
        ...(p.pixel_ids || []),
        ...(p.tiktok_pixel_ids || []),
        accountNamesById.value[p.account_id] || '',
      ].join(' ').toLowerCase()
      return hay.includes(q)
    })
  })

  const clearFilters = () => {
    searchQuery.value = ''
    statusFilter.value = 'all'
  }

  const clearLegacyTestCodes = async () => {
    if (!canManagePixels.value) return
    const stale = pixels.value.filter(
      (p) => p.test_codes?.some((c) => (c || '').trim()) || p.tiktok_test_codes?.some((c) => (c || '').trim()),
    )
    if (!stale.length) return
    await Promise.allSettled(
      stale.map((p) =>
        pixelsApi.update(p.id, {
          test_codes: (p.pixel_ids || []).map(() => ''),
          tiktok_test_codes: (p.tiktok_pixel_ids || []).map(() => ''),
        }),
      ),
    )
  }

  const loadAuxData = async (force = false) => {
    const [accRes, metaRes, gadsRes] = await Promise.allSettled([
      accountsApi.list(),
      metaApi.list(force),
      gadsApi.list(force),
    ])
    if (accRes.status === 'fulfilled') {
      accountNamesById.value = Object.fromEntries((accRes.value.data || []).map((a) => [a.id, a.name]))
    }
    if (metaRes.status === 'fulfilled') {
      metaById.value = Object.fromEntries((metaRes.value || []).map((a) => [a.id, a]))
    }
    if (gadsRes.status === 'fulfilled') {
      gadsById.value = Object.fromEntries((gadsRes.value || []).map((a) => [a.id, a]))
    }
  }

  const fetchPixels = async (force = false) => {
    try {
      await Promise.all([pixelsApi.list(force), loadAuxData(force)])
      await clearLegacyTestCodes()
    } catch (error: unknown) {
      toast.error('Erro ao carregar', errorMessage(error, 'Não foi possível listar os pixels'))
    }
  }

  const openCreateModal = () => { editing.value = null; formInitialTab.value = 'basic'; formModalOpen.value = true }
  const openEditModal = (pixel: Pixel) => { editing.value = pixel; formInitialTab.value = 'basic'; formModalOpen.value = true }
  const openDeleteModal = (pixel: Pixel) => { deletingPixel.value = pixel; deleteModalOpen.value = true }
  const openInstallModal = (pixel: Pixel) => { installingPixel.value = pixel; installModalOpen.value = true }
  const openTransferModal = (pixel: Pixel) => { transferringPixel.value = pixel; transferModalOpen.value = true }

  const onEditFromInstall = (pixel: Pixel) => {
    if (!canManagePixels.value) return
    installModalOpen.value = false
    installingPixel.value = null
    editing.value = pixel
    formInitialTab.value = 'tracker'
    formModalOpen.value = true
  }

  const onFormSubmit = async (input: PixelInput) => {
    saving.value = true
    try {
      if (editing.value) {
        const updated = await pixelsApi.update(editing.value.id, input)
        toast.success('Pixel atualizado', `"${updated.label || updated.name}" salvo com sucesso`)
      } else {
        const created = await pixelsApi.create(input)
        toast.success('Pixel criado', `"${created.label || created.name}" pronto para receber eventos`)
      }
      dashboardDataStore.clearAds()
      formModalOpen.value = false
      editing.value = null
    } catch (error: unknown) {
      toast.error('Erro ao salvar', errorMessage(error, 'Não foi possível salvar o pixel'))
    } finally {
      saving.value = false
    }
  }

  const onConfirmDelete = async () => {
    if (!deletingPixel.value) return
    deleting.value = true
    try {
      const removedName = deletingPixel.value.label || deletingPixel.value.name
      await pixelsApi.remove(deletingPixel.value.id)
      dashboardDataStore.clearAds()
      toast.success('Pixel excluído', `"${removedName}" foi removido`)
      deleteModalOpen.value = false
      deletingPixel.value = null
    } catch (error: unknown) {
      toast.error('Erro ao excluir', errorMessage(error, 'Não foi possível excluir o pixel'))
    } finally {
      deleting.value = false
    }
  }

  const onTokenRegenerated = (updated: Pixel) => {
    installingPixel.value = updated
    pixelStore.upsertPixel(updated)
  }

  const onConfirmTransfer = async (pixelId: number, targetAccountId: number) => {
    transferring.value = true
    try {
      await pixelsApi.update(pixelId, { account_id: targetAccountId })
      dashboardDataStore.clearAll()
      toast.success('Pixel transferido', 'O pixel e seu histórico foram movidos para a empresa selecionada.')
      transferModalOpen.value = false
      transferringPixel.value = null
      await fetchPixels(true)
    } catch (error: unknown) {
      toast.error('Erro na transferência', errorMessage(error, 'Não foi possível transferir o pixel'))
    } finally {
      transferring.value = false
    }
  }

  watch(() => tenantStore.switchVersion, () => {
    formModalOpen.value = false
    editing.value = null
    fetchPixels(true)
  })
  onMounted(() => fetchPixels())

  return {
    pixels, loading, canManagePixels, canTransferPixels, showCompanyBadge,
    searchQuery, statusFilter, filteredPixels, clearFilters,
    accountNamesById, metaById, gadsById,
    formModalOpen, editing, saving, formRef, formInitialTab,
    deleteModalOpen, deletingPixel, deleting,
    installModalOpen, installingPixel,
    transferModalOpen, transferringPixel, transferring,
    openCreateModal, openEditModal, openDeleteModal, openInstallModal, openTransferModal,
    onEditFromInstall, onFormSubmit, onConfirmDelete, onTokenRegenerated, onConfirmTransfer,
  }
}
