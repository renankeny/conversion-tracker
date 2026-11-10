import type { FetchOptions } from 'ofetch'

export const useAPI = () => {
  const nuxtApp = useNuxtApp()
  const config = useRuntimeConfig()
  const apiBase = config.public.apiUrl as string

  const client = nuxtApp.$api
  const publicClient = nuxtApp.$publicApi

  const request = async <T>(endpoint: string, options: FetchOptions = {}): Promise<T> => {
    return client<T>(endpoint, options as any)
  }

  const publicRequest = async <T>(endpoint: string, options: FetchOptions = {}): Promise<T> => {
    return publicClient<T>(endpoint, options as any)
  }

  return {
    request,
    publicRequest,
    apiBase,
    get: <T>(endpoint: string, options?: FetchOptions) =>
      request<T>(endpoint, { ...options, method: 'GET' }),
    publicGet: <T>(endpoint: string, options?: FetchOptions) =>
      publicRequest<T>(endpoint, { ...options, method: 'GET' }),
    post: <T>(endpoint: string, body?: any, options?: FetchOptions) =>
      request<T>(endpoint, { ...options, method: 'POST', body }),
    put: <T>(endpoint: string, body?: any, options?: FetchOptions) =>
      request<T>(endpoint, { ...options, method: 'PUT', body }),
    delete: <T>(endpoint: string, options?: FetchOptions) =>
      request<T>(endpoint, { ...options, method: 'DELETE' }),
  }
}
