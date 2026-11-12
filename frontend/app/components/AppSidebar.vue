<template>
  <aside class="app-sidebar">
    <div class="sidebar-header">
      <NuxtLink to="/" class="sidebar-logo">
        <BrandLogo size="md" />
      </NuxtLink>

      <div v-if="accounts.length >= 1" class="mt-4">
        <BaseDropdown
          v-model="selectedAccountId"
          :options="accountOptions"
          @update:modelValue="(val) => val && switchAccount(val)"
        />
      </div>
    </div>

    <nav class="sidebar-nav">
      <NuxtLink
        v-for="item in primaryLinks"
        :key="item.to"
        :to="item.to"
        class="sidebar-link"
        :class="{ active: item.to === '/' ? route.path === '/' : isActive(item.to) }"
      >
        <component :is="item.icon" :size="18" class="link-icon" />
        <span class="link-label">{{ item.label }}</span>
      </NuxtLink>

      <SidebarGroup label="Integrações" :icon="Plug" base-path="/integracoes">
        <NuxtLink
          v-for="sub in integrationLinks"
          :key="sub.to"
          :to="sub.to"
          class="sidebar-link sidebar-sublink"
          :class="{ active: isActive(sub.to) }"
        >
          <component :is="sub.icon" v-if="sub.icon" :size="15" class="sublink-icon" />
          <span class="link-label">{{ sub.label }}</span>
        </NuxtLink>
      </SidebarGroup>

      <SidebarGroup
        label="Configurações"
        :icon="Settings"
        base-path="/settings"
      >
        <NuxtLink to="/settings" class="sidebar-link sidebar-sublink" :class="{ active: route.path === '/settings' }">
          <span class="link-label">Geral</span>
        </NuxtLink>
        <NuxtLink to="/settings/api" class="sidebar-link sidebar-sublink" :class="{ active: isActive('/settings/api') }">
          <span class="link-label">API</span>
        </NuxtLink>
      </SidebarGroup>
    </nav>

    <div class="sidebar-footer">
      <button class="theme-toggle" @click="toggleTheme" :title="isDark ? 'Tema claro' : 'Tema escuro'">
        <Sun v-if="isDark" :size="16" />
        <Moon v-else :size="16" />
        <span>{{ isDark ? 'Tema claro' : 'Tema escuro' }}</span>
      </button>

      <div class="user-info">
        <div class="user-avatar">{{ userInitials }}</div>
        <div class="user-details">
          <p class="user-name">{{ user?.name || user?.email || 'Admin' }}</p>
          <p class="user-email">{{ user?.email }}</p>
        </div>
      </div>

      <button class="logout-btn" @click="logout">
        <LogOut :size="16" />
        <span>Sair</span>
      </button>
    </div>
  </aside>
</template>

<script setup lang="ts">
import {
  LayoutDashboard, Plug, Zap, Users, Settings,
  Sun, Moon, LogOut, Webhook, Kanban, Building2,
} from 'lucide-vue-next'
import { MetaAdsIcon, GoogleAdsIcon, WhatsAppIcon, PixelBrainIcon } from '~/utils/adIcons'

const {
  route, user, logout, isDark, toggleTheme,
  accounts, accountOptions, selectedAccountId,
  canManageAccounts, loadAccounts, switchAccount, isActive, userInitials,
} = useAccountNavigation()

onMounted(() => loadAccounts())

const primaryLinks = computed(() => {
  const links = [
    { to: '/', label: 'Dashboard', icon: LayoutDashboard },
    { to: '/crm', label: 'CRM', icon: Kanban },
    { to: '/events', label: 'Eventos', icon: Zap },
    { to: '/users', label: 'Usuários', icon: Users },
    { to: '/webhooks', label: 'Webhooks', icon: Webhook },
  ]
  if (canManageAccounts.value) {
    links.push({ to: '/empresas', label: 'Empresas', icon: Building2 })
  }
  return links
})

const integrationLinks = [
  { to: '/integracoes/meta-ads', label: 'Meta Ads', icon: MetaAdsIcon },
  { to: '/integracoes/google-ads', label: 'Google Ads', icon: GoogleAdsIcon },
  { to: '/integracoes/pixels', label: 'Pixel', icon: PixelBrainIcon },
  { to: '/integracoes/whatsapp', label: 'WhatsApp', icon: WhatsAppIcon },
  { to: '/integracoes/plataformas', label: 'Checkouts (infoprodutos)' },
  { to: '/integracoes/custom', label: 'Checkouts Personalizados' },
  { to: '/integracoes/ecommerce/confirmacao-vendas', label: 'Confirmação de Vendas' },
]
</script>

<style scoped>
.app-sidebar {
  position: fixed;
  top: 0;
  left: 0;
  bottom: 0;
  width: 260px;
  background: var(--surface-deep);
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
  text-decoration: none;
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
  font-size: var(--fs-lg);
  font-weight: var(--fw-medium);
  transition: all 0.15s ease;
  position: relative;
}
.sidebar-link:hover { background: var(--surface-soft); color: var(--text-1); }
.sidebar-link.active { background: transparent; color: var(--brand); }
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
.sidebar-sublink { font-size: var(--fs-md); padding: 0.5rem 0.75rem; gap: 0.625rem; }
.sidebar-sublink.active::before { display: none; }
.sublink-icon { width: 16px; height: 16px; display: inline-flex; align-items: center; justify-content: center; flex-shrink: 0; color: inherit; }
.link-icon { display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
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
  margin-bottom: 0.25rem;
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
  color: var(--brand-text);
  font-weight: var(--fw-bold);
  font-size: var(--fs-base);
}
.user-details { flex: 1; min-width: 0; }
.user-name {
  font-size: var(--fs-md);
  font-weight: var(--fw-semibold);
  color: var(--text-1);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.user-email {
  font-size: var(--fs-sm);
  color: var(--text-3);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.theme-toggle, .logout-btn {
  display: flex;
  align-items: center;
  gap: 0.625rem;
  padding: 0.5rem 0.875rem;
  border-radius: 8px;
  background: transparent;
  border: none;
  color: var(--text-2);
  font-size: var(--fs-md);
  font-weight: var(--fw-medium);
  cursor: pointer;
  transition: all 0.15s ease;
  text-align: left;
  width: 100%;
}
.theme-toggle:hover { background: var(--surface-soft); color: var(--text-1); }
.logout-btn:hover { background: var(--danger-soft); color: var(--danger); }
@media (max-width: 1023px) { .app-sidebar { transform: translateX(-100%); } }
</style>
