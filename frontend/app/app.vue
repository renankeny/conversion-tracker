<template>
  <div>
    <NuxtRouteAnnouncer />

    <div v-if="serviceNotice?.active" class="notice-shell">
      <div class="notice-glow" aria-hidden="true" />
      <main class="notice-card">
        <div class="notice-brand">
          <BrandLogo size="lg" show-tagline />
        </div>
        <div v-if="serviceNotice.badge" class="notice-badge">
          <ShieldAlert :size="16" />
          <span>{{ serviceNotice.badge }}</span>
        </div>
        <h1 class="notice-title">{{ serviceNotice.title }}</h1>
        <p class="notice-body">{{ serviceNotice.body }}</p>
        <div v-if="serviceNotice.items?.length" class="notice-list">
          <div v-for="(item, idx) in serviceNotice.items" :key="idx" class="notice-item">
            <span class="notice-dot" />
            <span>{{ item }}</span>
          </div>
        </div>
      </main>
      <footer class="notice-footer">{{ footerText }}</footer>
    </div>

    <NuxtLayout v-else>
      <NuxtPage />
    </NuxtLayout>

    <!-- Toast Container -->
    <ClientOnly>
      <Teleport to="body">
        <div class="toast-container">
          <TransitionGroup name="toast">
            <BaseToast
              v-for="toast in toasts"
              :key="toast.id"
              :type="toast.type"
              :title="toast.title"
              :message="toast.message"
              @close="remove(toast.id)"
            />
          </TransitionGroup>
        </div>
      </Teleport>
    </ClientOnly>
  </div>
</template>

<script setup lang="ts">
import '../assets/css/main.css'
import { ShieldAlert } from 'lucide-vue-next'
import { computePaletteVars } from '~/utils/colorPalette'

const { theme } = useTheme()
const { branding, brandName, footerText, applyBrandingVisuals } = useBranding()
const { serviceNotice } = useSetup()

const activePaletteVars = computed(() => {
  const palette = theme.value === 'light'
    ? branding.value.theme_config?.light
    : branding.value.theme_config?.dark
  return palette ? computePaletteVars(palette) : {}
})

const ssrInlineStyle = computed(() =>
  Object.entries(activePaletteVars.value)
    .map(([k, v]) => `${k}: ${v}`)
    .join('; ')
)

const ssrCssVars = computed(() => {
  const decls = ssrInlineStyle.value
  if (!decls) return ''
  return `html:root, html:root[data-theme='dark'], html:root[data-theme='light'] { ${decls}; }`
})

useHead({
  titleTemplate: (titleChunk) => {
    if (!titleChunk) return brandName.value
    return titleChunk.includes(brandName.value) ? titleChunk : `${titleChunk} - ${brandName.value}`
  },
  htmlAttrs: {
    'data-theme': computed(() => theme.value),
    style: ssrInlineStyle,
  },
  link: computed(() =>
    branding.value.favicon_url
      ? [{ rel: 'icon', href: branding.value.favicon_url }]
      : [{ rel: 'icon', type: 'image/x-icon', href: '/favicon.ico' }]
  ),
  style: [{ id: 'whitelabel-ssr-vars', innerHTML: ssrCssVars }],
})

onMounted(() => {
  applyBrandingVisuals(branding.value)
})

const { toasts, remove } = useToast()
</script>

<style>
.notice-shell {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 2rem 1.25rem;
  background: var(--surface-deep);
  color: var(--text-1);
  position: relative;
  overflow: hidden;
}
.notice-glow {
  position: absolute;
  width: 560px;
  height: 560px;
  border-radius: 50%;
  background: radial-gradient(circle, var(--danger-soft) 0%, transparent 70%);
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  pointer-events: none;
}
.notice-card {
  position: relative;
  z-index: 1;
  width: 100%;
  max-width: 520px;
  background: var(--surface-1);
  border: 1px solid var(--border);
  border-radius: 20px;
  padding: 2.5rem 2rem;
  text-align: center;
  box-shadow: 0 24px 60px rgba(0, 0, 0, 0.32);
}
.notice-brand {
  display: flex;
  justify-content: center;
  margin-bottom: 1.5rem;
}
.notice-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.45rem;
  padding: 0.35rem 0.85rem;
  border-radius: 999px;
  background: var(--danger-soft);
  color: var(--danger);
  font-size: var(--fs-xs);
  font-weight: var(--fw-bold);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin-bottom: 1rem;
}
.notice-title {
  font-size: var(--fs-xl);
  font-weight: var(--fw-bold);
  color: var(--text-strong);
  margin: 0 0 0.75rem;
}
.notice-body {
  font-size: var(--fs-base);
  color: var(--text-2);
  line-height: 1.65;
  margin: 0 0 1.5rem;
}
.notice-list {
  display: flex;
  flex-direction: column;
  gap: 0.6rem;
  text-align: left;
  padding: 1rem 1.15rem;
  border-radius: 12px;
  background: var(--surface-2);
  border: 1px solid var(--border-soft);
}
.notice-item {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  font-size: var(--fs-xs);
  color: var(--text-2);
}
.notice-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--danger);
  flex-shrink: 0;
}
.notice-footer {
  position: relative;
  z-index: 1;
  margin-top: 1.75rem;
  font-size: var(--fs-xs);
  color: var(--text-3);
}
.toast-container {
  position: fixed;
  top: 1rem;
  right: 1rem;
  z-index: 9999;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  max-width: 420px;
  pointer-events: none;
}
.toast-container > * { pointer-events: all; }
.toast-enter-active, .toast-leave-active { transition: all 0.3s ease; }
.toast-enter-from, .toast-leave-to { transform: translateX(100%); opacity: 0; }
@media (max-width: 640px) {
  .toast-container { left: 1rem; right: 1rem; max-width: none; }
}
</style>
