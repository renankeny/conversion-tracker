<template>
  <div class="client-portal-shell">
    <div v-if="state.notFound" class="portal-state-box">
      <div class="mb-4 flex justify-center"><BrandLogo size="md" show-tagline /></div>
      <h1>Link indisponível</h1>
      <p>Esse link de visualização é inválido ou foi desativado. Peça um novo link a quem compartilhou.</p>
    </div>

    <div v-else-if="state.loadError && !state.metaLoaded" class="portal-state-box">
      <div class="mb-4 flex justify-center"><BrandLogo size="md" show-tagline /></div>
      <h1>Falha temporária de conexão</h1>
      <p class="mb-4">{{ state.loadError }}</p>
      <div class="flex justify-center">
        <BaseButton variant="primary" size="sm" :loading="state.loadingMeta" @click="loadPortalMeta(true)">
          Tentar novamente
        </BaseButton>
      </div>
    </div>

    <template v-else>
      <ClientPortalSidebar
        :token="token"
        :pixel-label="state.pixelLabel"
        :show-crm="showCrm"
        :show-conversations="showConversations"
      />

      <main class="portal-main">
        <div class="mobile-brand-bar lg:hidden">
          <BrandLogo size="sm" />
          <span class="mobile-project-pill">{{ state.pixelLabel || 'Portal' }}</span>
        </div>

        <NuxtPage />
      </main>

      <ClientPortalMobileNav
        :token="token"
        :show-crm="showCrm"
        :show-conversations="showConversations"
      />
    </template>
  </div>
</template>

<script setup lang="ts">
import ClientPortalSidebar from '~/components/portal/ClientPortalSidebar.vue'
import ClientPortalMobileNav from '~/components/portal/ClientPortalMobileNav.vue'
import { useClientPortal } from '~/composables/useClientPortal'

definePageMeta({ layout: 'default' })

const route = useRoute()
const token = route.params.token as string
const { state, showCrm, showConversations, loadPortalMeta } = useClientPortal(token)

const enforceAllowedTab = () => {
  if (!state.value.metaLoaded) return
  const base = `/cliente/${encodeURIComponent(token)}`
  if (route.path.endsWith('/crm') && !showCrm.value) {
    void navigateTo(base, { replace: true })
  } else if (route.path.endsWith('/conversas') && !showConversations.value) {
    void navigateTo(base, { replace: true })
  }
}

watch(() => [state.value.metaLoaded, route.path, showCrm.value, showConversations.value], enforceAllowedTab)

onMounted(async () => {
  await loadPortalMeta(false)
  enforceAllowedTab()
})
</script>

<style scoped>
.client-portal-shell {
  min-height: 100vh;
  min-height: 100dvh;
  background: var(--surface-1);
  position: relative;
}
.portal-state-box {
  max-width: 480px;
  padding: 15vh 1.25rem 2rem;
  margin: 0 auto;
  text-align: center;
  color: var(--text-1);
}
.portal-state-box h1 {
  font-family: var(--font-heading);
  font-size: var(--fs-2xl);
  font-weight: var(--fw-bold);
  margin-bottom: 0.5rem;
}
.portal-state-box p {
  color: var(--text-2);
  font-size: var(--fs-md);
}
.portal-main {
  position: relative;
  z-index: 1;
  margin-left: 260px;
  padding: 2rem;
  min-height: 100vh;
  min-height: 100dvh;
}
.mobile-brand-bar {
  display: none;
}
.mobile-project-pill {
  font-size: var(--fs-xs);
  font-weight: var(--fw-semibold);
  color: var(--brand);
  background: var(--brand-soft);
  border: 1px solid var(--brand-border);
  padding: 0.2rem 0.6rem;
  border-radius: 999px;
  max-width: 55%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
@media (max-width: 1023px) {
  .portal-main {
    margin-left: 0;
    padding: 1rem;
    padding-bottom: calc(75px + 1rem + max(0.5rem, env(safe-area-inset-bottom, 0)));
  }
  .mobile-brand-bar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.75rem;
    margin-bottom: 1rem;
    padding-bottom: 0.75rem;
    border-bottom: 1px solid var(--border-soft);
  }
}
</style>
