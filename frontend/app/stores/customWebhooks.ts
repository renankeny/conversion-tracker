import { defineStore } from 'pinia'
import { ref } from 'vue'
import type {
  CreateCustomWebhookInput,
  CustomMappingConfig,
  CustomWebhook,
  UpdateCustomWebhookInput,
} from '#shared/types'
import { useTenantStore } from '~/stores/useTenantStore'

export type { CustomMappingConfig, CustomWebhook }

export const useCustomWebhooksStore = defineStore('customWebhooks', () => {
  const items = ref<CustomWebhook[]>([])
  const loading = ref(false)
  const loaded = ref(false)
  const loadedAccountId = ref<string | null>(null)

  let inFlightPromise: Promise<CustomWebhook[]> | null = null

  async function fetchItems(force = false): Promise<CustomWebhook[]> {
    const tenantStore = useTenantStore()
    const currentAcc = tenantStore.activeAccountId || ''

    if (!force && loaded.value && loadedAccountId.value === currentAcc) {
      return items.value
    }
    if (!force && inFlightPromise && loadedAccountId.value === currentAcc) {
      return inFlightPromise
    }

    loading.value = true
    loadedAccountId.value = currentAcc

    const api = useAPI()
    inFlightPromise = (async () => {
      try {
        const data = await api.get<CustomWebhook[]>('/admin/custom-webhooks')
        items.value = data || []
        loaded.value = true
        return items.value
      } finally {
        loading.value = false
        inFlightPromise = null
      }
    })()

    return inFlightPromise
  }

  async function createItem(payload: CreateCustomWebhookInput): Promise<CustomWebhook> {
    const api = useAPI()
    const data = await api.post<CustomWebhook>('/admin/custom-webhooks', payload)
    items.value.push(data)
    return data
  }

  async function updateItem(id: string, payload: UpdateCustomWebhookInput): Promise<CustomWebhook> {
    const api = useAPI()
    const data = await api.put<CustomWebhook>(`/admin/custom-webhooks/${id}`, payload)
    const idx = items.value.findIndex((i) => i.id === id)
    if (idx !== -1) {
      items.value[idx] = data
    }
    return data
  }

  async function deleteItem(id: string): Promise<void> {
    const api = useAPI()
    await api.delete(`/admin/custom-webhooks/${id}`)
    items.value = items.value.filter((i) => i.id !== id)
  }

  function invalidate(): void {
    items.value = []
    loaded.value = false
    loadedAccountId.value = null
    inFlightPromise = null
  }

  return {
    items,
    loading,
    loaded,
    fetchItems,
    createItem,
    updateItem,
    deleteItem,
    invalidate,
  }
})
