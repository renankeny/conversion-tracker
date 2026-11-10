<template>
  <div class="identity-card">
    <div class="section-header">
      <h3 class="section-title">Identidade Visual & Logos</h3>
      <p class="section-subtitle">
        Defina o nome da plataforma, rodapé e envie as logos que aparecem no login, sidebar e páginas públicas.
      </p>
    </div>

    <div class="fields-grid">
      <BaseInput
        v-model="draft.platform_name"
        label="Nome da Plataforma"
        placeholder="Ex: MinhaMarca Track"
        required
      />
      <BaseInput
        v-model="draft.tagline"
        label="Subtítulo (Tagline)"
        placeholder="Ex: Server-Side Tracking & CAPI"
      />
      <BaseInput
        v-model="draft.footer_text"
        label="Texto de Rodapé (Opcional)"
        placeholder="Deixe vazio para gerar automaticamente com o ano atual"
      />
      <div class="form-field">
        <BaseDropdown
          v-model="draft.logo_display_mode"
          label="Modo de Exibição da Logo"
          :options="displayOptions"
        />
      </div>
    </div>

    <div class="images-stack">
      <BrandingImageField
        v-model="draft.logo_dark_url"
        label="Logo — Tema Escuro"
        hint="Exibida sobre fundos escuros. Recomendado: PNG ou SVG transparente com texto claro."
        kind="logo"
        preview-theme="dark"
      />

      <BrandingImageField
        v-model="draft.logo_light_url"
        label="Logo — Tema Claro"
        hint="Exibida quando o usuário ativa o tema claro. Se vazio, usa a logo do tema escuro."
        kind="logo"
        preview-theme="light"
      />

      <BrandingImageField
        v-model="draft.favicon_url"
        label="Favicon (Ícone da Aba do Navegador)"
        hint="Quadrado (ex: 64x64 ou 128x128px) em PNG, SVG ou ICO."
        kind="favicon"
        preview-theme="neutral"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import type { PlatformBranding } from '#shared/types'
import BrandingImageField from '~/components/settings/branding/BrandingImageField.vue'

defineProps<{
  draft: PlatformBranding
}>()

const displayOptions = [
  { value: 'icon_text', label: 'Ícone padrão + Nome em texto' },
  { value: 'logo_and_text', label: 'Logo enviada + Nome em texto' },
  { value: 'logo_only', label: 'Somente a Logo enviada' },
]
</script>

<style scoped>
.identity-card {
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 1.5rem;
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.section-title {
  font-family: 'Instrument Sans', sans-serif;
  font-size: var(--fs-xl);
  font-weight: var(--fw-semibold);
  color: var(--text-strong);
  margin-bottom: 0.25rem;
}

.section-subtitle {
  font-size: var(--fs-md);
  color: var(--text-2);
  line-height: 1.45;
}

.fields-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: 1rem;
}

.form-field {
  display: flex;
  flex-direction: column;
}

.images-stack {
  display: flex;
  flex-direction: column;
  gap: 0.875rem;
}
</style>
