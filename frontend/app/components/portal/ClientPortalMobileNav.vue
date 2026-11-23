<template>
  <nav class="portal-mobile-nav lg:hidden">
    <div class="nav-items">
      <NuxtLink
        v-for="item in navLinks"
        :key="item.to"
        :to="item.to"
        class="nav-item"
        :class="{ active: item.exact ? route.path === item.to : route.path.startsWith(item.to) }"
      >
        <component :is="item.icon" class="nav-icon" />
        <span class="nav-label">{{ item.label }}</span>
      </NuxtLink>

      <button type="button" class="nav-item" @click="toggleTheme">
        <Sun v-if="isDark" class="nav-icon" />
        <Moon v-else class="nav-icon" />
        <span class="nav-label">{{ isDark ? 'Claro' : 'Escuro' }}</span>
      </button>
    </div>
  </nav>
</template>

<script setup lang="ts">
import { LayoutDashboard, Kanban, MessageSquare, Sun, Moon } from 'lucide-vue-next'
import type { Component } from 'vue'

const props = defineProps<{
  token: string
  showCrm: boolean
  showConversations: boolean
}>()

const route = useRoute()
const { isDark, toggle: toggleTheme } = useTheme()
const basePath = computed(() => `/cliente/${encodeURIComponent(props.token)}`)

const navLinks = computed<Array<{ to: string; label: string; icon: Component; exact?: boolean }>>(() => {
  const list: Array<{ to: string; label: string; icon: Component; exact?: boolean }> = [
    { to: basePath.value, label: 'Dashboard', icon: LayoutDashboard, exact: true },
  ]
  if (props.showCrm) list.push({ to: `${basePath.value}/crm`, label: 'CRM', icon: Kanban })
  if (props.showConversations) list.push({ to: `${basePath.value}/conversas`, label: 'Conversas', icon: MessageSquare })
  return list
})
</script>

<style scoped>
.portal-mobile-nav {
  position: fixed; bottom: 0; left: 0; right: 0; z-index: 50;
  background: var(--surface-2); border-top: 1px solid var(--border-soft);
  padding-bottom: calc(0.35rem + env(safe-area-inset-bottom, 0));
  backdrop-filter: blur(16px); -webkit-backdrop-filter: blur(16px);
}
.nav-items {
  display: flex; align-items: center; justify-content: space-around;
  height: 62px; padding: 0 0.5rem; max-width: 600px; margin: 0 auto;
}
.nav-item {
  display: flex; flex-direction: column; align-items: center; justify-content: center;
  gap: 3px; padding: 0.375rem 0.5rem; border-radius: 10px; color: var(--text-3);
  text-decoration: none; border: none; background: transparent; cursor: pointer; flex: 1; min-height: 48px;
}
.nav-item.active { color: var(--brand); }
.nav-icon { width: 21px; height: 21px; flex-shrink: 0; }
.nav-label { font-size: var(--fs-xs); font-weight: var(--fw-medium); white-space: nowrap; }
@media (min-width: 1024px) { .portal-mobile-nav { display: none; } }
</style>
