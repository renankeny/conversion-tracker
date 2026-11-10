<template>
  <div class="setup-hero-container">
    <div class="ambient-glow glow-top" />
    <div class="ambient-glow glow-bottom" />

    <div class="grid-overlay">
      <svg class="w-full h-full" xmlns="http://www.w3.org/2000/svg">
        <pattern id="setupGrid" width="40" height="40" patternUnits="userSpaceOnUse">
          <path d="M 40 0 L 0 0 0 40" fill="none" stroke="var(--brand-soft)" stroke-width="0.8" stroke-dasharray="3 3" />
        </pattern>
        <rect width="100%" height="100%" fill="url(#setupGrid)" />
      </svg>
    </div>

    <div class="hero-showcase-card">
      <div class="mb-6">
        <BrandLogo size="lg" show-tagline />
      </div>

      <h1 class="hero-headline">
        Construa e lance sua própria plataforma de rastreamento
      </h1>
      <p class="hero-subtext">
        Configure seu acesso mestre, defina sua operação principal e personalize cores, logotipo e domínio 100% com a identidade da sua marca.
      </p>

      <div class="pillars-card">
        <div class="pillars-header">
          <div class="flex items-center gap-2">
            <span class="live-indicator" />
            <span class="pillars-title">Sua Plataforma Whitelabel Pronta</span>
          </div>
          <span class="badge-status">
            <Sparkles :size="12" class="text-brand" /> 100% Personalizável
          </span>
        </div>

        <div class="pillars-content">
          <div v-for="item in pillars" :key="item.title" class="pillar-item" :class="{ highlighted: item.step === currentStep }">
            <div class="item-icon-wrap bg-brand-soft">
              <component :is="item.icon" :size="15" class="text-brand" />
            </div>
            <div class="item-info">
              <div class="flex items-center justify-between">
                <span class="item-title">{{ item.title }}</span>
                <span class="item-step">Etapa {{ item.step }}</span>
              </div>
              <p class="item-desc">{{ item.desc }}</p>
            </div>
          </div>
        </div>
      </div>

      <div class="features-grid">
        <div v-for="feat in features" :key="feat" class="feature-pill">
          <Check :size="13" class="text-brand" />
          <span>{{ feat }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Sparkles, ShieldCheck, Building2, Palette, Check } from 'lucide-vue-next'
import BrandLogo from '~/components/BrandLogo.vue'

defineProps<{
  currentStep?: number
}>()

const pillars = [
  {
    step: 1,
    icon: ShieldCheck,
    title: 'Conta Super Admin Mestre',
    desc: 'Controle total sobre usuários, permissões de equipe e configurações globais.',
  },
  {
    step: 2,
    icon: Building2,
    title: 'Arquitetura Multi-Empresa',
    desc: 'Isole pixels, funis CRM e webhooks por operação ou cliente com segurança.',
  },
  {
    step: 3,
    icon: Palette,
    title: 'Sua Marca, Suas Cores',
    desc: 'Defina o nome da sua plataforma, logotipo próprio e paleta Dark & Light.',
  },
]

const features = ['Sua Marca Própria', 'Multi-Tenant Nativo', 'Server-Side CAPI']
</script>

<style scoped>
.setup-hero-container {
  height: 100vh; width: 100%; display: flex; align-items: center; justify-content: center;
  padding: 2.5rem; position: relative; overflow: hidden; background: var(--surface-deep); color: var(--text-1); user-select: none;
}
.ambient-glow { position: absolute; width: 28rem; height: 28rem; border-radius: 9999px; pointer-events: none; filter: blur(120px); background: var(--brand-soft); }
.glow-top { top: -6rem; left: -6rem; opacity: 0.85; }
.glow-bottom { bottom: -6rem; right: -6rem; opacity: 0.55; }
.grid-overlay { position: absolute; inset: 0; width: 100%; height: 100%; pointer-events: none; }
.hero-showcase-card {
  position: relative; z-index: 10; max-width: 32rem; width: 100%; background: var(--surface-2);
  border: 1px solid var(--border); padding: 2rem; border-radius: 1.5rem; box-shadow: var(--shadow-elevated);
}
.text-brand { color: var(--brand); }
.bg-brand-soft { background: var(--brand-soft); }
.hero-headline { font-size: var(--fs-2xl); font-weight: var(--fw-bold); color: var(--text-strong); line-height: 1.3; margin-bottom: 0.625rem; letter-spacing: -0.02em; }
.hero-subtext { font-size: var(--fs-base); color: var(--text-2); line-height: 1.5; margin-bottom: 1.5rem; }
.pillars-card { background: var(--surface-deep); border: 1px solid var(--border); border-radius: 1rem; padding: 0.875rem; margin-bottom: 1.25rem; }
.pillars-header { display: flex; align-items: center; justify-content: space-between; padding-bottom: 0.625rem; border-bottom: 1px solid var(--border-soft); margin-bottom: 0.625rem; }
.live-indicator { width: 7px; height: 7px; border-radius: 9999px; background: var(--brand); box-shadow: 0 0 8px var(--brand); }
.pillars-title { font-size: var(--fs-xs); font-weight: var(--fw-semibold); color: var(--text-2); text-transform: uppercase; letter-spacing: 0.04em; }
.badge-status { font-size: var(--fs-xs); font-weight: var(--fw-semibold); color: var(--brand); background: var(--brand-soft); padding: 0.15rem 0.5rem; border-radius: 0.375rem; display: flex; align-items: center; gap: 0.25rem; }
.pillars-content { display: flex; flex-direction: column; gap: 0.5rem; }
.pillar-item { display: flex; align-items: flex-start; gap: 0.625rem; padding: 0.625rem; background: var(--surface-2); border-radius: 0.5rem; border: 1px solid var(--border-soft); transition: border-color 0.2s ease; }
.pillar-item.highlighted { border-color: var(--brand-border); background: var(--surface-soft-2); }
.item-icon-wrap { width: 28px; height: 28px; border-radius: 6px; display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
.item-info { flex: 1; min-width: 0; }
.item-title { font-size: var(--fs-sm); font-weight: var(--fw-semibold); color: var(--text-strong); }
.item-step { font-size: var(--fs-2xs); font-weight: var(--fw-semibold); color: var(--brand); }
.item-desc { font-size: var(--fs-xs); color: var(--text-3); margin-top: 0.125rem; line-height: 1.4; }
.features-grid { display: flex; flex-wrap: wrap; gap: 0.5rem; }
.feature-pill { display: inline-flex; align-items: center; gap: 0.375rem; padding: 0.35rem 0.75rem; background: var(--surface-deep); border: 1px solid var(--border); border-radius: 9999px; font-size: var(--fs-xs); font-weight: var(--fw-medium); color: var(--text-2); }
</style>

