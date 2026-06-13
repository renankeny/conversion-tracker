import type { AdminUser, LoginRequest, LoginResponse } from '#shared/types'

export const useAuth = () => {
  const apiUrl = useRuntimeConfig().public.apiUrl as string

  // Token persiste em cookie por 24h
  const token = useCookie<string | null>('auth_token', {
    maxAge: 60 * 60 * 24, // 24 horas
    sameSite: 'lax',
    default: () => null,
  })

  // User também persiste em cookie pra evitar flash de "deslogado" no reload
  const user = useCookie<AdminUser | null>('auth_user', {
    maxAge: 60 * 60 * 24, // 24 horas
    sameSite: 'lax',
    default: () => null,
  })

  const isAuthenticated = computed(() => !!token.value)

  const login = async (credentials: LoginRequest) => {
    try {
      const response = await $fetch<LoginResponse>('/admin/login', {
        baseURL: apiUrl,
        method: 'POST',
        body: credentials,
      })

      token.value = response.access_token
      user.value = response.user

      return response
    } catch (error: any) {
      throw new Error(error.response?._data?.error || 'Erro ao fazer login')
    }
  }

  const logout = async () => {
    try {
      if (token.value) {
        await $fetch('/admin/logout', {
          baseURL: apiUrl,
          method: 'POST',
          headers: {
            Authorization: `Bearer ${token.value}`,
          },
        }).catch(() => {
          // Backend pode não ter endpoint de logout; ignora.
        })
      }
    } finally {
      token.value = null
      user.value = null
      await navigateTo('/login')
    }
  }

  // Valida o token contra o backend e atualiza o user em cookie.
  // Retorna true se o token é válido, false se expirou ou foi revogado.
  const fetchMe = async (): Promise<boolean> => {
    if (!token.value) return false

    try {
      const me = await $fetch<AdminUser>('/admin/me', {
        baseURL: apiUrl,
        headers: {
          Authorization: `Bearer ${token.value}`,
        },
      })
      user.value = me
      return true
    } catch (error) {
      // Token inválido/expirado: limpa o estado.
      token.value = null
      user.value = null
      return false
    }
  }

  return {
    token,
    user,
    isAuthenticated,
    login,
    logout,
    fetchMe,
  }
}
