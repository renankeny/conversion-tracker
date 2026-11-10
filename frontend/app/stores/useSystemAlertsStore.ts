import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import type { HealthAlert, HealthAlertsResponse } from '#shared/types'
import { useTenantStore } from '~/stores/useTenantStore'

export const useSystemAlertsStore = defineStore('systemAlerts', () => {
  const alerts = ref<HealthAlert[]>([])
  const dismissedIds = ref<string[]>([])
  const loading = ref(false)
  const loaded = ref(false)
  const loadedAccountId = ref<string | null>(null)

  let inFlightPromise: Promise<HealthAlert[]> | null = null

  const visibleAlerts = computed(() =>
    alerts.value.filter((a) => !dismissedIds.value.includes(a.id)),
  )

  async function fetchAlerts(force = false): Promise<HealthAlert[]> {
    const tenantStore = useTenantStore()
    const currentAcc = tenantStore.activeAccountId || ''

    if (!force && loaded.value && loadedAccountId.value === currentAcc) {
      return alerts.value
    }
    if (!force && inFlightPromise && loadedAccountId.value === currentAcc) {
      return inFlightPromise
    }

    loading.value = true
    loadedAccountId.value = currentAcc

    const api = useAPI()
    inFlightPromise = (async () => {
      try {
        const res = await api.get<HealthAlertsResponse>('/admin/system/health-alerts')
        alerts.value = res?.alerts || []
        loaded.value = true
        return alerts.value
      } catch {
        return alerts.value
      } finally {
        loading.value = false
        inFlightPromise = null
      }
    })()

    return inFlightPromise
  }

  function dismissAlert(id: string): void {
    if (!dismissedIds.value.includes(id)) {
      dismissedIds.value = [...dismissedIds.value, id]
    }
  }

  function invalidate(): void {
    alerts.value = []
    dismissedIds.value = []
    loaded.value = false
    loadedAccountId.value = null
    inFlightPromise = null
  }

  return {
    alerts,
    dismissedIds,
    visibleAlerts,
    loading,
    loaded,
    fetchAlerts,
    dismissAlert,
    invalidate,
  }
})

