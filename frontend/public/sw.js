self.addEventListener('install', () => {
  self.skipWaiting()
})

self.addEventListener('activate', (event) => {
  event.waitUntil(self.clients.claim())
})

self.addEventListener('push', (event) => {
  let payload = {
    title: 'Nova notificação',
    body: '',
    url: '/notificacoes',
    tag: 'tf-notif',
  }

  if (event.data) {
    try {
      const parsed = event.data.json()
      payload = {
        title: parsed.title || payload.title,
        body: parsed.body || '',
        url: parsed.url || '/notificacoes',
        tag: parsed.tag || 'tf-notif',
      }
    } catch {
      payload.body = event.data.text() || ''
    }
  }

  const options = {
    body: payload.body,
    icon: '/pwa-icon.svg',
    badge: '/favicon.ico',
    tag: payload.tag,
    renotify: true,
    data: {
      url: payload.url,
    },
  }

  event.waitUntil(self.registration.showNotification(payload.title, options))
})

self.addEventListener('notificationclick', (event) => {
  event.notification.close()
  const rawTarget = (event.notification.data && event.notification.data.url) || '/notificacoes'
  const targetUrl = new URL(rawTarget, self.location.origin).href

  event.waitUntil(
    self.clients.matchAll({ type: 'window', includeUncontrolled: true }).then((windowClients) => {
      for (const client of windowClients) {
        if (client.url.startsWith(self.location.origin) && 'focus' in client) {
          if ('navigate' in client) {
            return client.navigate(targetUrl).then((c) => (c ? c.focus() : undefined))
          }
          return client.focus()
        }
      }
      if (self.clients.openWindow) {
        return self.clients.openWindow(targetUrl)
      }
    }),
  )
})
