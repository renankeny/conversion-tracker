import { storeToRefs } from 'pinia'
import { useThemeStore } from '~/stores/useThemeStore'

export const useTheme = () => {
  const store = useThemeStore()
  const { theme, isDark, isLight } = storeToRefs(store)

  return {
    theme,
    isDark,
    isLight,
    setTheme: store.setTheme,
    toggle: store.toggle,
  }
}
