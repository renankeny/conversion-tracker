<template>
  <form class="step-form" @submit.prevent="emit('finish')">
    <div class="step-intro">
      <h2 class="step-title">Identidade Visual (Whitelabel)</h2>
      <p class="step-desc">
        Deixe a plataforma com a sua marca. Observe o painel ao lado mudando em tempo real conforme você personaliza.
      </p>
    </div>

    <div class="fields-stack">
      <div class="grid-two">
        <BaseInput
          v-model="brandingDraft.platform_name"
          label="Nome da Plataforma"
          placeholder="Ex: MinhaMarca Track"
          required
        />
        <BaseInput
          v-model="brandingDraft.tagline"
          label="Subtítulo (Tagline)"
          placeholder="Ex: Intelligence & Attribution"
        />
      </div>

      <div class="presets-block">
        <span class="block-label">Tema de Cores</span>
        <div class="presets-grid">
          <button
            v-for="preset in BRANDING_PRESETS"
            :key="preset.id"
            type="button"
            class="preset-chip"
            :class="{ active: brandingDraft.theme_config.preset === preset.id }"
            @click="selectPreset(preset)"
          >
            <span class="preset-swatch" :style="{ backgroundColor: preset.dark.brand }" />
            <span>{{ preset.name }}</span>
          </button>
        </div>
      </div>

      <BrandingImageField
        v-model="brandingDraft.logo_dark_url"
        label="Logo da Plataforma (Opcional)"
        hint="Envie PNG/SVG transparente ou deixe em branco para usar o ícone automático com a inicial da marca."
        kind="logo"
        preview-theme="dark"
        @update:model-value="onLogoUpdate"
      />
    </div>

    <div class="step-actions">
      <BaseButton type="button" variant="ghost" :disabled="submitting" @click="emit('back')">
        ← Voltar
      </BaseButton>
      <BaseButton type="submit" variant="primary" :loading="submitting">
        Concluir e Acessar Painel
      </BaseButton>
    </div>
  </form>
</template>

<script setup lang="ts">
import type { BrandingPreset, PlatformBranding } from '#shared/types'
import { BRANDING_PRESETS } from '~/utils/colorPalette'
import BrandingImageField from '~/components/settings/branding/BrandingImageField.vue'

const props = defineProps<{
  brandingDraft: PlatformBranding
  submitting: boolean
}>()

const emit = defineEmits<{
  back: []
  finish: []
}>()

const selectPreset = (preset: BrandingPreset) => {
  props.brandingDraft.theme_config.preset = preset.id
  props.brandingDraft.theme_config.dark = { ...preset.dark }
  props.brandingDraft.theme_config.light = { ...preset.light }
}

const onLogoUpdate = (val: string) => {
  if (val && props.brandingDraft.logo_display_mode === 'icon_text') {
    props.brandingDraft.logo_display_mode = 'logo_and_text'
  } else if (!val) {
    props.brandingDraft.logo_display_mode = 'icon_text'
  }
}
</script>

<style scoped>
.step-form { display: flex; flex-direction: column; gap: 1.25rem; }
.step-intro { display: flex; flex-direction: column; gap: 0.375rem; }
.step-title { font-family: 'Instrument Sans', sans-serif; font-size: var(--fs-2xl); font-weight: var(--fw-bold); color: var(--text-strong); }
.step-desc { font-size: var(--fs-md); color: var(--text-2); line-height: 1.5; }
.fields-stack { display: flex; flex-direction: column; gap: 1rem; }
.grid-two { display: grid; grid-template-columns: 1fr 1fr; gap: 0.75rem; }
@media (max-width: 520px) { .grid-two { grid-template-columns: 1fr; } }
.presets-block { display: flex; flex-direction: column; gap: 0.5rem; }
.block-label { font-size: var(--fs-sm); font-weight: var(--fw-semibold); color: var(--text-2); }
.presets-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(135px, 1fr)); gap: 0.5rem; }
.preset-chip { display: flex; align-items: center; gap: 0.5rem; padding: 0.5rem 0.75rem; border-radius: 8px; border: 1px solid var(--border); background: var(--surface-soft-2); color: var(--text-1); font-size: var(--fs-base); font-weight: var(--fw-medium); cursor: pointer; transition: all 0.15s ease; }
.preset-chip:hover { border-color: var(--brand-border); }
.preset-chip.active { border-color: var(--brand); background: var(--brand-soft); color: var(--brand); }
.preset-swatch { width: 14px; height: 14px; border-radius: 50%; border: 1px solid var(--border); flex-shrink: 0; }
.step-actions { display: flex; justify-content: space-between; align-items: center; gap: 0.75rem; padding-top: 0.25rem; }
</style>

