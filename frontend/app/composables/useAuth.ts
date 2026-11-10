import { storeToRefs } from 'pinia'
import { useAuthStore } from '~/stores/useAuthStore'

export const useAuth = () => {
  const store = useAuthStore()
  const {
    token,
    user,
    isAuthenticated,
    isOwner,
    isAdmin,
    canManageAccounts,
    canManagePixels,
    hasOpenAiKey,
  } = storeToRefs(store)

  return {
    token,
    user,
    isAuthenticated,
    isOwner,
    isAdmin,
    canManageAccounts,
    canManagePixels,
    hasOpenAiKey,
    setUser: store.setUser,
    clearSession: store.clearSession,
    login: store.login,
    logout: store.logout,
    fetchMe: store.fetchMe,
    updatePreferences: store.updatePreferences,
  }
}
