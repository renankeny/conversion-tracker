<template>
  <div class="colors-card">
    <div class="section-header">
      <div>
        <h3 class="section-title">Paleta de Cores (Dark & Light)</h3>
        <p class="section-subtitle">
          Escolha um preset pronto ou personalize cada cor manualmente. Tons suaves, bordas e contraste de texto são calculados automaticamente.
        </p>
      </div>
      <BaseButton variant="ghost" size="sm" type="button" @click="emit('reset-defaults')">
        Restaurar padrão
      </BaseButton>
    </div>

    <div class="presets-block">
      <span class="block-label">Presets Rápidos</span>
      <div class="presets-grid">
        <button
          v-for="preset in BRANDING_PRESETS"
          :key="preset.id"
          type="button"
          class="preset-chip"
          :class="{ active: draft.theme_config.preset === preset.id }"
          @click="applyPreset(preset)"
        >
          <span class="preset-swatch" :style="{ backgroundColor: preset.dark.brand }" />
          <span class="preset-name">{{ preset.name }}</span>
        </button>
      </div>
    </div>

    <div class="themes-grid">
      <div v-for="theme in themeBlocks" :key="theme.mode" class="theme-column">
        <div class="theme-header">
          <span class="theme-title">{{ theme.title }}</span>
          <BaseBadge :variant="theme.badgeVariant">{{ theme.badgeLabel }}</BaseBadge>
        </div>

        <div class="color-list">
          <div v-for="field in colorFields" :key="`${theme.mode}-${field.key}`" class="color-row">
            <div class="color-meta">
              <span class="color-label">{{ field.label }}</span>
              <span class="color-hint">{{ field.hint }}</span>
            </div>
            <div class="color-inputs">
              <input
                v-model="draft.theme_config[theme.paletteKey][field.key]"
                type="color"
                class="color-picker"
                @input="draft.theme_config.preset = 'custom'"
              />
              <input
                v-model="draft.theme_config[theme.paletteKey][field.key]"
                type="text"
                maxlength="7"
                class="color-hex"
                @input="draft.theme_config.preset = 'custom'"
              />
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { BrandingPreset, PlatformBranding, ThemePalette } from '#shared/types'
import { BRANDING_PRESETS } from '~/utils/colorPalette'

const props = defineProps<{
  draft: PlatformBranding
}>()

const emit = defineEmits<{
  'reset-defaults': []
}>()

const themeBlocks = [
  { mode: 'dark', title: 'Tema Escuro (Dark)', badgeLabel: 'Padrão', badgeVariant: 'info' as const, paletteKey: 'dark' as const },
  { mode: 'light', title: 'Tema Claro (Light)', badgeLabel: 'Alternativo', badgeVariant: 'default' as const, paletteKey: 'light' as const },
]

const colorFields: { key: keyof ThemePalette; label: string; hint: string }[] = [
  { key: 'brand', label: 'Cor Principal (Brand)', hint: 'Botões primários, links ativos e destaques' },
  { key: 'surface_1', label: 'Fundo da Página', hint: 'Cor de fundo geral do painel' },
  { key: 'surface_2', label: 'Superfície de Cards', hint: 'Fundo de cartões, tabelas e sidebar' },
  { key: 'surface_deep', label: 'Superfície Profunda', hint: 'Fundo de inputs, blocos de código e modais' },
]

const applyPreset = (preset: BrandingPreset) => {
  props.draft.theme_config.preset = preset.id
  props.draft.theme_config.dark = { ...preset.dark }
  props.draft.theme_config.light = { ...preset.light }
}
</script>

<style scoped>
.colors-card { background: var(--surface-2); border: 1px solid var(--border); border-radius: 12px; padding: 1.5rem; display: flex; flex-direction: column; gap: 1.25rem; }
.section-header { display: flex; justify-content: space-between; align-items: flex-start; gap: 0.75rem; flex-wrap: wrap; }
.section-title { font-family: 'Instrument Sans', sans-serif; font-size: var(--fs-xl); font-weight: var(--fw-semibold); color: var(--text-strong); margin-bottom: 0.25rem; }
.section-subtitle { font-size: var(--fs-md); color: var(--text-2); }
.presets-block { display: flex; flex-direction: column; gap: 0.5rem; }
.block-label { font-size: var(--fs-sm); font-weight: var(--fw-semibold); text-transform: uppercase; letter-spacing: 0.05em; color: var(--text-3); }
.presets-grid { display: flex; flex-wrap: wrap; gap: 0.5rem; }
.preset-chip { display: inline-flex; align-items: center; gap: 0.5rem; padding: 0.5rem 0.875rem; border-radius: 999px; border: 1px solid var(--border); background: var(--surface-soft-2); color: var(--text-1); font-size: var(--fs-base); font-weight: var(--fw-medium); cursor: pointer; transition: all 0.15s ease; }
.preset-chip:hover { border-color: var(--brand-border); }
.preset-chip.active { border-color: var(--brand); background: var(--brand-soft); color: var(--brand); }
.preset-swatch { width: 14px; height: 14px; border-radius: 50%; border: 1px solid var(--border); }
.themes-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(280px, 1fr)); gap: 1rem; }
.theme-column { padding: 1rem; border: 1px solid var(--border-soft); border-radius: 10px; background: var(--surface-soft-2); display: flex; flex-direction: column; gap: 0.875rem; }
.theme-header { display: flex; justify-content: space-between; align-items: center; }
.theme-title { font-size: var(--fs-md); font-weight: var(--fw-semibold); color: var(--text-strong); }
.color-list { display: flex; flex-direction: column; gap: 0.75rem; }
.color-row { display: flex; justify-content: space-between; align-items: center; gap: 0.75rem; }
.color-meta { display: flex; flex-direction: column; gap: 0.125rem; }
.color-label { font-size: var(--fs-base); font-weight: var(--fw-medium); color: var(--text-1); }
.color-hint { font-size: var(--fs-xs); color: var(--text-3); }
.color-inputs { display: flex; align-items: center; gap: 0.375rem; }
.color-picker { width: 32px; height: 32px; padding: 0; border: 1px solid var(--border); border-radius: 6px; background: transparent; cursor: pointer; }
.color-hex { width: 82px; padding: 0.375rem 0.5rem; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: var(--fs-sm); color: var(--text-1); background: var(--surface-deep); border: 1px solid var(--border); border-radius: 6px; text-transform: uppercase; }
</style>
