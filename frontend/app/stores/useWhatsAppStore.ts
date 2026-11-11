import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { WhatsAppInstance, WhatsAppServer } from '#shared/types'
import { useTenantStore } from '~/stores/useTenantStore'

export const useWhatsAppStore = defineStore('whatsapp', () => {
  const server = ref<WhatsAppServer | null>(null)
  const instances = ref<WhatsAppInstance[]>([])
  const loading = ref(false)
  const loaded = ref(false)
  const loadedAccountId = ref<string | null>(null)

  let inFlight: Promise<void> | null = null

  async function fetchAll(force = false): Promise<void> {
    const tenantStore = useTenantStore()
    const acc = tenantStore.activeAccountId || ''

    if (!force && loaded.value && loadedAccountId.value === acc) {
      return
    }
    if (!force && inFlight && loadedAccountId.value === acc) {
      return inFlight
    }

    loading.value = true
    loadedAccountId.value = acc
    const api = useAPI()

    inFlight = (async () => {
      try {
        const [srvRes, instRes] = await Promise.all([
          api.get<{ server: WhatsAppServer | null }>('/admin/whatsapp/server'),
          api.get<{ data: WhatsAppInstance[] }>('/admin/whatsapp/instances'),
        ])
        server.value = srvRes.server || null
        instances.value = instRes.data || []
        loaded.value = true
      } finally {
        loading.value = false
        inFlight = null
      }
    })()

    return inFlight
  }

  function invalidate(): void {
    loaded.value = false
    loadedAccountId.value = null
    server.value = null
    instances.value = []
    inFlight = null
  }

  function upsertInstance(inst: WhatsAppInstance): void {
    const idx = instances.value.findIndex((i) => i.id === inst.id)
    if (idx >= 0) {
      instances.value[idx] = inst
    } else {
      instances.value = [inst, ...instances.value]
    }
  }

  function updateInstanceStatus(
    id: number,
    status: string,
    ownerJid?: string,
    profileName?: string,
    profilePicUrl?: string,
  ): void {
    const idx = instances.value.findIndex((i) => i.id === id)
    if (idx < 0) return
    const cur = instances.value[idx]!
    const nextStatus = status || cur.connection_status
    const isReset = nextStatus === 'CLOSED' || (nextStatus === 'CONNECTING' && !ownerJid)
    instances.value[idx] = {
      ...cur,
      connection_status: nextStatus,
      owner_jid: isReset ? '' : (ownerJid ?? cur.owner_jid),
      profile_name: isReset ? '' : (profileName ?? cur.profile_name),
      profile_pic_url: isReset ? '' : (profilePicUrl ?? cur.profile_pic_url),
    }
  }

  function removeInstance(id: number): void {
    instances.value = instances.value.filter((i) => i.id !== id)
  }

  return {
    server,
    instances,
    loading,
    loaded,
    fetchAll,
    invalidate,
    upsertInstance,
    updateInstanceStatus,
    removeInstance,
  }
})

