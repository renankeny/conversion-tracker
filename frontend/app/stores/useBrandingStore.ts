import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import type { PlatformBranding } from '#shared/types'
import { applyPaletteToDocument, DEFAULT_BRANDING, normalizeHexColor } from '~/utils/colorPalette'
import { useThemeStore } from '~/stores/useThemeStore'

const STORAGE_KEY = 'wl_branding_cache_v2'

export function clonePlatformBranding(src: PlatformBranding): PlatformBranding {
  const darkFallback = DEFAULT_BRANDING.theme_config.dark
  const lightFallback = DEFAULT_BRANDING.theme_config.light
  const darkSrc = src.theme_config?.dark || {}
  const lightSrc = src.theme_config?.light || {}

  return {
    platform_name: src.platform_name === 'TrackFind' ? 'Sua Plataforma' : (src.platform_name || 'Sua Plataforma'),
    tagline: src.tagline || DEFAULT_BRANDING.tagline,
    footer_text: src.footer_text || '',
    logo_display_mode: src.logo_display_mode || DEFAULT_BRANDING.logo_display_mode,
    logo_dark_url: src.logo_dark_url || '',
    logo_light_url: src.logo_light_url || '',
    favicon_url: src.favicon_url || '',
    theme_config: {
      preset: src.theme_config?.preset || DEFAULT_BRANDING.theme_config.preset,
      dark: {
        brand: normalizeHexColor(darkSrc.brand || darkFallback.brand, darkFallback.brand),
        surface_1: normalizeHexColor(darkSrc.surface_1 || darkFallback.surface_1, darkFallback.surface_1),
        surface_2: normalizeHexColor(darkSrc.surface_2 || darkFallback.surface_2, darkFallback.surface_2),
        surface_deep: normalizeHexColor(darkSrc.surface_deep || darkFallback.surface_deep, darkFallback.surface_deep),
      },
      light: {
        brand: normalizeHexColor(lightSrc.brand || lightFallback.brand, lightFallback.brand),
        surface_1: normalizeHexColor(lightSrc.surface_1 || lightFallback.surface_1, lightFallback.surface_1),
        surface_2: normalizeHexColor(lightSrc.surface_2 || lightFallback.surface_2, lightFallback.surface_2),
        surface_deep: normalizeHexColor(lightSrc.surface_deep || lightFallback.surface_deep, lightFallback.surface_deep),
      },
    },
  }
}

export const useBrandingStore = defineStore('branding', () => {
  const savedBranding = ref<PlatformBranding>(clonePlatformBranding(DEFAULT_BRANDING))
  const draftBranding = ref<PlatformBranding | null>(null)
  const loaded = ref(false)
  let inFlightPromise: Promise<PlatformBranding> | null = null

  const branding = computed<PlatformBranding>({
    get: () => draftBranding.value ?? savedBranding.value,
    set: (val) => {
      savedBranding.value = clonePlatformBranding(val)
    },
  })

  const brandName = computed(() => branding.value.platform_name?.trim() || 'Sua Plataforma')
  const tagline = computed(() => branding.value.tagline?.trim() || 'Intelligence & Attribution')
  const footerText = computed(() => {
    if (branding.value.footer_text?.trim()) {
      return branding.value.footer_text.trim()
    }
    const year = new Date().getFullYear()
    return `© ${year} ${brandName.value}. Todos os direitos reservados.`
  })

  const activeLogoUrl = computed(() => {
    const themeStore = useThemeStore()
    if (themeStore.isDark) {
      return branding.value.logo_dark_url || branding.value.logo_light_url || ''
    }
    return branding.value.logo_light_url || branding.value.logo_dark_url || ''
  })

  function hydrateFromLocalCache(): void {
    if (!import.meta.client || loaded.value) return
    try {
      localStorage.removeItem('tf_branding_cache_v1')
      const cached = localStorage.getItem(STORAGE_KEY)
      if (cached) {
        savedBranding.value = clonePlatformBranding({ ...DEFAULT_BRANDING, ...JSON.parse(cached) })
      }
    } catch {
      // Ignora erro de parse ou bloqueio de storage
    }
  }

  function persistCache(b: PlatformBranding): void {
    if (!import.meta.client) return
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(b))
    } catch {
      // Ignora limite de quota
    }
  }

  function applyBrandingVisuals(b: PlatformBranding = branding.value): void {
    if (!import.meta.client) return
    const themeStore = useThemeStore()
    const palette = themeStore.theme === 'light' ? b.theme_config.light : b.theme_config.dark
    if (palette) {
      applyPaletteToDocument(palette)
    }
    if (b.favicon_url) {
      let link = document.querySelector("link[rel~='icon']") as HTMLLinkElement | null
      if (!link) {
        link = document.createElement('link')
        link.rel = 'icon'
        document.head.appendChild(link)
      }
      link.href = b.favicon_url
    }
  }

  async function fetchBranding(force = false): Promise<PlatformBranding> {
    if (loaded.value && !force) {
      persistCache(savedBranding.value)
      applyBrandingVisuals(branding.value)
      return branding.value
    }
    if (inFlightPromise && !force) {
      return inFlightPromise
    }

    const api = useAPI()
    inFlightPromise = (async () => {
      try {
        const data = await api.publicGet<PlatformBranding>('/branding')
        if (data && data.platform_name) {
          savedBranding.value = clonePlatformBranding({ ...DEFAULT_BRANDING, ...data })
          loaded.value = true
          persistCache(savedBranding.value)
          applyBrandingVisuals(branding.value)
        }
      } catch {
        applyBrandingVisuals(branding.value)
      } finally {
        inFlightPromise = null
      }
      return branding.value
    })()

    return inFlightPromise
  }

  async function saveBranding(payload: PlatformBranding): Promise<PlatformBranding> {
    const api = useAPI()
    const saved = await api.put<PlatformBranding>('/admin/branding', payload)
    const normalized = clonePlatformBranding(saved)
    savedBranding.value = normalized
    draftBranding.value = null
    loaded.value = true
    persistCache(normalized)
    applyBrandingVisuals(normalized)
    return normalized
  }

  function previewDraft(draft: PlatformBranding): void {
    const cloned = clonePlatformBranding(draft)
    draftBranding.value = cloned
    applyBrandingVisuals(cloned)
  }

  function revertPreview(): void {
    if (!draftBranding.value) return
    draftBranding.value = null
    applyBrandingVisuals(savedBranding.value)
  }

  function formatTitle(pageTitle?: string): string {
    const base = brandName.value || 'Sua Plataforma'
    const cleanTitle = pageTitle?.trim()
    return cleanTitle ? `${cleanTitle} - ${base}` : base
  }

  return {
    savedBranding,
    draftBranding,
    branding,
    brandName,
    tagline,
    footerText,
    activeLogoUrl,
    loaded,
    hydrateFromLocalCache,
    fetchBranding,
    saveBranding,
    applyBrandingVisuals,
    previewDraft,
    revertPreview,
    formatTitle,
  }
})

