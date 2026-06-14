<template>
  <nav
    class="fixed bottom-0 left-0 right-0 z-50 lg:hidden"
    :class="[
      'bg-surface-2 border-t border-border backdrop-blur-sm',
      'safe-area-bottom',
    ]"
    :style="{
      paddingBottom: 'calc(0.5rem + env(safe-area-inset-bottom))',
    }"
  >
    <div class="flex items-center justify-around h-[75px] px-2">
      <NuxtLink
        v-for="item in navItems"
        :key="item.path"
        :to="item.path"
        :aria-label="item.label"
        class="flex flex-col items-center justify-center w-12 h-12 rounded-lg transition-all duration-200 group"
        :class="[
          isActive(item.path)
            ? 'text-brand'
            : 'text-text-2 hover:text-text-1 hover:bg-surface-3',
        ]"
      >
        <component
          :is="item.icon"
          class="w-6 h-6 transition-transform duration-200"
          :class="isActive(item.path) ? 'scale-110' : 'group-hover:scale-105'"
        />
        <span class="text-xs mt-1 font-medium truncate">{{ item.label }}</span>
      </NuxtLink>
    </div>
  </nav>
</template>

<script setup lang="ts">
import {
  HomeIcon,
  CursorArrowRaysIcon,
  PaperAirplaneIcon,
  UsersIcon,
  Cog6ToothIcon,
} from '@heroicons/vue/24/outline'

const route = useRoute()

interface NavItem {
  path: string
  label: string
  icon: any
}

const navItems: NavItem[] = [
  {
    path: '/',
    label: 'Dashboard',
    icon: HomeIcon,
  },
  {
    path: '/pixels',
    label: 'Pixels',
    icon: CursorArrowRaysIcon,
  },
  {
    path: '/events',
    label: 'Eventos',
    icon: PaperAirplaneIcon,
  },
  {
    path: '/users',
    label: 'Usuários',
    icon: UsersIcon,
  },
  {
    path: '/settings',
    label: 'Config',
    icon: Cog6ToothIcon,
  },
]

const isActive = (path: string) => {
  if (path === '/' && route.path === '/') return true
  return route.path.startsWith(path) && path !== '/'
}
</script>

<style scoped>
/* Suporta safe area do notch em iOS */
.safe-area-bottom {
  padding-bottom: calc(0.5rem + env(safe-area-inset-bottom, 0));
}

/* Garante fundo sólido sem transparência */
nav {
  background-color: var(--surface-2) !important;
  opacity: 1 !important;
  box-shadow: 0 -2px 8px rgba(0, 0, 0, 0.15);
}
</style>
