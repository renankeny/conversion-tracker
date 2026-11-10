<template>
  <div class="error-screen">
    <div class="error-glow" aria-hidden="true" />

    <main class="error-card">
      <div class="brand-wrapper">
        <BrandLogo size="lg" show-tagline />
      </div>

      <div class="status-pill" :class="{ 'is-server-error': !isNotFound }">
        <Compass v-if="isNotFound" :size="15" />
        <AlertTriangle v-else :size="15" />
        <span>{{ statusBadgeText }}</span>
      </div>

      <div class="code-Watermark">{{ statusCode }}</div>

      <h1 class="error-title">{{ errorTitle }}</h1>
      <p class="error-description">{{ errorDescription }}</p>

      <div class="error-actions">
        <BaseButton v-if="!isNotFound" variant="primary" @click="retryPage">
          <template #iconLeft>
            <RefreshCw :size="15" />
          </template>
          Tentar novamente
        </BaseButton>

        <BaseButton :variant="isNotFound ? 'primary' : 'secondary'" @click="goHome">
          <template #iconLeft>
            <Home :size="15" />
          </template>
          Ir para o painel principal
        </BaseButton>

        <BaseButton v-if="isNotFound" variant="ghost" @click="goBack">
          <template #iconLeft>
            <ArrowLeft :size="15" />
          </template>
          Voltar
        </BaseButton>
      </div>
    </main>

    <footer class="error-footer">
      {{ footerText }}
    </footer>
  </div>
</template>

<script setup lang="ts">
import '../assets/css/main.css'
import { Compass, AlertTriangle, Home, ArrowLeft, RefreshCw } from 'lucide-vue-next'
import type { NuxtError } from '#app'
import { computePaletteVars } from '~/utils/colorPalette'

const props = defineProps<{
  error: NuxtError
}>()

const { theme } = useTheme()
const { branding, brandName, footerText, fetchBranding, applyBrandingVisuals } = useBranding()

const statusCode = computed(() => Number(props.error?.statusCode || 500))
const isNotFound = computed(() => statusCode.value === 404)

const statusBadgeText = computed(() =>
  isNotFound.value
    ? 'Erro 404 · Página não encontrada'
    : `Erro ${statusCode.value} · Instabilidade interna`,
)

const errorTitle = computed(() =>
  isNotFound.value
    ? 'Página não encontrada'
    : 'Não foi possível concluir sua solicitação',
)

const errorDescription = computed(() =>
  isNotFound.value
    ? 'O endereço que você tentou acessar não existe, foi movido ou não está disponível em nossa plataforma. Verifique se o link está correto ou retorne ao painel principal.'
    : 'Ocorreu uma falha inesperada ao processar esta página no servidor. Tente atualizar a página em alguns instantes ou retorne ao painel principal para continuar.',
)

const activePaletteVars = computed(() => {
  const palette =
    theme.value === 'light'
      ? branding.value.theme_config?.light
      : branding.value.theme_config?.dark
  return palette ? computePaletteVars(palette) : {}
})

const ssrInlineStyle = computed(() =>
  Object.entries(activePaletteVars.value)
    .map(([k, v]) => `${k}: ${v}`)
    .join('; '),
)

const ssrCssVars = computed(() => {
  const decls = ssrInlineStyle.value
  if (!decls) return ''
  return `html:root, html:root[data-theme='dark'], html:root[data-theme='light'] { ${decls}; }`
})

useHead({
  title: computed(() => `${statusCode.value} · ${errorTitle.value} - ${brandName.value}`),
  htmlAttrs: {
    'data-theme': computed(() => theme.value),
    style: ssrInlineStyle,
  },
  link: computed(() =>
    branding.value.favicon_url
      ? [{ rel: 'icon', href: branding.value.favicon_url }]
      : [{ rel: 'icon', type: 'image/x-icon', href: '/favicon.ico' }],
  ),
  style: [{ id: 'whitelabel-ssr-vars', innerHTML: ssrCssVars }],
})

onMounted(async () => {
  applyBrandingVisuals(branding.value)
  await fetchBranding()
})

const goHome = () => clearError({ redirect: '/' })

const goBack = () => {
  if (import.meta.client && window.history.length > 1) {
    window.history.back()
  } else {
    clearError({ redirect: '/' })
  }
}

const retryPage = () => {
  if (import.meta.client) {
    window.location.reload()
  } else {
    clearError({ redirect: '/' })
  }
}
</script>

<style scoped>
.error-screen {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 2rem 1.25rem;
  background: var(--surface-1);
  color: var(--text-1);
  position: relative;
  overflow: hidden;
}
.error-glow {
  position: absolute;
  top: -15%;
  left: 50%;
  transform: translateX(-50%);
  width: 620px;
  height: 360px;
  background: radial-gradient(circle, var(--brand-soft) 0%, transparent 70%);
  pointer-events: none;
  z-index: 0;
}
.error-card {
  position: relative;
  z-index: 1;
  width: 100%;
  max-width: 520px;
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: 16px;
  padding: 2.5rem 2rem;
  box-shadow: var(--shadow-card);
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
}
.brand-wrapper {
  margin-bottom: 1.5rem;
}
.status-pill {
  display: inline-flex;
  align-items: center;
  gap: 0.4375rem;
  padding: 0.375rem 0.875rem;
  border-radius: 999px;
  background: var(--brand-soft);
  border: 1px solid var(--brand-border);
  color: var(--brand);
  font-size: var(--fs-xs);
  font-weight: var(--fw-semibold);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin-bottom: 1rem;
}
.status-pill.is-server-error {
  background: var(--brand-soft);
  border-color: var(--brand-border);
  color: var(--brand);
}
.code-Watermark {
  font-family: var(--font-heading);
  font-size: var(--fs-3xl);
  font-weight: var(--fw-bold);
  color: var(--brand);
  letter-spacing: -0.04em;
  line-height: 1;
  margin-bottom: 0.5rem;
}
.error-title {
  font-family: var(--font-heading);
  font-size: var(--fs-2xl);
  font-weight: var(--fw-bold);
  color: var(--text-strong);
  margin-bottom: 0.625rem;
}
.error-description {
  font-size: var(--fs-md);
  color: var(--text-2);
  line-height: 1.6;
  max-width: 440px;
  margin-bottom: 1.75rem;
}
.error-actions {
  display: flex;
  align-items: center;
  justify-content: center;
  flex-wrap: wrap;
  gap: 0.625rem;
  width: 100%;
}
.error-footer {
  position: relative;
  z-index: 1;
  margin-top: 1.75rem;
  font-size: var(--fs-xs);
  color: var(--text-3);
  text-align: center;
}
</style>

