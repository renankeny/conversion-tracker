<template>
  <div class="login-hero-container">
    <div class="ambient-glow glow-top"></div>
    <div class="ambient-glow glow-bottom"></div>

    <!-- Geometric Tech Grid Overlay -->
    <div class="grid-overlay">
      <svg class="w-full h-full" xmlns="http://www.w3.org/2000/svg">
        <pattern id="trackGrid" width="40" height="40" patternUnits="userSpaceOnUse">
          <path d="M 40 0 L 0 0 0 40" fill="none" stroke="var(--brand-soft)" stroke-width="0.8" stroke-dasharray="3 3" />
        </pattern>
        <rect width="100%" height="100%" fill="url(#trackGrid)" />
      </svg>
    </div>

    <!-- Animated Circuit Lines using dynamic --brand color -->
    <svg class="circuit-svg" xmlns="http://www.w3.org/2000/svg">
      <path class="circuit-line line-1" d="M -100 160 L 220 160 L 380 320 L 900 320" fill="none" stroke="var(--brand)" stroke-width="2" />
      <path class="circuit-line line-2" d="M -100 500 L 320 500 L 480 660 L 1000 660" fill="none" stroke="var(--brand)" stroke-width="1.8" />
      <path class="circuit-line line-3" d="M 200 -100 L 200 240 L 340 380 L 340 1000" fill="none" stroke="var(--brand)" stroke-width="1.5" />
    </svg>

    <!-- Glassmorphism Card with Whitelabel Brand & Features -->
    <div class="hero-showcase-card">
      <div class="mb-6">
        <BrandLogo size="lg" show-tagline />
      </div>

      <h1 class="hero-headline">
        Rastreamento avançado e atribuição em tempo real
      </h1>
      <p class="hero-subtext">
        Identifique a jornada completa do cliente, detecte tráfego de robôs com IA e sincronize conversões com Meta CAPI e Google Ads.
      </p>

      <!-- Simulated Realtime Stream Card -->
      <div class="stream-card">
        <div class="stream-header">
          <div class="flex items-center gap-2">
            <span class="live-indicator"></span>
            <span class="stream-title">Monitor de Tráfego em Tempo Real</span>
          </div>
          <span class="badge-status">
            <ShieldCheck :size="12" class="text-brand" /> Protegido
          </span>
        </div>

        <div class="stream-content">
          <div v-for="item in streamItems" :key="item.title" class="stream-item">
            <div class="item-icon-wrap bg-brand-soft">
              <component :is="item.icon" :size="14" class="text-brand" />
            </div>
            <div class="item-info">
              <div class="flex items-center justify-between">
                <span class="item-title">{{ item.title }}</span>
                <span class="item-time">{{ item.time }}</span>
              </div>
              <p class="item-desc">{{ item.desc }}</p>
            </div>
          </div>
        </div>
      </div>

      <!-- Feature Bullets -->
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
import { ShieldCheck, UserCheck, Zap, CheckCircle2, Check } from 'lucide-vue-next'

const streamItems = [
  {
    icon: UserCheck,
    title: 'Visitante Humano Identificado',
    time: 'agora',
    desc: 'User #YXXO3 • Chrome Mac • Score: +85 (Provável Humano)',
  },
  {
    icon: Zap,
    title: 'Atribuição de Campanha',
    time: '2s atrás',
    desc: 'utm_source: instagram_ads • Campanha: Escala_Principal',
  },
  {
    icon: CheckCircle2,
    title: 'Conversão Purchase Sincronizada',
    time: '5s atrás',
    desc: 'Meta CAPI + Google Ads com fbp & gclid enriquecidos',
  },
]

const features = ['Atribuição Multi-Canal', 'Anti-Bot Scoring', 'Webhook Instantâneo']
</script>

<style scoped>
.login-hero-container {
  height: 100vh;
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 2.5rem;
  position: relative;
  overflow: hidden;
  background: var(--surface-deep);
  color: var(--text-1);
  user-select: none;
}

.ambient-glow {
  position: absolute;
  width: 28rem;
  height: 28rem;
  border-radius: 9999px;
  pointer-events: none;
  filter: blur(120px);
  background: var(--brand-soft);
}
.glow-top { top: -6rem; left: -6rem; opacity: 0.85; }
.glow-bottom { bottom: -6rem; right: -6rem; opacity: 0.55; }

.grid-overlay, .circuit-svg {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  pointer-events: none;
}
.circuit-svg { opacity: 0.25; }
.circuit-line { stroke-dasharray: 10, 8; animation: circuitMove 35s linear infinite; }
.line-2 { animation-duration: 45s; animation-direction: reverse; }
.line-3 { animation-duration: 40s; }
@keyframes circuitMove { from { stroke-dashoffset: 2000; } to { stroke-dashoffset: 0; } }

.hero-showcase-card {
  position: relative;
  z-index: 10;
  max-width: 32rem;
  width: 100%;
  background: var(--surface-2);
  border: 1px solid var(--border);
  padding: 2rem;
  border-radius: 1.5rem;
  box-shadow: var(--shadow-elevated);
}

.text-brand { color: var(--brand); }
.bg-brand-soft { background: var(--brand-soft); }

.hero-headline {
  font-size: var(--fs-2xl);
  font-weight: var(--fw-bold);
  color: var(--text-strong);
  line-height: 1.3;
  margin-bottom: 0.625rem;
  letter-spacing: -0.02em;
}
.hero-subtext {
  font-size: var(--fs-base);
  color: var(--text-2);
  line-height: 1.5;
  margin-bottom: 1.5rem;
}

.stream-card {
  background: var(--surface-deep);
  border: 1px solid var(--border);
  border-radius: 1rem;
  padding: 0.875rem;
  margin-bottom: 1.25rem;
}
.stream-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-bottom: 0.625rem;
  border-bottom: 1px solid var(--border-soft);
  margin-bottom: 0.625rem;
}
.live-indicator {
  width: 7px;
  height: 7px;
  border-radius: 9999px;
  background: var(--brand);
  box-shadow: 0 0 8px var(--brand);
  animation: blink 1.8s infinite;
}
@keyframes blink { 0%, 100% { opacity: 1; } 50% { opacity: 0.4; } }

.stream-title { font-size: var(--fs-xs); font-weight: var(--fw-semibold); color: var(--text-2); text-transform: uppercase; letter-spacing: 0.04em; }
.badge-status { font-size: var(--fs-xs); font-weight: var(--fw-semibold); color: var(--brand); background: var(--brand-soft); padding: 0.15rem 0.5rem; border-radius: 0.375rem; display: flex; align-items: center; gap: 0.25rem; }
.stream-content { display: flex; flex-direction: column; gap: 0.5rem; }
.stream-item { display: flex; align-items: flex-start; gap: 0.625rem; padding: 0.5rem 0.625rem; background: var(--surface-2); border-radius: 0.5rem; border: 1px solid var(--border-soft); }
.item-icon-wrap { width: 26px; height: 26px; border-radius: 6px; display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
.item-info { flex: 1; min-width: 0; }
.item-title { font-size: var(--fs-sm); font-weight: var(--fw-semibold); color: var(--text-strong); }
.item-time { font-size: var(--fs-2xs); color: var(--text-3); }
.item-desc { font-size: var(--fs-xs); color: var(--text-3); margin-top: 0.125rem; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.features-grid { display: flex; flex-wrap: wrap; gap: 0.5rem; }
.feature-pill { display: inline-flex; align-items: center; gap: 0.375rem; padding: 0.35rem 0.75rem; background: var(--surface-deep); border: 1px solid var(--border); border-radius: 9999px; font-size: var(--fs-xs); font-weight: var(--fw-medium); color: var(--text-2); }
</style>
