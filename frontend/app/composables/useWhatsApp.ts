import type {
  AccountAISettings,
  AccountAISettingsInput,
  WhatsAppAIEvent,
  WhatsAppAIRunSummary,
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

  const getChatByLead = (
    leadId: number,
    params?: { before_id?: number | null; limit?: number },
  ): Promise<WhatsAppChatWithMessages> => {
    const q = new URLSearchParams()
    if (params?.before_id) q.set('before_id', String(params.before_id))
    if (params?.limit) q.set('limit', String(params.limit))
    const qs = q.toString() ? `?${q.toString()}` : ''
    return api.get<WhatsAppChatWithMessages>(`/admin/whatsapp/chats/by-lead/${leadId}${qs}`)
  }

  const getChatByTrackedUser = (
    userId: number,
    params?: { before_id?: number | null; limit?: number },
  ): Promise<WhatsAppChatWithMessages> => {
    const q = new URLSearchParams()
    if (params?.before_id) q.set('before_id', String(params.before_id))
    if (params?.limit) q.set('limit', String(params.limit))
    const qs = q.toString() ? `?${q.toString()}` : ''
    return api.get<WhatsAppChatWithMessages>(`/admin/whatsapp/chats/by-user/${userId}${qs}`)
  }

  const getChatMessages = (
    chatId: number,
    params?: { before_id?: number | null; limit?: number },
  ): Promise<WhatsAppChatWithMessages> => {
    const q = new URLSearchParams()
    if (params?.before_id) q.set('before_id', String(params.before_id))
    if (params?.limit) q.set('limit', String(params.limit))
    const qs = q.toString() ? `?${q.toString()}` : ''
    return api.get<WhatsAppChatWithMessages>(`/admin/whatsapp/chats/${chatId}/messages${qs}`)
  }

  const convertChat = (
    chatId: number,
    body: { event_name: string; value?: number; currency?: string },
  ): Promise<{ queued: boolean; event_id: string }> => {
    return api.post(`/admin/whatsapp/chats/${chatId}/convert`, body)
  }

  const getAISettings = (): Promise<AccountAISettings> => {
    return api.get<AccountAISettings>('/admin/whatsapp/ai-settings')
  }

  const updateAISettings = (body: AccountAISettingsInput): Promise<AccountAISettings> => {
    return api.put<AccountAISettings>('/admin/whatsapp/ai-settings', body)
  }

  const runAINow = (): Promise<WhatsAppAIRunSummary> => {
    return api.post<WhatsAppAIRunSummary>('/admin/whatsapp/ai-settings/run-now', {})
  }

  const contestAIEvent = (
    eventId: number,
    body: { reason: string; add_to_prompt?: boolean },
  ): Promise<WhatsAppAIEvent> => {
    return api.post<WhatsAppAIEvent>(`/admin/whatsapp/ai-events/${eventId}/contest`, body)
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
    getChatMessages,
    convertChat,
    getAISettings,
    updateAISettings,
    runAINow,
    contestAIEvent,
  }
}

