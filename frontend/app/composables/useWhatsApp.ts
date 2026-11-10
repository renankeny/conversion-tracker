import type {
  WhatsAppChatWithMessages,
  WhatsAppInstance,
  WhatsAppInstanceInput,
  WhatsAppServer,
  WhatsAppServerHealth,
} from '#shared/types'
import { useWhatsAppStore } from '~/stores/useWhatsAppStore'

export const useWhatsApp = () => {
  const api = useAPI()
  const store = useWhatsAppStore()

  const saveServer = async (body: { name: string; url: string; api_key: string }): Promise<WhatsAppServer> => {
    const res = await api.put<{ server: WhatsAppServer }>('/admin/whatsapp/server', body)
    store.server = res.server
    return res.server
  }

  const testServer = (body?: { url?: string; api_key?: string }): Promise<WhatsAppServerHealth> => {
    return api.post<WhatsAppServerHealth>('/admin/whatsapp/server/test', body || {})
  }

  const removeServer = async (): Promise<void> => {
    await api.delete('/admin/whatsapp/server')
    store.server = null
  }

  const createInstance = async (input: WhatsAppInstanceInput): Promise<WhatsAppInstance> => {
    const created = await api.post<WhatsAppInstance>('/admin/whatsapp/instances', input)
    store.upsertInstance(created)
    return created
  }

  const updateInstance = async (id: number, input: WhatsAppInstanceInput): Promise<WhatsAppInstance> => {
    const updated = await api.put<WhatsAppInstance>(`/admin/whatsapp/instances/${id}`, input)
    store.upsertInstance(updated)
    return updated
  }

  const deleteInstance = async (id: number): Promise<void> => {
    await api.delete(`/admin/whatsapp/instances/${id}`)
    store.removeInstance(id)
  }

  const getQrCode = (id: number): Promise<{ connected: boolean; qrcode?: string; code?: string; status?: string }> => {
    return api.get(`/admin/whatsapp/instances/${id}/qr`)
  }

  const pairPhone = (id: number, phone: string): Promise<{ pairing_code: string }> => {
    return api.post(`/admin/whatsapp/instances/${id}/pair`, { phone })
  }

  const refreshStatus = async (id: number): Promise<WhatsAppInstance> => {
    const updated = await api.get<WhatsAppInstance>(`/admin/whatsapp/instances/${id}/status`)
    if (updated?.id) store.upsertInstance(updated)
    return updated
  }

  const disconnectInstance = async (id: number): Promise<void> => {
    await api.post(`/admin/whatsapp/instances/${id}/disconnect`, {})
    store.updateInstanceStatus(id, 'CLOSED')
  }

  const getChatByLead = (leadId: number): Promise<WhatsAppChatWithMessages> => {
    return api.get<WhatsAppChatWithMessages>(`/admin/whatsapp/chats/by-lead/${leadId}`)
  }

  const getChatByTrackedUser = (userId: number): Promise<WhatsAppChatWithMessages> => {
    return api.get<WhatsAppChatWithMessages>(`/admin/whatsapp/chats/by-user/${userId}`)
  }

  const convertChat = (
    chatId: number,
    body: { event_name: string; value?: number; currency?: string },
  ): Promise<{ queued: boolean; event_id: string }> => {
    return api.post(`/admin/whatsapp/chats/${chatId}/convert`, body)
  }

  return {
    saveServer,
    testServer,
    removeServer,
    createInstance,
    updateInstance,
    deleteInstance,
    getQrCode,
    pairPhone,
    refreshStatus,
    disconnectInstance,
    getChatByLead,
    getChatByTrackedUser,
    convertChat,
  }
}

