import { computed } from 'vue'
import { useAuthStore } from '~/stores/useAuthStore'
import { useToast } from '~/composables/useToast'
import { errorMessage } from '~/utils/errorMessage'

export interface SystemUpdateStatusResponse {
  current_version: string
  latest_version?: string
  published_at?: string
  update_available: boolean
  can_auto_update: boolean
  update_status?: string
  update_message?: string
  checked_at?: string
}

interface TriggerUpdateResponse {
  ok: boolean
  status: string
  message: string
}

let pollTimer: ReturnType<typeof setInterval> | null = null
let pollTicks = 0

export const useSystemUpdate = () => {
  const api = useAPI()
  const authStore = useAuthStore()
  const { add: addToast } = useToast()

  const status = useState<SystemUpdateStatusResponse | null>('sys_update_status', () => null)
  const checked = useState<boolean>('sys_update_checked', () => false)
  const checking = useState<boolean>('sys_update_checking', () => false)
  const updating = useState<boolean>('sys_update_triggering', () => false)
  const modalOpen = useState<boolean>('sys_update_modal_open', () => false)
  const errorMsg = useState<string>('sys_update_error', () => '')

  const hasUpdate = computed(() => authStore.isOwner && !!status.value?.update_available)
  const isUpdatingNow = computed(() => {
    const st = status.value?.update_status
    return st === 'queued' || st === 'updating'
  })

  const stopPolling = () => {
    if (pollTimer) {
      clearInterval(pollTimer)
      pollTimer = null
    }
    pollTicks = 0
  }

  const startPolling = () => {
    if (!import.meta.client || pollTimer) return
    pollTicks = 0
    pollTimer = setInterval(async () => {
      pollTicks++
      if (pollTicks > 45 || !authStore.isAuthenticated || !authStore.isOwner) {
        stopPolling()
        return
      }
      const prevAvailable = !!status.value?.update_available
      await checkForUpdates(true)
      if (prevAvailable && !status.value?.update_available) {
        stopPolling()
        modalOpen.value = false
        addToast({
          type: 'success',
          title: 'Plataforma atualizada!',
          message: 'A nova versão já está ativa. Recarregando o painel...',
        })
        setTimeout(() => window.location.reload(), 1200)
      }
    }, 12000)
  }

  const checkForUpdates = async (force = false): Promise<void> => {
    if (!authStore.isAuthenticated || !authStore.isOwner) {
      status.value = null
      checked.value = false
      return
    }
    if (checked.value && !force) return
    if (checking.value) return

    checking.value = true
    errorMsg.value = ''
    try {
      const qs = force ? '?force=1' : ''
      const res = await api.get<SystemUpdateStatusResponse>(`/admin/system/update-status${qs}`)
      status.value = res || null
      checked.value = true
      if (res?.update_available && (res.update_status === 'queued' || res.update_status === 'updating')) {
        startPolling()
      } else if (!res?.update_available) {
        stopPolling()
      }
    } catch {
      // Falha transitória (ex.: API reiniciando durante deploy) não limpa nem quebra a UI
    } finally {
      checking.value = false
    }
  }

  const triggerUpdate = async (): Promise<boolean> => {
    if (!authStore.isOwner || updating.value) return false
    updating.value = true
    errorMsg.value = ''
    try {
      const res = await api.post<TriggerUpdateResponse>('/admin/system/update-now', {})
      if (status.value) {
        status.value = {
          ...status.value,
          update_status: res.status || 'queued',
          update_message: res.message || 'Atualização iniciada!',
        }
      }
      addToast({
        type: 'info',
        title: 'Atualização iniciada',
        message: res.message || 'Os serviços serão reiniciados automaticamente em instantes.',
      })
      startPolling()
      return true
    } catch (err: unknown) {
      errorMsg.value = errorMessage(err, 'Não foi possível iniciar a atualização agora.')
      return false
    } finally {
      updating.value = false
    }
  }

  const openModal = () => {
    errorMsg.value = ''
    modalOpen.value = true
    void checkForUpdates(true)
  }

  const closeModal = () => {
    modalOpen.value = false
  }

  const resetUpdateState = () => {
    stopPolling()
    status.value = null
    checked.value = false
    checking.value = false
    updating.value = false
    modalOpen.value = false
    errorMsg.value = ''
  }

  return {
    status,
    checking,
    updating,
    modalOpen,
    errorMsg,
    hasUpdate,
    isUpdatingNow,
    checkForUpdates,
    triggerUpdate,
    openModal,
    closeModal,
    resetUpdateState,
  }
}
