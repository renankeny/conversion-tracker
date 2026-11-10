import { watch } from 'vue'
import { useBrandingStore } from '~/stores/useBrandingStore'
import { useThemeStore } from '~/stores/useThemeStore'

// Plugin universal (SSR + Client) que carrega a identidade Whitelabel e sincroniza
// o tema visual em um único watcher global.
export default defineNuxtPlugin({
  name: 'branding',
  dependsOn: ['pinia', 'api'],
  async setup() {
    const brandingStore = useBrandingStore()
    const themeStore = useThemeStore()

    if (import.meta.server) {
      await brandingStore.fetchBranding()
      return
    }

    brandingStore.hydrateFromLocalCache()
    themeStore.applyTheme()
    brandingStore.applyBrandingVisuals()

    watch(
      () => themeStore.theme,
      (nextTheme) => {
        themeStore.applyTheme(nextTheme)
        brandingStore.applyBrandingVisuals()
      },
    )

    if (!brandingStore.loaded) {
      await brandingStore.fetchBranding()
    }
  },
})
