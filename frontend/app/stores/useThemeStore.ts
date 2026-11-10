import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { setBrowserCookie } from '~/utils/cookies'

export type Theme = 'dark' | 'light'

const COOKIE_MAX_AGE = 60 * 60 * 24 * 365

export const useThemeStore = defineStore('theme', () => {
  const theme = ref<Theme>('dark')
  const hydrated = ref(false)

  const isDark = computed(() => theme.value === 'dark')
  const isLight = computed(() => theme.value === 'light')

  function applyTheme(value: Theme = theme.value): void {
    if (import.meta.client) {
      document.documentElement.setAttribute('data-theme', value)
    }
  }

  function hydrate(initialValue?: string | null): void {
    const resolved: Theme = initialValue === 'light' ? 'light' : 'dark'
    theme.value = resolved
    hydrated.value = true
    applyTheme(resolved)
  }

  function setTheme(next: Theme): void {
    const resolved: Theme = next === 'light' ? 'light' : 'dark'
    theme.value = resolved
    setBrowserCookie('tf_theme', resolved, COOKIE_MAX_AGE)
    setBrowserCookie('theme', resolved, COOKIE_MAX_AGE)
    applyTheme(resolved)
  }

  function toggle(): void {
    setTheme(isDark.value ? 'light' : 'dark')
  }

  return {
    theme,
    hydrated,
    isDark,
    isLight,
    applyTheme,
    hydrate,
    setTheme,
    toggle,
  }
})

