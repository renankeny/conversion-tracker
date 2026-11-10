<template>
  <nav class="mobile-bottom-nav lg:hidden">
    <div class="nav-items">
      <NuxtLink
        v-for="item in mainNavItems"
        :key="item.path"
        :to="item.path"
        :aria-label="item.label"
        class="nav-item"
        :class="{ active: isActive(item.path) }"
      >
        <component :is="item.icon" class="nav-icon" />
        <span class="nav-label">{{ item.label }}</span>
      </NuxtLink>

      <button
        class="nav-item"
        :class="{ active: drawerOpen || isMoreActive }"
        @click="drawerOpen = !drawerOpen"
        aria-label="Mais opções"
      >
        <Menu v-if="!drawerOpen" class="nav-icon" />
        <X v-else class="nav-icon" />
        <span class="nav-label">Mais</span>
        <span v-if="isMoreActive && !drawerOpen" class="more-dot" />
      </button>
    </div>
  </nav>

  <NavigationMobileDrawerMenu v-model="drawerOpen" />
</template>

<script setup lang="ts">
import { LayoutDashboard, Kanban, Zap, Users, Menu, X } from 'lucide-vue-next'

const { isActive, loadAccounts } = useAccountNavigation()
const drawerOpen = ref(false)

onMounted(loadAccounts)

const mainNavItems = [
  { path: '/', label: 'Dashboard', icon: LayoutDashboard },
  { path: '/crm', label: 'CRM', icon: Kanban },
  { path: '/events', label: 'Eventos', icon: Zap },
  { path: '/users', label: 'Usuários', icon: Users },
]

const isMoreActive = computed(() => !mainNavItems.some((item) => isActive(item.path)))
</script>

<style scoped>
.mobile-bottom-nav {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  z-index: 50;
  background: var(--surface-2);
  border-top: 1px solid var(--border-soft);
  box-shadow: 0 -4px 20px rgba(0, 0, 0, 0.25);
  padding-bottom: calc(0.35rem + env(safe-area-inset-bottom, 0));
  backdrop-filter: blur(16px);
  -webkit-backdrop-filter: blur(16px);
}

.nav-items {
  display: flex;
  align-items: center;
  justify-content: space-around;
  height: 62px;
  padding: 0 0.5rem;
  max-width: 600px;
  margin: 0 auto;
}

.nav-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 3px;
  padding: 0.375rem 0.5rem;
  border-radius: 10px;
  color: var(--text-3);
  text-decoration: none;
  border: none;
  background: transparent;
  cursor: pointer;
  transition: all 0.2s ease;
  flex: 1;
  position: relative;
  min-height: 48px;
}

.nav-item:hover {
  color: var(--text-1);
}

.nav-item.active {
  color: var(--brand);
}

.nav-icon {
  width: 22px;
  height: 22px;
  flex-shrink: 0;
}

.nav-label {
  font-size: var(--fs-xs);
  font-weight: var(--fw-medium);
  white-space: nowrap;
}

.more-dot {
  position: absolute;
  top: 6px;
  right: calc(50% - 14px);
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--brand);
}
</style>
