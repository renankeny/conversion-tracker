<template>
  <aside class="portal-sidebar">
    <div class="sidebar-header">
      <NuxtLink :to="basePath" class="sidebar-logo">
        <BrandLogo size="md" />
      </NuxtLink>

      <div class="project-badge">
        <div class="project-dot" />
        <div class="project-texts">
          <span class="project-kicker">Projeto conectado</span>
          <span class="project-name">{{ pixelLabel || 'Portal do Cliente' }}</span>
        </div>
      </div>
    </div>

    <nav class="sidebar-nav">
      <NuxtLink
        v-for="item in navLinks"
        :key="item.to"
        :to="item.to"
        class="sidebar-link"
        :class="{ active: item.exact ? route.path === item.to : route.path.startsWith(item.to) }"
      >
        <component :is="item.icon" :size="18" class="link-icon" />
        <span class="link-label">{{ item.label }}</span>
      </NuxtLink>
    </nav>

    <div class="sidebar-footer">
      <button type="button" class="theme-toggle" @click="toggleTheme">
        <Sun v-if="isDark" :size="16" />
        <Moon v-else :size="16" />
        <span>{{ isDark ? 'Tema claro' : 'Tema escuro' }}</span>
      </button>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { LayoutDashboard, Kanban, MessageSquare, Sun, Moon } from 'lucide-vue-next'
import type { Component } from 'vue'

const props = defineProps<{
  token: string
  pixelLabel: string
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
  if (props.showCrm) {
    list.push({ to: `${basePath.value}/crm`, label: 'CRM', icon: Kanban })
  }
  if (props.showConversations) {
    list.push({ to: `${basePath.value}/conversas`, label: 'Conversas', icon: MessageSquare })
  }
  return list
})
</script>

<style scoped>
.portal-sidebar {
  position: fixed; top: 0; left: 0; bottom: 0; width: 260px;
  background: var(--surface-deep); display: flex; flex-direction: column;
  z-index: 50;
}
.sidebar-header { padding: 1.5rem 1.25rem; border-bottom: 1px solid var(--border-soft); display: flex; flex-direction: column; gap: 1rem; }
.sidebar-logo { display: flex; align-items: center; text-decoration: none; }
.project-badge {
  display: flex; align-items: center; gap: 0.625rem; padding: 0.625rem 0.75rem;
  border-radius: 10px; background: var(--surface-2); border: 1px solid var(--border-soft);
}
.project-dot {
  width: 8px; height: 8px; border-radius: 50%; background: var(--brand);
  box-shadow: 0 0 8px var(--brand); flex-shrink: 0;
}
.project-texts { display: flex; flex-direction: column; min-width: 0; }
.project-kicker { font-size: var(--fs-2xs); color: var(--text-3); text-transform: uppercase; letter-spacing: 0.05em; }
.project-name { font-size: var(--fs-md); font-weight: var(--fw-semibold); color: var(--text-strong); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.sidebar-nav { flex: 1; padding: 1rem 0.75rem; display: flex; flex-direction: column; gap: 0.25rem; overflow-y: auto; }
.sidebar-link {
  display: flex; align-items: center; gap: 0.75rem; padding: 0.65rem 0.875rem; border-radius: 8px;
  color: var(--text-2); text-decoration: none; font-size: var(--fs-lg); font-weight: var(--fw-medium);
  transition: all 0.15s ease; position: relative;
}
.sidebar-link:hover { background: var(--surface-soft); color: var(--text-1); }
.sidebar-link.active { background: transparent; color: var(--brand); font-weight: var(--fw-semibold); }
.sidebar-link.active::before {
  content: ''; position: absolute; left: -0.75rem; top: 50%; transform: translateY(-50%);
  width: 3px; height: 60%; background: var(--brand); border-radius: 0 3px 3px 0;
}
.link-icon { display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
.sidebar-footer { padding: 1rem 0.75rem; border-top: 1px solid var(--border-soft); }
.theme-toggle {
  display: flex; align-items: center; gap: 0.625rem; padding: 0.55rem 0.875rem; border-radius: 8px;
  background: transparent; border: none; color: var(--text-2); font-size: var(--fs-md);
  font-weight: var(--fw-medium); cursor: pointer; transition: all 0.15s ease; width: 100%; text-align: left;
}
.theme-toggle:hover { background: var(--surface-soft); color: var(--text-1); }
@media (max-width: 1023px) { .portal-sidebar { transform: translateX(-100%); } }
</style>
