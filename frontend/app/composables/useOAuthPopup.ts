export interface OAuthCompletePayload {
  type: 'trackfind:oauth-complete'
  channel: 'meta' | 'google_ads' | string
  status: 'success' | 'error'
  accountId: number
  message: string
  ts: number
}

export interface PreparedOAuthSession {
  url: string
  accountId: number
  createdTemp?: boolean
}

export interface RunOAuthPopupOptions {
  channel: 'meta' | 'google_ads'
  providerLabel: string
  prepare: (popupMode: boolean) => Promise<PreparedOAuthSession>
  onSuccess: (payload: OAuthCompletePayload) => Promise<void> | void
  onError: (message: string, accountId: number, createdTemp: boolean) => Promise<void> | void
  onManualClose: (accountId: number, createdTemp: boolean) => Promise<void> | void
}

const POPUP_LOADING_HTML = (label: string) => `<!doctype html>
<html lang="pt-BR"><head><meta charset="utf-8"><title>Conectando ${label}</title>
<style>
  body { margin: 0; min-height: 100vh; display: flex; align-items: center; justify-content: center; background: #18181b; color: #f4f4f5; font-family: system-ui, -apple-system, sans-serif; }
  .box { text-align: center; padding: 24px; max-width: 300px; }
  .spin { width: 32px; height: 32px; margin: 0 auto 14px; border-radius: 50%; border: 3px solid rgba(255,255,255,0.14); border-top-color: #22c55e; animation: r 0.75s linear infinite; }
  p { margin: 6px 0 0; font-size: 13px; color: #a1a1aa; line-height: 1.45; }
  h3 { margin: 0; font-size: 15px; font-weight: 600; }
  @keyframes r { to { transform: rotate(360deg); } }
</style></head>
<body><div class="box"><div class="spin"></div><h3>Conectando ao ${label}</h3><p>Aguarde, abrindo janela segura de autenticação...</p></div></body></html>`

export const useOAuthPopup = () => {
  const openCenteredWindow = (providerLabel: string): Window | null => {
    if (!import.meta.client) return null
    const width = 540
    const height = 680
    const dualLeft = window.screenLeft ?? window.screenX ?? 0
    const dualTop = window.screenTop ?? window.screenY ?? 0
    const w = window.innerWidth || document.documentElement.clientWidth || screen.width
    const h = window.innerHeight || document.documentElement.clientHeight || screen.height
    const left = Math.max(0, Math.round(dualLeft + (w - width) / 2))
    const top = Math.max(0, Math.round(dualTop + (h - height) / 2))
    const features = `width=${width},height=${height},left=${left},top=${top},resizable=yes,scrollbars=yes`

    const win = window.open('about:blank', 'trackfind_oauth_popup', features)
    if (win) {
      try {
        win.document.open()
        win.document.write(POPUP_LOADING_HTML(providerLabel))
        win.document.close()
        win.focus()
      } catch {}
    }
    return win
  }

  const runOAuthPopup = async (opts: RunOAuthPopupOptions): Promise<void> => {
    if (!import.meta.client) return

    const popup = openCenteredWindow(opts.providerLabel)
    if (!popup) {
      const fallback = await opts.prepare(false)
      if (fallback.url) {
        window.location.href = fallback.url
      }
      return
    }

    let session: PreparedOAuthSession
    try {
      session = await opts.prepare(true)
      if (!session.url) throw new Error('URL de autenticação indisponível')
    } catch (err) {
      try { popup.close() } catch {}
      throw err
    }

    return new Promise<void>((resolve) => {
      let settled = false
      let pollTimer: ReturnType<typeof setInterval> | null = null
      let bc: BroadcastChannel | null = null

      const cleanup = () => {
        if (pollTimer) clearInterval(pollTimer)
        pollTimer = null
        window.removeEventListener('message', onWindowMessage)
        window.removeEventListener('storage', onStorageEvent)
        if (bc) {
          try { bc.close() } catch {}
          bc = null
        }
      }

      const handlePayload = async (data: unknown) => {
        if (settled || !data || typeof data !== 'object') return
        const p = data as Partial<OAuthCompletePayload>
        if (p.type !== 'trackfind:oauth-complete' || p.channel !== opts.channel) return
        if (p.accountId && session.accountId && p.accountId !== session.accountId) return

        settled = true
        cleanup()
        try { popup.close() } catch {}

        const full: OAuthCompletePayload = {
          type: 'trackfind:oauth-complete',
          channel: opts.channel,
          status: p.status === 'success' ? 'success' : 'error',
          accountId: p.accountId || session.accountId,
          message: p.message || '',
          ts: p.ts || Date.now(),
        }

        if (full.status === 'success') {
          await opts.onSuccess(full)
        } else {
          await opts.onError(
            full.message || 'Autenticação não concluída',
            session.accountId,
            Boolean(session.createdTemp),
          )
        }
        resolve()
      }

      const onWindowMessage = (ev: MessageEvent) => {
        if (ev.origin && ev.origin !== window.location.origin) return
        void handlePayload(ev.data)
      }

      const onStorageEvent = (ev: StorageEvent) => {
        if (ev.key !== 'trackfind_oauth_event' || !ev.newValue) return
        try {
          void handlePayload(JSON.parse(ev.newValue))
        } catch {}
      }

      window.addEventListener('message', onWindowMessage)
      window.addEventListener('storage', onStorageEvent)
      try {
        if (typeof BroadcastChannel !== 'undefined') {
          bc = new BroadcastChannel('trackfind_oauth')
          bc.onmessage = (ev) => { void handlePayload(ev.data) }
        }
      } catch {}

      popup.location.href = session.url

      pollTimer = setInterval(() => {
        let isClosed = false
        try { isClosed = popup.closed } catch { isClosed = false }
        if (!isClosed || settled) return

        setTimeout(async () => {
          if (settled) return
          settled = true
          cleanup()
          await opts.onManualClose(session.accountId, Boolean(session.createdTemp))
          resolve()
        }, 350)
      }, 450)
    })
  }

  return { runOAuthPopup }
}
