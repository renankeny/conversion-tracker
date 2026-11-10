import { storeToRefs } from 'pinia'
import { useBrandingStore } from '~/stores/useBrandingStore'

export const useBranding = () => {
  const store = useBrandingStore()
  const {
    branding,
    brandName,
    tagline,
    footerText,
    activeLogoUrl,
    loaded,
  } = storeToRefs(store)

  return {
    branding,
    brandName,
    tagline,
    footerText,
    activeLogoUrl,
    loaded,
    fetchBranding: store.fetchBranding,
    saveBranding: store.saveBranding,
    applyBrandingVisuals: store.applyBrandingVisuals,
    previewDraft: store.previewDraft,
    revertPreview: store.revertPreview,
    formatTitle: store.formatTitle,
  }
}
