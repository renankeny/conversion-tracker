import type { FetchOptions } from 'ofetch'

export const useAPI = () => {
  const config = useRuntimeConfig()
  const { token } = useAuth()

  const apiBase = config.public.apiUrl as string

  const request = async <T>(
    endpoint: string,
    options: FetchOptions = {}
  ): Promise<T> => {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...(options.headers as Record<string, string>),
    }

    if (token.value) {
      headers['Authorization'] = `Bearer ${token.value}`
    }

    try {
      const response = await $fetch<T>(`${apiBase}${endpoint}`, {
        ...options,
        headers,
      })

      return response
    } catch (error: any) {
      // Tratar erros da API
      if (error.response?._data) {
        throw new Error(error.response._data.error || 'Erro desconhecido')
      }
      throw error
    }
  }

  return {
    request,
    apiBase,
    get: <T>(endpoint: string, options?: FetchOptions) =>
      request<T>(endpoint, { ...options, method: 'GET' }),
    post: <T>(endpoint: string, body?: any, options?: FetchOptions) =>
      request<T>(endpoint, { ...options, method: 'POST', body }),
    put: <T>(endpoint: string, body?: any, options?: FetchOptions) =>
      request<T>(endpoint, { ...options, method: 'PUT', body }),
    delete: <T>(endpoint: string, options?: FetchOptions) =>
      request<T>(endpoint, { ...options, method: 'DELETE' }),
  }
}
