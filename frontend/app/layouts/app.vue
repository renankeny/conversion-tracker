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

<style scoped>
.app-layout {
  min-height: 100vh;
  background: var(--surface-1);
  position: relative;
}

.app-background {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  height: 60vh;
  background: radial-gradient(
    ellipse at top,
    var(--brand-soft) 0%,
    transparent 60%
  );
  pointer-events: none;
  z-index: 0;
}

.app-main {
  position: relative;
  z-index: 1;
  margin-left: 260px;
  padding: 2rem;
  min-height: 100vh;
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
