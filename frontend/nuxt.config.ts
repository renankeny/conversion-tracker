// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  compatibilityDate: '2025-07-15',
  devtools: { enabled: false },

  future: {
    compatibilityVersion: 4,
  },

  // Recarrega a página automaticamente quando um chunk JS/CSS antigo falha
  // ao carregar (acontece após rebuilds em dev e deploys em prod).
  // 'automatic' é o valor correto na API atual do Nuxt 4.
  experimental: {
    emitRouteChunkError: 'automatic',
  },

  build: {
    transpile: ['echarts', 'zrender', 'vue-echarts', 'tslib'],
  },

  modules: ['@nuxtjs/tailwindcss', '@pinia/nuxt'],

  app: {
    head: {
      meta: [
        { charset: 'utf-8' },
        { name: 'viewport', content: 'width=device-width, initial-scale=1' },
        { name: 'description', content: 'Plataforma de rastreamento avançado e atribuição de conversões' },
      ],
      link: [
        { rel: 'icon', type: 'image/x-icon', href: '/favicon.ico' },
      ],
    },
  },

  // Nuxt lê variáveis de ambiente automaticamente via prefixo NUXT_PUBLIC_*.
  // Os defaults aqui servem apenas para desenvolvimento local.
  runtimeConfig: {
    public: {
      apiUrl: 'http://localhost:8080/api/v1',
      appUrl: 'http://localhost:3000',
      appName: 'Sua Plataforma',
    },
  },

  tailwindcss: {
    configPath: 'tailwind.config.ts',
  },
})
