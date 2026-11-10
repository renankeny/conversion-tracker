import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Pixel } from '#shared/types'
import { useTenantStore } from '~/stores/useTenantStore'

export const usePixelStore = defineStore('pixels', () => {
  const pixels = ref<Pixel[]>([])
  const loading = ref(false)
  const loaded = ref(false)
  const loadedAccountId = ref<string | null>(null)
  const error = ref<string | null>(null)

  let inFlightPromise: Promise<Pixel[]> | null = null

  async function fetch(force = false): Promise<Pixel[]> {
    const tenantStore = useTenantStore()
    const currentAcc = tenantStore.activeAccountId || ''

    if (!force && loaded.value && loadedAccountId.value === currentAcc) {
      return pixels.value
    }
    if (!force && inFlightPromise && loadedAccountId.value === currentAcc) {
      return inFlightPromise
    }

    loading.value = true
    error.value = null
    loadedAccountId.value = currentAcc

    const api = useAPI()
    inFlightPromise = (async () => {
      try {
        const response = await api.get<{ data: Pixel[] }>('/admin/pixels')
        const list = response.data ?? []
        pixels.value = list
        loaded.value = true
        return list
      } catch (e: any) {
        error.value = e.message ?? 'Erro ao carregar pixels'
        throw e
      } finally {
        loading.value = false
        inFlightPromise = null
      }
    })()

    return inFlightPromise
  }

  function invalidate(): void {
    loaded.value = false
    loadedAccountId.value = null
    pixels.value = []
    error.value = null
    inFlightPromise = null
  }

  function upsertPixel(pixel: Pixel): void {
    const idx = pixels.value.findIndex((p) => p.id === pixel.id)
    if (idx >= 0) {
      pixels.value[idx] = pixel
    } else {
      pixels.value.unshift(pixel)
    }
  }

  function removePixel(id: number): void {
    pixels.value = pixels.value.filter((p) => p.id !== id)
  }

  return {
    pixels,
    loading,
    loaded,
    loadedAccountId,
    error,
    fetch,
    invalidate,
    upsertPixel,
    removePixel,
  }
})
