import type { FetchOptions } from 'ofetch'

type ApiRequestOptions = FetchOptions<'json'>

export const useAPI = () => {
  const nuxtApp = useNuxtApp()
  const config = useRuntimeConfig()
  const apiBase = config.public.apiUrl as string

  const client = nuxtApp.$api
  const publicClient = nuxtApp.$publicApi

  const request = async <T>(endpoint: string, options: ApiRequestOptions = {}): Promise<T> => {
    return client<T>(endpoint, options)
  }

  const publicRequest = async <T>(endpoint: string, options: ApiRequestOptions = {}): Promise<T> => {
    return publicClient<T>(endpoint, options)
  }

  return {
    request,
    publicRequest,
    apiBase,
    get: <T>(endpoint: string, options?: ApiRequestOptions) =>
      request<T>(endpoint, { ...options, method: 'GET' }),
    publicGet: <T>(endpoint: string, options?: ApiRequestOptions) =>
      publicRequest<T>(endpoint, { ...options, method: 'GET' }),
    post: <T>(endpoint: string, body?: Record<string, unknown> | FormData | unknown, options?: ApiRequestOptions) =>
      request<T>(endpoint, { ...options, method: 'POST', body: body as ApiRequestOptions['body'] }),
    put: <T>(endpoint: string, body?: Record<string, unknown> | FormData | unknown, options?: ApiRequestOptions) =>
      request<T>(endpoint, { ...options, method: 'PUT', body: body as ApiRequestOptions['body'] }),
    delete: <T>(endpoint: string, options?: ApiRequestOptions) =>
      request<T>(endpoint, { ...options, method: 'DELETE' }),
  }
}
