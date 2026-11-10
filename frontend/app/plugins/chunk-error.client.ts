// Plugin client-only que recarrega a página quando um chunk JS/CSS antigo
// falha ao carregar. Isso acontece quando o servidor faz um novo build
// (deploy ou HMR em dev) e o cliente ainda tem a versão antiga em memória,
// tentando buscar arquivos com hash que já não existem.
//
// Sintomas típicos:
//   - "Unable to preload CSS for /assets/Term-kHJ-D0s7.css"
//   - "Failed to fetch dynamically imported module"
//   - ChunkLoadError
export default defineNuxtPlugin((nuxtApp) => {
  if (!import.meta.client) return

  // Evita loop infinito de reload se o erro persistir.
  const RELOAD_KEY = 'tf_chunk_reload_at'
  const RELOAD_COOLDOWN_MS = 10_000

  const shouldReload = () => {
    const last = Number(sessionStorage.getItem(RELOAD_KEY) || 0)
    return Date.now() - last > RELOAD_COOLDOWN_MS
  }

  const reloadNow = () => {
    sessionStorage.setItem(RELOAD_KEY, String(Date.now()))
    window.location.reload()
  }

  const isChunkError = (err: unknown) => {
    if (!err) return false
    const msg = err instanceof Error ? err.message : String(err)
    return (
      /Unable to preload CSS/i.test(msg) ||
      /Failed to fetch dynamically imported module/i.test(msg) ||
      /Loading chunk \d+ failed/i.test(msg) ||
      /ChunkLoadError/i.test(msg) ||
      /error loading dynamically imported module/i.test(msg)
    )
  }

  // Captura erros não tratados de promise (onde os chunk errors caem).
  window.addEventListener('unhandledrejection', (event) => {
    if (isChunkError(event.reason) && shouldReload()) {
      reloadNow()
    }
  })

  // Captura erros síncronos.
  window.addEventListener('error', (event) => {
    if (isChunkError(event.error) && shouldReload()) {
      reloadNow()
    }
  })

  // Hook do Nuxt: erro durante navegação por rota.
  nuxtApp.hook('vue:error', (err) => {
    if (isChunkError(err) && shouldReload()) {
      reloadNow()
    }
  })

  nuxtApp.hook('app:chunkError', () => {
    if (shouldReload()) reloadNow()
  })
})
