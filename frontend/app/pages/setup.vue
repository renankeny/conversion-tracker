<template>
  <div v-if="checking" class="setup-loading">
    <div class="spinner-brand" />
    <p class="loading-text">Preparando assistente de configuração...</p>
  </div>

  <div v-else class="setup-page-wrapper">
    <div class="hero-column">
      <SetupHero :current-step="currentStep" />
    </div>

    <div class="wizard-column">
      <div class="wizard-card">
        <div class="wizard-top">
          <div class="wizard-heading">
            <span class="wizard-eyebrow">Bem-vindo</span>
            <h1 class="wizard-main-title">Configure sua plataforma</h1>
          </div>
          <span class="setup-badge">Primeiro Acesso</span>
        </div>

        <div class="stepper-bar">
          <div
            v-for="s in steps"
            :key="s.num"
            class="step-item"
            :class="{ active: currentStep === s.num, done: currentStep > s.num }"
          >
            <span class="step-circle">{{ currentStep > s.num ? '✓' : s.num }}</span>
            <span class="step-label">{{ s.label }}</span>
          </div>
        </div>

        <SetupAdminStep
          v-if="currentStep === 1"
          :model="form"
          @next="currentStep = 2"
        />

        <SetupCompanyStep
          v-else-if="currentStep === 2"
          :model="form"
          @back="currentStep = 1"
          @next="currentStep = 3"
        />

        <SetupBrandingStep
          v-else
          :branding-draft="brandingDraft"
          :submitting="submitting"
          @back="currentStep = 2"
          @finish="onCompleteSetup"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { PlatformBranding } from '#shared/types'
import { DEFAULT_BRANDING } from '~/utils/colorPalette'
import { errorMessage } from '~/utils/errorMessage'
import SetupHero from '~/components/setup/SetupHero.vue'
import SetupAdminStep from '~/components/setup/SetupAdminStep.vue'
import SetupCompanyStep from '~/components/setup/SetupCompanyStep.vue'
import SetupBrandingStep from '~/components/setup/SetupBrandingStep.vue'

definePageMeta({
  layout: false,
})

const { branding, applyBrandingVisuals } = useBranding()
const { checkSetupStatus, completeSetup } = useSetup()
const toast = useToast()

const checking = ref(true)
const submitting = ref(false)
const currentStep = ref(1)

const steps = [
  { num: 1, label: 'Super Admin' },
  { num: 2, label: 'Operação' },
  { num: 3, label: 'Sua Marca' },
]

const form = reactive({
  admin_name: '',
  admin_email: '',
  admin_phone: '',
  admin_password: '',
  account_name: 'Operação Principal',
})

const initialName =
  branding.value.platform_name && branding.value.platform_name !== 'TrackFind' && branding.value.platform_name !== 'Sua Plataforma'
    ? branding.value.platform_name
    : ''

const brandingDraft = reactive<PlatformBranding>({
  platform_name: initialName,
  tagline: branding.value.tagline || DEFAULT_BRANDING.tagline,
  footer_text: branding.value.footer_text || '',
  logo_display_mode: branding.value.logo_display_mode || 'icon_text',
  logo_dark_url: branding.value.logo_dark_url || '',
  logo_light_url: branding.value.logo_light_url || '',
  favicon_url: branding.value.favicon_url || '',
  theme_config: {
    preset: branding.value.theme_config?.preset || DEFAULT_BRANDING.theme_config.preset,
    dark: { ...DEFAULT_BRANDING.theme_config.dark, ...(branding.value.theme_config?.dark || {}) },
    light: { ...DEFAULT_BRANDING.theme_config.light, ...(branding.value.theme_config?.light || {}) },
  },
})

useHead({
  title: computed(() =>
    brandingDraft.platform_name.trim()
      ? `Configuração Inicial - ${brandingDraft.platform_name.trim()}`
      : 'Configuração Inicial - Sua Plataforma'
  ),
})

const syncPreviewBranding = () => {
  branding.value = {
    ...brandingDraft,
    platform_name: brandingDraft.platform_name.trim() || 'Sua Plataforma',
    theme_config: {
      preset: brandingDraft.theme_config.preset,
      dark: { ...brandingDraft.theme_config.dark },
      light: { ...brandingDraft.theme_config.light },
    },
  }
  applyBrandingVisuals(branding.value)
}

watch(brandingDraft, syncPreviewBranding, { deep: true })

onMounted(async () => {
  const mustSetup = await checkSetupStatus(true)
  if (!mustSetup) {
    await navigateTo('/login')
    return
  }
  syncPreviewBranding()
  checking.value = false
})

const onCompleteSetup = async () => {
  if (!brandingDraft.platform_name.trim()) {
    toast.error('Campo obrigatório', 'Informe o nome da sua plataforma')
    return
  }
  submitting.value = true
  try {
    await completeSetup({
      admin_name: form.admin_name,
      admin_email: form.admin_email,
      admin_phone: form.admin_phone,
      admin_password: form.admin_password,
      account_name: form.account_name,
      branding: {
        ...branding.value,
        platform_name: brandingDraft.platform_name.trim(),
      },
    })
    toast.success('Plataforma configurada!', 'Bem-vindo ao painel Super Admin.')
    if (import.meta.client) {
      window.location.href = '/'
      return
    }
    await navigateTo('/', { replace: true })
  } catch (err: unknown) {
    toast.error('Falha na configuração', errorMessage(err, 'Não foi possível concluir o setup'))
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
.setup-loading { height: 100vh; width: 100vw; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 0.75rem; background: var(--surface-1); }
.spinner-brand { width: 32px; height: 32px; border: 3px solid var(--border); border-top-color: var(--brand); border-radius: 50%; animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
.loading-text { font-size: var(--fs-base); color: var(--text-3); font-weight: var(--fw-medium); }
.setup-page-wrapper { min-height: 100vh; width: 100vw; display: flex; background: var(--surface-1); color: var(--text-1); overflow-x: hidden; }
.hero-column { display: none; }
@media (min-width: 900px) { .hero-column { display: block; width: 48%; height: 100vh; overflow: hidden; } }
.wizard-column { width: 100%; display: flex; flex-direction: column; align-items: center; justify-content: center; padding: 2rem 1.5rem; min-height: 100vh; overflow-y: auto; }
@media (min-width: 900px) { .wizard-column { width: 52%; padding: 3rem; } }
.wizard-card { width: 100%; max-width: 500px; background: var(--surface-2); border: 1px solid var(--border); border-radius: 16px; padding: 2rem; box-shadow: var(--shadow-card); display: flex; flex-direction: column; gap: 1.5rem; }
.wizard-top { display: flex; justify-content: space-between; align-items: flex-start; gap: 0.75rem; flex-wrap: wrap; }
.wizard-heading { display: flex; flex-direction: column; gap: 0.125rem; }
.wizard-eyebrow { font-size: var(--fs-sm); font-weight: var(--fw-semibold); color: var(--brand); text-transform: uppercase; letter-spacing: 0.05em; }
.wizard-main-title { font-family: var(--font-sans); font-size: var(--fs-2xl); font-weight: var(--fw-bold); color: var(--text-strong); }
.setup-badge { font-size: var(--fs-xs); font-weight: var(--fw-semibold); text-transform: uppercase; letter-spacing: 0.06em; padding: 0.25rem 0.625rem; border-radius: 999px; background: var(--brand-soft); color: var(--brand); border: 1px solid var(--brand-border); }
.stepper-bar { display: grid; grid-template-columns: repeat(3, 1fr); gap: 0.5rem; padding-bottom: 1rem; border-bottom: 1px solid var(--border-soft); }
.step-item { display: flex; align-items: center; gap: 0.5rem; font-size: var(--fs-base); color: var(--text-3); font-weight: var(--fw-medium); }
.step-item.active { color: var(--text-strong); font-weight: var(--fw-semibold); }
.step-item.done { color: var(--brand); }
.step-circle { width: 22px; height: 22px; border-radius: 50%; display: inline-flex; align-items: center; justify-content: center; font-size: var(--fs-sm); font-weight: var(--fw-bold); border: 1px solid var(--border); background: var(--surface-deep); flex-shrink: 0; }
.step-item.active .step-circle { border-color: var(--brand); background: var(--brand); color: var(--brand-text); }
.step-item.done .step-circle { border-color: var(--brand); background: var(--brand-soft); color: var(--brand); }
</style>
