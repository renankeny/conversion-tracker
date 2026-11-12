<template>
  <Teleport to="body">
    <Transition name="overlay">
      <div v-if="modelValue" class="drawer-overlay lg:hidden" @click="close" />
    </Transition>

    <Transition name="drawer">
      <div v-if="modelValue" class="mobile-drawer lg:hidden">
        <div class="drawer-header">
          <NuxtLink to="/" class="drawer-logo" @click="close">
            <BrandLogo size="sm" />
          </NuxtLink>
          <button class="drawer-close" @click="close" aria-label="Fechar menu">
            <X :size="20" />
          </button>
        </div>

        <div v-if="user" class="drawer-user">
          <div class="drawer-avatar">{{ userInitials }}</div>
          <div class="flex-1 min-w-0">
            <p class="drawer-user-name">{{ user.name || user.email || 'Admin' }}</p>
            <p class="drawer-user-email">{{ user.email }}</p>
          </div>
        </div>

        <div v-if="accounts.length > 1" class="drawer-account">
          <div class="drawer-account-label">
            <Building2 :size="14" />
            <span>Empresa</span>
          </div>
          <div class="flex flex-col gap-1">
            <button
              v-for="acc in accounts"
              :key="acc.id"
              class="drawer-account-btn"
              :class="{ active: selectedAccountId === acc.id.toString() }"
              @click="onSelectAccount(acc.id.toString())"
            >
              <span class="acc-dot" :class="{ active: selectedAccountId === acc.id.toString() }" />
              <span class="truncate">{{ acc.name }}</span>
            </button>
          </div>
        </div>

        <nav class="drawer-nav">
          <NuxtLink
            v-for="link in primaryLinks"
            :key="link.to"
            :to="link.to"
            class="drawer-link"
            :class="{ active: link.to === '/' ? route.path === '/' : isActive(link.to) }"
            @click="close"
          >
            <component :is="link.icon" :size="18" />
            <span class="flex-1">{{ link.label }}</span>
          </NuxtLink>

          <div class="flex flex-col">
            <button
              type="button"
              class="drawer-group-toggle"
              @click="integrationsOpen = !integrationsOpen"
            >
              <Plug :size="18" />
              <span class="flex-1">Integrações</span>
              <ChevronDown class="group-chevron" :class="{ rotated: integrationsOpen }" :size="16" />
            </button>
            <div v-show="integrationsOpen" class="drawer-group-children">
              <NuxtLink
                v-for="sub in integrationLinks"
                :key="sub.to"
                :to="sub.to"
                class="drawer-sublink"
                :class="{ active: isActive(sub.to) }"
                @click="close"
              >
                <component :is="sub.icon" v-if="sub.icon" :size="15" class="sublink-icon" />
                <span v-else class="sublink-dot" />
                <span>{{ sub.label }}</span>
              </NuxtLink>
            </div>
          </div>

          <div class="flex flex-col">
            <button
              type="button"
              class="drawer-group-toggle"
              @click="settingsOpen = !settingsOpen"
            >
              <Settings :size="18" />
              <span class="flex-1">Configurações</span>
              <ChevronDown class="group-chevron" :class="{ rotated: settingsOpen }" :size="16" />
            </button>
            <div v-show="settingsOpen" class="drawer-group-children">
              <NuxtLink to="/settings" class="drawer-sublink" :class="{ active: route.path === '/settings' }" @click="close">
                <span class="sublink-dot" /><span>Geral</span>
              </NuxtLink>
              <NuxtLink to="/settings/api" class="drawer-sublink" :class="{ active: isActive('/settings/api') }" @click="close">
                <span class="sublink-dot" /><span>API</span>
              </NuxtLink>
            </div>
          </div>
        </nav>

        <div class="drawer-footer">
          <button class="drawer-action" @click="toggleTheme">
            <Sun v-if="isDark" :size="16" />
            <Moon v-else :size="16" />
            <span>{{ isDark ? 'Tema claro' : 'Tema escuro' }}</span>
          </button>
          <button class="drawer-action danger" @click="onLogout">
            <LogOut :size="16" />
            <span>Sair</span>
          </button>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import {
  LayoutDashboard, Kanban, Zap, Users, Webhook,
  Plug, Settings, X, Sun, Moon, LogOut,
  Building2, ChevronDown,
} from 'lucide-vue-next'
import { MetaAdsIcon, GoogleAdsIcon, WhatsAppIcon, PixelBrainIcon } from '~/utils/adIcons'

defineProps<{ modelValue: boolean }>()
const emit = defineEmits<{ (e: 'update:modelValue', v: boolean): void }>()

const {
  route, user, logout, isDark, toggleTheme,
  accounts, selectedAccountId, canManageAccounts, switchAccount, isActive, userInitials,
} = useAccountNavigation()

const integrationsOpen = ref(route.path.startsWith('/integracoes'))
const settingsOpen = ref(route.path.startsWith('/settings'))

const close = () => emit('update:modelValue', false)
const onSelectAccount = (id: string) => {
  close()
  switchAccount(id)
}
const onLogout = async () => {
  close()
  await logout()
}

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
.drawer-overlay { position: fixed; inset: 0; background: rgba(0, 0, 0, 0.65); z-index: 55; backdrop-filter: blur(4px); }
.mobile-drawer {
  position: fixed; bottom: 0; left: 0; right: 0; z-index: 60;
  background: var(--surface-1); border-radius: 20px 20px 0 0; max-height: 88vh;
  display: flex; flex-direction: column;
  padding-bottom: calc(0.75rem + env(safe-area-inset-bottom, 0));
  box-shadow: 0 -10px 40px rgba(0, 0, 0, 0.35); border-top: 1px solid var(--border-soft);
}
.drawer-header { display: flex; align-items: center; justify-content: space-between; padding: 1.125rem 1.25rem 0.875rem; border-bottom: 1px solid var(--border-soft); }
.drawer-logo { text-decoration: none; }
.drawer-close { width: 34px; height: 34px; border-radius: 8px; background: var(--surface-soft); border: 1px solid var(--border-soft); color: var(--text-2); display: flex; align-items: center; justify-content: center; cursor: pointer; }
.drawer-user { display: flex; align-items: center; gap: 0.75rem; padding: 0.875rem 1.25rem; border-bottom: 1px solid var(--border-soft); }
.drawer-avatar { width: 38px; height: 38px; border-radius: 50%; background: var(--brand); color: var(--brand-text); display: flex; align-items: center; justify-content: center; font-weight: var(--fw-bold); font-size: var(--fs-md); flex-shrink: 0; }
.drawer-user-name { font-weight: var(--fw-semibold); font-size: var(--fs-md); color: var(--text-1); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.drawer-user-email { font-size: var(--fs-sm); color: var(--text-3); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.drawer-account { padding: 0.75rem 1.25rem; border-bottom: 1px solid var(--border-soft); }
.drawer-account-label { display: flex; align-items: center; gap: 0.375rem; font-size: var(--fs-xs); font-weight: var(--fw-bold); text-transform: uppercase; letter-spacing: 0.06em; color: var(--text-3); margin-bottom: 0.5rem; }
.drawer-account-btn { display: flex; align-items: center; gap: 0.625rem; padding: 0.5rem 0.75rem; border-radius: 8px; background: transparent; border: 1px solid transparent; color: var(--text-2); font-size: var(--fs-md); font-weight: var(--fw-medium); cursor: pointer; width: 100%; text-align: left; }
.drawer-account-btn.active { background: var(--brand-soft); color: var(--brand); border-color: var(--brand-border); font-weight: var(--fw-semibold); }
.acc-dot { width: 8px; height: 8px; border-radius: 50%; background: var(--border); flex-shrink: 0; }
.acc-dot.active { background: var(--brand); }
.drawer-nav { flex: 1; overflow-y: auto; padding: 0.75rem 1rem; display: flex; flex-direction: column; gap: 3px; }
.drawer-link, .drawer-group-toggle { display: flex; align-items: center; gap: 0.75rem; padding: 0.6875rem 0.875rem; border-radius: 10px; color: var(--text-2); text-decoration: none; font-size: var(--fs-lg); font-weight: var(--fw-medium); background: transparent; border: none; width: 100%; text-align: left; cursor: pointer; }
.drawer-link.active { background: var(--brand-soft); color: var(--brand); font-weight: var(--fw-semibold); }
.group-chevron { transition: transform 0.2s ease; opacity: 0.6; }
.group-chevron.rotated { transform: rotate(180deg); }
.drawer-group-children { display: flex; flex-direction: column; gap: 2px; padding: 0.25rem 0 0.25rem 1.75rem; }
.drawer-sublink { display: flex; align-items: center; gap: 0.625rem; padding: 0.5625rem 0.75rem; border-radius: 8px; color: var(--text-2); text-decoration: none; font-size: var(--fs-md); font-weight: var(--fw-medium); }
.drawer-sublink.active { background: var(--brand-soft); color: var(--brand); font-weight: var(--fw-semibold); }
.sublink-icon { width: 16px; height: 16px; display: inline-flex; align-items: center; justify-content: center; flex-shrink: 0; color: inherit; }
.sublink-dot { width: 5px; height: 5px; border-radius: 50%; background: currentColor; opacity: 0.6; }
.drawer-footer { display: flex; gap: 0.5rem; padding: 0.75rem 1rem 0.25rem; border-top: 1px solid var(--border-soft); }
.drawer-action { flex: 1; display: flex; align-items: center; justify-content: center; gap: 0.5rem; padding: 0.625rem; border-radius: 10px; background: var(--surface-2); border: 1px solid var(--border-soft); color: var(--text-2); font-size: var(--fs-base); font-weight: var(--fw-semibold); cursor: pointer; }
.drawer-action.danger { color: var(--danger); background: var(--danger-soft); }
</style>

