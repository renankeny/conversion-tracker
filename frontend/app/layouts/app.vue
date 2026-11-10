<template>
  <div class="app-layout">
    <div class="app-background" aria-hidden="true" />

    <!-- Desktop Sidebar -->
    <AppSidebar />

    <main class="app-main">
      <slot />
    </main>

    <!-- Mobile Bottom Navigation -->
    <MobileBottomNav />
  </div>
</template>

<script setup lang="ts">
import { useLiveEventsStore } from '~/stores/useLiveEventsStore'

// Conecta o WebSocket de eventos ao vivo quando o usuário entra no painel.
// Uma única conexão por sessão autenticada — todas as páginas compartilham
// via useLiveEventsStore (singleton Pinia).
const { token } = useAuth()
const liveStore = useLiveEventsStore()

onMounted(() => {
  if (token.value) {
    void liveStore.connect(token.value)
  }
})

watch(token, (newToken) => {
  if (newToken) {
    void liveStore.connect(newToken)
  } else {
    liveStore.disconnect()
  }
})

onUnmounted(() => {
  liveStore.disconnect()
})
</script>

<style scoped>
.app-layout {
  min-height: 100vh;
  min-height: 100dvh;
  background: var(--surface-1);
  position: relative;
}

.app-background {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  height: 100vh;
  height: 100dvh;
  /* Sem mancha no fundo */
  background: transparent;
  pointer-events: none;
  z-index: 0;
}

.app-main {
  position: relative;
  z-index: 1;
  margin-left: 260px;
  padding: 2rem;
  min-height: 100vh;
  min-height: 100dvh;
  /* Espaço para o bottom nav no mobile */
  padding-bottom: 2rem;
}

/* Mobile - bottom nav handling */
@media (max-width: 1023px) {
  .app-main {
    margin-left: 0;
    padding: 1rem;
    /* Bottom nav + safe area do iOS */
    padding-bottom: calc(75px + 1rem + max(0.5rem, env(safe-area-inset-bottom, 0)));
  }
}

/* Garante que o bottom nav fique sempre no topo do conteúdo */
:deep(> nav) {
  z-index: 40;
}
</style>
