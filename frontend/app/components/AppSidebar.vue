<template>
  <aside class="app-sidebar">
    <div class="sidebar-header">
      <NuxtLink to="/" class="sidebar-logo">
        <div class="logo-icon">
          <svg
            xmlns="http://www.w3.org/2000/svg"
            class="h-6 w-6"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <path stroke-linecap="round" stroke-linejoin="round" d="M3 3v18h18" />
            <path stroke-linecap="round" stroke-linejoin="round" d="M7 14l4-4 4 4 5-5" />
          </svg>
        </div>
        <span class="logo-text">TrackFind</span>
      </NuxtLink>
    </div>

    <nav class="sidebar-nav">
      <NuxtLink to="/" class="sidebar-link" :class="{ active: route.path === '/' }">
        <span class="link-icon" v-html="icons.dashboard" />
        <span class="link-label">Dashboard</span>
      </NuxtLink>

      <SidebarGroup
        label="Integrações"
        :icon="icons.integrations"
        base-path="/integracoes"
      >
        <NuxtLink
          to="/integracoes/pixels"
          class="sidebar-link sidebar-sublink"
          :class="{ active: isActive('/integracoes/pixels') }"
        >
          <span class="link-label">Pixels</span>
        </NuxtLink>
        <NuxtLink
          to="/integracoes/plataformas"
          class="sidebar-link sidebar-sublink"
          :class="{ active: isActive('/integracoes/plataformas') }"
        >
          <span class="link-label">Plataformas</span>
        </NuxtLink>
      </SidebarGroup>

      <NuxtLink to="/events" class="sidebar-link" :class="{ active: isActive('/events') }">
        <span class="link-icon" v-html="icons.events" />
        <span class="link-label">Eventos</span>
      </NuxtLink>

      <NuxtLink to="/users" class="sidebar-link" :class="{ active: isActive('/users') }">
        <span class="link-icon" v-html="icons.users" />
        <span class="link-label">Usuários</span>
      </NuxtLink>

      <NuxtLink to="/mappings" class="sidebar-link" :class="{ active: isActive('/mappings') }">
        <span class="link-icon" v-html="icons.mappings" />
        <span class="link-label">URL Mappings</span>
      </NuxtLink>

      <NuxtLink to="/settings" class="sidebar-link" :class="{ active: isActive('/settings') }">
        <span class="link-icon" v-html="icons.settings" />
        <span class="link-label">Configurações</span>
      </NuxtLink>
    </nav>

    <div class="sidebar-footer">
      <div class="user-info">
        <div class="user-avatar">{{ userInitials }}</div>
        <div class="user-details">
          <p class="user-name">{{ user?.name || user?.email || 'Admin' }}</p>
          <p class="user-email">{{ user?.email }}</p>
        </div>
      </div>

      <button class="theme-toggle" @click="toggle" :title="isDark ? 'Tema claro' : 'Tema escuro'">
        <svg v-if="isDark" xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
          <path d="M10 2a1 1 0 011 1v1a1 1 0 11-2 0V3a1 1 0 011-1zm4 8a4 4 0 11-8 0 4 4 0 018 0zm-.464 4.95l.707.707a1 1 0 001.414-1.414l-.707-.707a1 1 0 00-1.414 1.414zm2.12-10.607a1 1 0 010 1.414l-.706.707a1 1 0 11-1.414-1.414l.707-.707a1 1 0 011.414 0zM17 11a1 1 0 100-2h-1a1 1 0 100 2h1zm-7 4a1 1 0 011 1v1a1 1 0 11-2 0v-1a1 1 0 011-1zM5.05 6.464A1 1 0 106.465 5.05l-.708-.707a1 1 0 00-1.414 1.414l.707.707zm1.414 8.486l-.707.707a1 1 0 01-1.414-1.414l.707-.707a1 1 0 011.414 1.414zM4 11a1 1 0 100-2H3a1 1 0 000 2h1z" />
        </svg>
        <svg v-else xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
          <path d="M17.293 13.293A8 8 0 016.707 2.707a8.001 8.001 0 1010.586 10.586z" />
        </svg>
        <span>{{ isDark ? 'Tema claro' : 'Tema escuro' }}</span>
      </button>

      <button class="logout-btn" @click="handleLogout">
        <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
          <path fill-rule="evenodd" d="M3 3a1 1 0 00-1 1v12a1 1 0 102 0V4a1 1 0 00-1-1zm10.293 9.293a1 1 0 001.414 1.414l3-3a1 1 0 000-1.414l-3-3a1 1 0 10-1.414 1.414L14.586 9H7a1 1 0 100 2h7.586l-1.293 1.293z" clip-rule="evenodd" />
        </svg>
        <span>Sair</span>
      </button>
    </div>
  </aside>
</template>

<script setup lang="ts">
const route = useRoute()
const { user, logout } = useAuth()
const { isDark, toggle } = useTheme()

const icons = {
  dashboard:
    '<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor"><path d="M10.707 2.293a1 1 0 00-1.414 0l-7 7a1 1 0 001.414 1.414L4 10.414V17a1 1 0 001 1h2a1 1 0 001-1v-2a1 1 0 011-1h2a1 1 0 011 1v2a1 1 0 001 1h2a1 1 0 001-1v-6.586l.293.293a1 1 0 001.414-1.414l-7-7z"/></svg>',
  integrations:
    '<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor"><path d="M11 3a1 1 0 10-2 0v1a1 1 0 102 0V3zM15.657 5.757a1 1 0 00-1.414-1.414l-.707.707a1 1 0 001.414 1.414l.707-.707zM18 10a1 1 0 01-1 1h-1a1 1 0 110-2h1a1 1 0 011 1zM5.05 6.464A1 1 0 106.464 5.05l-.707-.707a1 1 0 00-1.414 1.414l.707.707zM5 10a1 1 0 01-1 1H3a1 1 0 110-2h1a1 1 0 011 1zM8 16v-1h4v1a2 2 0 11-4 0zM12 14c.015-.34.208-.646.477-.859a4 4 0 10-4.954 0c.27.213.462.519.477.859h4z"/></svg>',
  events:
    '<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M11.3 1.046A1 1 0 0112 2v5h4a1 1 0 01.82 1.573l-7 10A1 1 0 018 18v-5H4a1 1 0 01-.82-1.573l7-10a1 1 0 011.12-.38z" clip-rule="evenodd"/></svg>',
  users:
    '<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor"><path d="M9 6a3 3 0 11-6 0 3 3 0 016 0zM17 6a3 3 0 11-6 0 3 3 0 016 0zM12.93 17c.046-.327.07-.66.07-1a6.97 6.97 0 00-1.5-4.33A5 5 0 0119 16v1h-6.07zM6 11a5 5 0 015 5v1H1v-1a5 5 0 015-5z"/></svg>',
  mappings:
    '<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M12.586 4.586a2 2 0 112.828 2.828l-3 3a2 2 0 01-2.828 0 1 1 0 00-1.414 1.414 4 4 0 005.656 0l3-3a4 4 0 00-5.656-5.656l-1.5 1.5a1 1 0 101.414 1.414l1.5-1.5zm-5 5a2 2 0 012.828 0 1 1 0 101.414-1.414 4 4 0 00-5.656 0l-3 3a4 4 0 105.656 5.656l1.5-1.5a1 1 0 10-1.414-1.414l-1.5 1.5a2 2 0 11-2.828-2.828l3-3z" clip-rule="evenodd"/></svg>',
  settings:
    '<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M11.49 3.17c-.38-1.56-2.6-1.56-2.98 0a1.532 1.532 0 01-2.286.948c-1.372-.836-2.942.734-2.106 2.106.54.886.061 2.042-.947 2.287-1.561.379-1.561 2.6 0 2.978a1.532 1.532 0 01.947 2.287c-.836 1.372.734 2.942 2.106 2.106a1.532 1.532 0 012.287.947c.379 1.561 2.6 1.561 2.978 0a1.533 1.533 0 012.287-.947c1.372.836 2.942-.734 2.106-2.106a1.533 1.533 0 01.947-2.287c1.561-.379 1.561-2.6 0-2.978a1.532 1.532 0 01-.947-2.287c.836-1.372-.734-2.942-2.106-2.106a1.532 1.532 0 01-2.287-.947zM10 13a3 3 0 100-6 3 3 0 000 6z" clip-rule="evenodd"/></svg>',
}

const isActive = (path: string) => {
  if (path === '/') return route.path === '/'
  return route.path === path || route.path.startsWith(path + '/')
}

const userInitials = computed(() => {
  const name = user.value?.name || user.value?.email || 'A'
  return name
    .split(/[\s@]+/)
    .map((p: string) => p.charAt(0).toUpperCase())
    .slice(0, 2)
    .join('')
})

const handleLogout = async () => {
  await logout()
}
</script>

<style scoped>
.app-sidebar {
  position: fixed;
  top: 0;
  left: 0;
  bottom: 0;
  width: 260px;
  background: var(--surface-2);
  border-right: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  z-index: 50;
}

.sidebar-header {
  padding: 1.5rem 1.25rem;
  border-bottom: 1px solid var(--border-soft);
}

.sidebar-logo {
  display: flex;
  align-items: center;
  gap: 0.625rem;
  text-decoration: none;
  color: var(--text-strong);
}

.logo-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: 8px;
  background: var(--brand-soft);
  color: var(--brand);
}

.logo-text {
  font-family: 'Instrument Sans', sans-serif;
  font-size: 1.25rem;
  font-weight: 700;
}

.sidebar-nav {
  flex: 1;
  padding: 1rem 0.75rem;
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  overflow-y: auto;
}

.sidebar-link {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.625rem 0.875rem;
  border-radius: 8px;
  color: var(--text-2);
  text-decoration: none;
  font-size: 0.9375rem;
  font-weight: 500;
  transition: all 0.15s ease;
  position: relative;
}

.sidebar-link:hover {
  background: var(--surface-soft);
  color: var(--text-1);
}

.sidebar-link.active {
  background: var(--brand-soft);
  color: var(--brand);
}

.sidebar-link.active::before {
  content: '';
  position: absolute;
  left: -0.75rem;
  top: 50%;
  transform: translateY(-50%);
  width: 3px;
  height: 60%;
  background: var(--brand);
  border-radius: 0 3px 3px 0;
}

.sidebar-sublink {
  font-size: 0.875rem;
  padding: 0.5rem 0.75rem;
}

.sidebar-sublink.active::before {
  display: none;
}

.link-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.link-icon :deep(svg) {
  width: 1.125rem;
  height: 1.125rem;
}

.sidebar-footer {
  padding: 1rem 0.75rem;
  border-top: 1px solid var(--border-soft);
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.user-info {
  display: flex;
  align-items: center;
  gap: 0.625rem;
  padding: 0.5rem 0.75rem;
  margin-bottom: 0.5rem;
}

.user-avatar {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: 50%;
  background: var(--brand);
  color: white;
  font-weight: 600;
  font-size: 0.8125rem;
}

.user-details {
  flex: 1;
  min-width: 0;
}

.user-name {
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--text-1);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.user-email {
  font-size: 0.75rem;
  color: var(--text-3);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.theme-toggle,
.logout-btn {
  display: flex;
  align-items: center;
  gap: 0.625rem;
  padding: 0.5rem 0.875rem;
  border-radius: 8px;
  background: transparent;
  border: none;
  color: var(--text-2);
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s ease;
  text-align: left;
  width: 100%;
}

.theme-toggle:hover,
.logout-btn:hover {
  background: var(--surface-soft);
  color: var(--text-1);
}

.logout-btn:hover {
  background: var(--danger-soft);
  color: var(--danger);
}

@media (max-width: 1023px) {
  .app-sidebar {
    transform: translateX(-100%);
  }
}
</style>
