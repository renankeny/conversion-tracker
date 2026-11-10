<template>
  <section class="branding-tab">
    <BrandingLivePreview
      :draft="draft"
      :saving="saving"
      @save="onSave"
    />

    <BrandingIdentityCard :draft="draft" />

    <BrandingColorsCard
      :draft="draft"
      @reset-defaults="onResetDefaults"
    />
  </section>
</template>

<script setup lang="ts">
import type { PlatformBranding } from '#shared/types'
import { DEFAULT_BRANDING, isValidHexColor, normalizeHexColor } from '~/utils/colorPalette'
import BrandingLivePreview from '~/components/settings/branding/BrandingLivePreview.vue'
import BrandingIdentityCard from '~/components/settings/branding/BrandingIdentityCard.vue'
import BrandingColorsCard from '~/components/settings/branding/BrandingColorsCard.vue'

const { branding, saveBranding, previewDraft, revertPreview } = useBranding()
const toast = useToast()
const saving = ref(false)

const cloneBranding = (src: PlatformBranding): PlatformBranding => ({
  platform_name: src.platform_name,
  tagline: src.tagline,
  footer_text: src.footer_text,
  logo_display_mode: src.logo_display_mode,
  logo_dark_url: src.logo_dark_url,
  logo_light_url: src.logo_light_url,
  favicon_url: src.favicon_url,
  theme_config: {
    preset: src.theme_config?.preset || DEFAULT_BRANDING.theme_config.preset,
    dark: {
      brand: normalizeHexColor(src.theme_config?.dark?.brand || DEFAULT_BRANDING.theme_config.dark.brand),
      surface_1: normalizeHexColor(src.theme_config?.dark?.surface_1 || DEFAULT_BRANDING.theme_config.dark.surface_1),
      surface_2: normalizeHexColor(src.theme_config?.dark?.surface_2 || DEFAULT_BRANDING.theme_config.dark.surface_2),
      surface_deep: normalizeHexColor(src.theme_config?.dark?.surface_deep || DEFAULT_BRANDING.theme_config.dark.surface_deep),
    },
    light: {
      brand: normalizeHexColor(src.theme_config?.light?.brand || DEFAULT_BRANDING.theme_config.light.brand),
      surface_1: normalizeHexColor(src.theme_config?.light?.surface_1 || DEFAULT_BRANDING.theme_config.light.surface_1),
      surface_2: normalizeHexColor(src.theme_config?.light?.surface_2 || DEFAULT_BRANDING.theme_config.light.surface_2),
      surface_deep: normalizeHexColor(src.theme_config?.light?.surface_deep || DEFAULT_BRANDING.theme_config.light.surface_deep),
    },
  },
})

const draft = reactive<PlatformBranding>(cloneBranding(branding.value))

watch(
  draft,
  (val) => {
    previewDraft(cloneBranding(val))
  },
  { deep: true },
)

onBeforeUnmount(() => {
  revertPreview()
})

const onResetDefaults = () => {
  draft.theme_config.preset = DEFAULT_BRANDING.theme_config.preset
  draft.theme_config.dark = { ...DEFAULT_BRANDING.theme_config.dark }
  draft.theme_config.light = { ...DEFAULT_BRANDING.theme_config.light }
  toast.info('Cores restauradas', 'Clique em Salvar personalização para confirmar')
}

const onSave = async () => {
  if (!draft.platform_name.trim()) {
    toast.error('Campo obrigatório', 'Informe o nome da plataforma')
    return
  }

  // Validação explícita de cores antes de enviar
  const palettes = [
    { name: 'Tema Escuro', p: draft.theme_config.dark },
    { name: 'Tema Claro', p: draft.theme_config.light },
  ]
  for (const { name, p } of palettes) {
    for (const [k, v] of Object.entries(p)) {
      if (v && !isValidHexColor(String(v))) {
        toast.error('Cor inválida', `${name}: a cor "${k}" deve estar no formato #RRGGBB (ex: #1C87C1)`)
        return
      }
    }
  }

  saving.value = true
  try {
    await saveBranding(cloneBranding(draft))
    toast.success('Whitelabel atualizado', 'Identidade visual e cores salvas com sucesso')
  } catch (err: any) {
    toast.error('Erro ao salvar', err?.message || 'Não foi possível salvar o branding')
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.branding-tab {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}
</style>
