export type Theme = 'dark' | 'light'

export const useTheme = () => {
  const theme = useCookie<Theme>('theme', {
    default: () => 'dark',
    maxAge: 60 * 60 * 24 * 365, // 1 ano
    sameSite: 'lax',
  })

  const isDark = computed(() => theme.value === 'dark')
  const isLight = computed(() => theme.value === 'light')

  const setTheme = (newTheme: Theme) => {
    theme.value = newTheme
    applyTheme(newTheme)
  }

  const toggle = () => {
    setTheme(isDark.value ? 'light' : 'dark')
  }

  const applyTheme = (themeValue: Theme) => {
    if (import.meta.client) {
      document.documentElement.setAttribute('data-theme', themeValue)
    }
  }

  // Aplicar tema no cliente quando o componente montar
  onMounted(() => {
    applyTheme(theme.value)
  })

  // Watch para mudanças no tema
  watch(theme, (newTheme) => {
    applyTheme(newTheme)
  })

  return {
    theme,
    isDark,
    isLight,
    setTheme,
    toggle,
  }
}
