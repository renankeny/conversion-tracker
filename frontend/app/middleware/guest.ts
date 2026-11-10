import { useAuthStore } from '~/stores/useAuthStore'

export default defineNuxtRouteMiddleware(() => {
  const authStore = useAuthStore()
  const token = useCookie('auth_token')

  if (authStore.token || token.value) {
    return navigateTo('/')
  }
})
