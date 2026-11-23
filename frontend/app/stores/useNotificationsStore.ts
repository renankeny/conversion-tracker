import { defineStore } from 'pinia'
import type {
  ConfigurableNotificationCategory,
  NotificationPreferences,
  NotificationsOverview,
  PushSubscriptionItem,
  UserNotificationItem,
} from '#shared/types'

interface BeforeInstallPromptEvent extends Event {
  prompt: () => Promise<void>
  userChoice: Promise<{ outcome: 'accepted' | 'dismissed' }>
}

function urlBase64ToUint8Array(base64String: string): Uint8Array {
  const padding = '='.repeat((4 - (base64String.length % 4)) % 4)
  const base64 = (base64String + padding).replace(/-/g, '+').replace(/_/g, '/')
  const rawData = window.atob(base64)
  const outputArray = new Uint8Array(rawData.length)
  for (let i = 0; i < rawData.length; ++i) {
    outputArray[i] = rawData.charCodeAt(i)
  }
  return outputArray
}

export const useNotificationsStore = defineStore('notifications', () => {
  const api = useAPI()

  const vapidPublicKey = ref('')
  const unreadCount = ref(0)
  const preferences = ref<NotificationPreferences>({
    user_id: 0,
    account_scope_mode: 'all',
    enabled_account_ids: [],
    whatsapp_connection: true,
    lead_created: true,
    ai_lead_qualified: true,
    sale_approved: true,
    meta_account_lifecycle: true,
    google_ads_account_lifecycle: true,
    pixel_lifecycle: true,
    event_dispatch_error: false,
  })
  const subscriptions = ref<PushSubscriptionItem[]>([])
  const notifications = ref<UserNotificationItem[]>([])

  const loaded = ref(false)
  const loading = ref(false)
  const savingPrefs = ref(false)
  const subscribing = ref(false)
  const testingCategory = ref<ConfigurableNotificationCategory | null>(null)

  const pushSupported = ref(false)
  const permissionState = ref<NotificationPermission>('default')
  const isIOS = ref(false)
  const isStandalonePWA = ref(false)
  const currentEndpoint = ref('')
  const deferredInstallPrompt = shallowRef<BeforeInstallPromptEvent | null>(null)
  let clientInitialized = false

  const canInstallPWA = computed(() => Boolean(deferredInstallPrompt.value) && !isStandalonePWA.value)
  const isThisDeviceSubscribed = computed(() =>
    Boolean(currentEndpoint.value) && subscriptions.value.some((s) => s.endpoint === currentEndpoint.value),
  )

  async function initClientPWA(): Promise<void> {
    if (!import.meta.client || clientInitialized) return
    clientInitialized = true

    const ua = navigator.userAgent || ''
    isIOS.value = /iPad|iPhone|iPod/.test(ua) || (navigator.platform === 'MacIntel' && navigator.maxTouchPoints > 1)
    isStandalonePWA.value = window.matchMedia('(display-mode: standalone)').matches || Boolean((navigator as unknown as { standalone?: boolean }).standalone)
    pushSupported.value = 'serviceWorker' in navigator && 'PushManager' in window && 'Notification' in window

    if ('Notification' in window) {
      permissionState.value = Notification.permission
    }

    window.addEventListener('beforeinstallprompt', (e: Event) => {
      e.preventDefault()
      deferredInstallPrompt.value = e as BeforeInstallPromptEvent
    })
    window.addEventListener('appinstalled', () => {
      isStandalonePWA.value = true
      deferredInstallPrompt.value = null
    })

    if ('serviceWorker' in navigator) {
      try {
        const reg = await navigator.serviceWorker.register('/sw.js', { scope: '/' })
        if ('pushManager' in reg) {
          const existing = await reg.pushManager.getSubscription()
          currentEndpoint.value = existing?.endpoint || ''
        }
      } catch {
        // Ignora erro silenciosamente caso navegador restrinja SW em aba anônima
      }
    }
  }

  async function fetchOverview(force = false): Promise<void> {
    if (loading.value || (loaded.value && !force)) return
    loading.value = true
    try {
      const data = await api.get<NotificationsOverview>('/admin/notifications')
      vapidPublicKey.value = data.vapid_public_key || ''
      unreadCount.value = data.unread_count || 0
      preferences.value = {
        ...preferences.value,
        ...data.preferences,
        enabled_account_ids: data.preferences?.enabled_account_ids || [],
      }
      subscriptions.value = data.subscriptions || []
      notifications.value = (data.notifications || []).slice(0, 10)
      loaded.value = true
    } finally {
      loading.value = false
    }
  }

  async function savePreferences(next: Partial<NotificationPreferences>): Promise<void> {
    const merged: NotificationPreferences = {
      ...preferences.value,
      ...next,
    }
    preferences.value = merged
    savingPrefs.value = true
    try {
      const saved = await api.put<NotificationPreferences>('/admin/notifications/preferences', merged)
      preferences.value = { ...saved, enabled_account_ids: saved.enabled_account_ids || [] }
    } finally {
      savingPrefs.value = false
    }
  }

  async function installPWA(): Promise<boolean> {
    const promptEv = deferredInstallPrompt.value
    if (!promptEv) return false
    await promptEv.prompt()
    const choice = await promptEv.userChoice
    if (choice.outcome === 'accepted') {
      deferredInstallPrompt.value = null
      return true
    }
    return false
  }

  async function subscribeThisDevice(): Promise<void> {
    if (!import.meta.client || !pushSupported.value) {
      throw new Error('Este navegador não suporta Web Push.')
    }
    subscribing.value = true
    try {
      const perm = await Notification.requestPermission()
      permissionState.value = perm
      if (perm !== 'granted') {
        throw new Error('Permissão de notificações bloqueada no navegador.')
      }
      if (!vapidPublicKey.value) {
        const keyRes = await api.get<{ public_key: string }>('/push/vapid-public-key')
        vapidPublicKey.value = keyRes.public_key || ''
      }
      const reg = await navigator.serviceWorker.ready
      let sub = await reg.pushManager.getSubscription()
      if (!sub) {
        sub = await reg.pushManager.subscribe({
          userVisibleOnly: true,
          applicationServerKey: urlBase64ToUint8Array(vapidPublicKey.value) as unknown as BufferSource,
        })
      }
      const json = sub.toJSON()
      const saved = await api.post<PushSubscriptionItem>('/admin/notifications/subscriptions', {
        endpoint: sub.endpoint,
        keys: {
          p256dh: json.keys?.p256dh || '',
          auth: json.keys?.auth || '',
        },
      })
      currentEndpoint.value = sub.endpoint
      const idx = subscriptions.value.findIndex((s) => s.id === saved.id || s.endpoint === saved.endpoint)
      if (idx >= 0) subscriptions.value[idx] = saved
      else subscriptions.value.unshift(saved)
    } finally {
      subscribing.value = false
    }
  }

  async function removeSubscription(sub: PushSubscriptionItem): Promise<void> {
    await api.delete(`/admin/notifications/subscriptions/${sub.id}`)
    subscriptions.value = subscriptions.value.filter((s) => s.id !== sub.id)
    if (import.meta.client && sub.endpoint === currentEndpoint.value && 'serviceWorker' in navigator) {
      const reg = await navigator.serviceWorker.getRegistration('/')
      const browserSub = await reg?.pushManager.getSubscription()
      if (browserSub && browserSub.endpoint === sub.endpoint) {
        await browserSub.unsubscribe().catch(() => {})
      }
      currentEndpoint.value = ''
    }
  }

  async function markRead(notifId: number): Promise<void> {
    const target = notifications.value.find((n) => n.id === notifId)
    if (target && !target.is_read) {
      target.is_read = true
      unreadCount.value = Math.max(0, unreadCount.value - 1)
    }
    await api.post(`/admin/notifications/${notifId}/read`, {})
  }

  async function markAllRead(): Promise<void> {
    for (const n of notifications.value) n.is_read = true
    unreadCount.value = 0
    await api.post('/admin/notifications/read-all', {})
  }

  async function sendTest(category: ConfigurableNotificationCategory): Promise<void> {
    testingCategory.value = category
    try {
      await api.post<UserNotificationItem>('/admin/notifications/test', { category })
    } finally {
      testingCategory.value = null
    }
  }

  function pushIncomingRealtime(item: UserNotificationItem): void {
    if (!item?.id) return
    if (notifications.value.some((n) => n.id === item.id)) return
    notifications.value.unshift(item)
    if (notifications.value.length > 10) {
      notifications.value.splice(10)
    }
    if (!item.is_read) {
      unreadCount.value++
    }
  }

  return {
    vapidPublicKey, unreadCount, preferences, subscriptions, notifications,
    loaded, loading, savingPrefs, subscribing, testingCategory,
    pushSupported, permissionState, isIOS, isStandalonePWA, canInstallPWA, isThisDeviceSubscribed, currentEndpoint,
    initClientPWA, fetchOverview, savePreferences, installPWA, subscribeThisDevice,
    removeSubscription, markRead, markAllRead, sendTest, pushIncomingRealtime,
  }
})
