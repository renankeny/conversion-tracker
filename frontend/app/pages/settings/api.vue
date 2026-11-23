<script setup lang="ts">
import { KeyRound, Copy, Check } from 'lucide-vue-next'
import ApiDocsSidebar from '~/components/api/ApiDocsSidebar.vue'
import ApiMcpSetupSection from '~/components/api/ApiMcpSetupSection.vue'
import ApiAuthSection from '~/components/api/ApiAuthSection.vue'
import ApiEndpointCard from '~/components/api/ApiEndpointCard.vue'
import SettingsApiKeysTab from '~/components/settings/SettingsApiKeysTab.vue'
import { API_MODULES } from '~/utils/apiCatalog'

definePageMeta({
  layout: 'app',
  middleware: 'auth',
})

const { formatTitle } = useBranding()
useHead({ title: computed(() => formatTitle('API e Agentes IA (MCP)')) })

const config = useRuntimeConfig()
const apiUrl = computed(() => {
  const configured = String(config.public.apiUrl || '').trim()
  if (configured) return configured.replace(/\/+$/, '')
  if (import.meta.client) return `${window.location.origin}/api/v1`
  return 'http://localhost:8080/api/v1'
})

const activeSection = ref('mcp')
const searchQuery = ref('')
const keysModalOpen = ref(false)
const copiedBase = ref(false)

const activeModule = computed(() => API_MODULES.find((m) => m.id === activeSection.value) || null)

const filteredResults = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  if (!q) return []
  return API_MODULES.map((mod) => ({
    ...mod,
    endpoints: mod.endpoints.filter(
      (ep) =>
        ep.title.toLowerCase().includes(q) ||
        ep.path.toLowerCase().includes(q) ||
        ep.summary.toLowerCase().includes(q) ||
        ep.method.toLowerCase().includes(q)
    ),
  })).filter((mod) => mod.endpoints.length > 0)
})

const copyBaseUrl = async () => {
  await navigator.clipboard.writeText(apiUrl.value)
  copiedBase.value = true
  setTimeout(() => {
    copiedBase.value = false
  }, 1800)
}
</script>

<template>
  <div class="api-docs-page">
    <header class="page-top">
      <div>
        <h1 class="page-title">API e Servidor MCP (IA)</h1>
        <p class="page-subtitle">
          Conecte agentes de IA, n8n, Make ou sistemas próprios com cabeçalho único <code>X-API-Key</code>.
        </p>
      </div>

      <div class="top-actions">
        <button type="button" class="base-pill" title="Copiar URL Base da API" @click="copyBaseUrl">
          <span class="base-pill-label">Base:</span>
          <code>{{ apiUrl }}</code>
          <Check v-if="copiedBase" class="icon-sm text-brand" />
          <Copy v-else class="icon-sm" />
        </button>

        <BaseButton variant="primary" size="sm" @click="keysModalOpen = true">
          <KeyRound class="icon-sm" />
          Gerenciar API Keys
        </BaseButton>
      </div>
    </header>

    <div class="docs-layout">
      <ApiDocsSidebar
        v-model:active-section="activeSection"
        v-model:search-query="searchQuery"
        :modules="API_MODULES"
      />

      <main class="docs-content">
        <template v-if="searchQuery.trim()">
          <div v-if="!filteredResults.length" class="empty-search">
            Nenhuma rota encontrada para <strong>"{{ searchQuery }}"</strong>.
          </div>

          <section v-for="group in filteredResults" :key="group.id" class="module-section">
            <div class="module-head">
              <h2 class="module-title">{{ group.badge }} · {{ group.title }}</h2>
              <p class="module-sub">{{ group.subtitle }}</p>
            </div>
            <div class="endpoints-stack">
              <ApiEndpointCard
                v-for="ep in group.endpoints"
                :key="ep.id"
                :endpoint="ep"
                :api-url="apiUrl"
                default-open
              />
            </div>
          </section>
        </template>

        <ApiMcpSetupSection
          v-else-if="activeSection === 'mcp'"
          :api-url="apiUrl"
          @open-keys="keysModalOpen = true"
        />

        <ApiAuthSection
          v-else-if="activeSection === 'auth'"
          :api-url="apiUrl"
          @open-keys="keysModalOpen = true"
        />

        <section v-else-if="activeModule" class="module-section">
          <div class="module-head">
            <h2 class="module-title">{{ activeModule.badge }} · {{ activeModule.title }}</h2>
            <p class="module-sub">{{ activeModule.subtitle }}</p>
          </div>

          <div class="endpoints-stack">
            <ApiEndpointCard
              v-for="(ep, idx) in activeModule.endpoints"
              :key="ep.id"
              :endpoint="ep"
              :api-url="apiUrl"
              :default-open="idx === 0"
            />
          </div>
        </section>
      </main>
    </div>

    <BaseModal v-model="keysModalOpen" title="Gerenciar Chaves de API" size="lg">
      <SettingsApiKeysTab />
    </BaseModal>
  </div>
</template>

<style scoped>
.api-docs-page { display: flex; flex-direction: column; gap: 1.35rem; }
.page-top { display: flex; align-items: flex-start; justify-content: space-between; flex-wrap: wrap; gap: 1rem; padding-bottom: 1rem; border-bottom: 1px solid var(--border); }
.page-title { font-family: var(--font-sans); font-size: var(--fs-3xl); font-weight: var(--fw-bold); color: var(--text-strong); margin-bottom: 0.2rem; }
.page-subtitle { font-size: var(--fs-lg); color: var(--text-2); }
.page-subtitle code { font-family: var(--font-mono); color: var(--brand); background: var(--brand-soft); padding: 0.1rem 0.4rem; border-radius: 5px; }
.top-actions { display: flex; align-items: center; flex-wrap: wrap; gap: 0.65rem; }
.base-pill { display: inline-flex; align-items: center; gap: 0.45rem; padding: 0.45rem 0.75rem; background: var(--surface-deep); border: 1px solid var(--border); border-radius: 8px; color: var(--text-2); font-size: var(--fs-sm); cursor: pointer; }
.base-pill:hover { border-color: var(--brand-border); }
.base-pill code { font-family: var(--font-mono); color: var(--text-1); }
.base-pill-label { color: var(--text-3); }
.icon-sm { width: 0.9rem; height: 0.9rem; }
.text-brand { color: var(--brand); }
.docs-layout { display: flex; align-items: flex-start; gap: 1.5rem; }
.docs-content { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 1.5rem; }
.module-section { display: flex; flex-direction: column; gap: 0.85rem; }
.module-head { margin-bottom: 0.15rem; }
.module-title { font-family: var(--font-sans); font-size: var(--fs-2xl); font-weight: var(--fw-bold); color: var(--text-strong); margin-bottom: 0.2rem; }
.module-sub { font-size: var(--fs-md); color: var(--text-2); }
.endpoints-stack { display: flex; flex-direction: column; gap: 0.65rem; }
.empty-search { background: var(--surface-2); border: 1px solid var(--border); border-radius: 12px; padding: 2rem; text-align: center; color: var(--text-2); font-size: var(--fs-lg); }
@media (max-width: 1023px) {
  .docs-layout { flex-direction: column; }
}
</style>
