<template>
  <div class="platforms-page">
    <div class="page-header">
      <div class="header-copy">
        <div class="title-row">
          <h1 class="page-title">Plataformas</h1>
          <BaseBadge variant="info">{{ PLATFORMS.length }} integradas</BaseBadge>
        </div>
        <p class="page-subtitle">
          Gateways e checkouts de infoprodutos homologados. Copie a URL do webhook e ative na compra aprovada.
        </p>
      </div>

      <div class="search-box">
        <BaseInput
          v-model="searchQuery"
          placeholder="Buscar plataforma (ex: Hotmart, Kiwify, Eduzz...)"
        />
      </div>
    </div>

    <div v-if="!filteredPlatforms.length" class="empty-state">
      <p class="empty-title">Nenhuma plataforma encontrada para "{{ searchQuery }}"</p>
      <BaseButton variant="ghost" size="sm" @click="searchQuery = ''">Limpar busca</BaseButton>
    </div>

    <div v-else class="platforms-grid">
      <div
        v-for="p in filteredPlatforms"
        :key="p.id"
        class="platform-card"
        @click="openModal(p)"
      >
        <div class="platform-header">
          <div class="platform-logo" :style="{ background: p.brandColor }">
            <img v-if="p.logoUrl" :src="p.logoUrl" :alt="p.name" loading="lazy" />
            <span v-else>{{ p.name.charAt(0) }}</span>
          </div>
          <div class="platform-info">
            <div class="platform-name">{{ p.name }}</div>
            <p class="platform-desc">{{ p.description }}</p>
          </div>
        </div>

        <div class="card-footer">
          <span class="webhook-pill">{{ p.webhookPath }}</span>
          <BaseButton variant="primary" size="sm" @click.stop="openModal(p)">
            Configurar
          </BaseButton>
        </div>
      </div>
    </div>

    <PlatformIntegrateModal
      v-model="showModal"
      v-model:token-value="tokenForm"
      :platform="selectedPlatform"
      :webhook-url="selectedPlatform ? buildURL(selectedPlatform.webhookPath) : ''"
      :copied="Boolean(selectedPlatform && copiedId === selectedPlatform.id)"
      :loading-token="loadingToken"
      :saving-token="savingToken"
      @copy="copy"
      @save-token="saveToken"
    />
  </div>
</template>

<script setup lang="ts">
import { PLATFORMS, type PlatformDef } from '#shared/constants/platforms'
import PlatformIntegrateModal from '~/components/integrations/PlatformIntegrateModal.vue'
import { useAuth } from '~/composables/useAuth'

definePageMeta({ layout: 'app', middleware: 'auth' })
const { formatTitle } = useBranding()
useHead({ title: computed(() => formatTitle('Plataformas')) })

const toast = useToast()
const config = useRuntimeConfig()
const auth = useAuth()
const api = useAPI()
const tenantStore = useTenantStore()

const searchQuery = ref('')
const filteredPlatforms = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  if (!q) return PLATFORMS
  return PLATFORMS.filter(
    (p) =>
      p.name.toLowerCase().includes(q) ||
      p.id.toLowerCase().includes(q) ||
      p.description.toLowerCase().includes(q),
  )
})

const apiBase = computed(() => String(config.public.apiUrl || '') || 'http://localhost:8080/api/v1')

const buildURL = (path: string): string => {
  const base = apiBase.value.replace(/\/+$/, '') + path
  const accId = tenantStore.activeAccountId || auth.user?.value?.account_id
  return accId ? `${base}?acc=${accId}` : base
}

const copiedId = ref<string | null>(null)
const showModal = ref(false)
const selectedPlatform = ref<PlatformDef | null>(null)
const tokenForm = ref('')
const loadingToken = ref(false)
const savingToken = ref(false)

const openModal = async (p: PlatformDef) => {
  selectedPlatform.value = p
  showModal.value = true
  tokenForm.value = ''
  loadingToken.value = true
  try {
    const res = await api.get<{ token_secret: string }>(`/admin/integrations/platforms/tokens/${p.id}`)
    tokenForm.value = res.token_secret || ''
  } catch {
    // sem token prévio cadastrado
  } finally {
    loadingToken.value = false
  }
}

const saveToken = async () => {
  if (!selectedPlatform.value) return
  savingToken.value = true
  try {
    await api.put(`/admin/integrations/platforms/tokens/${selectedPlatform.value.id}`, {
      token_secret: tokenForm.value,
    })
    toast.success('Token salvo', `Validação da ${selectedPlatform.value.name} atualizada.`)
  } catch {
    toast.error('Erro', 'Não foi possível salvar o token.')
  } finally {
    savingToken.value = false
  }
}

const copy = async (p: PlatformDef) => {
  const url = buildURL(p.webhookPath)
  try {
    await navigator.clipboard.writeText(url)
    copiedId.value = p.id
    toast.success('URL copiada', `Cole no painel da ${p.name}`)
    setTimeout(() => {
      if (copiedId.value === p.id) copiedId.value = null
    }, 2000)
  } catch {
    toast.error('Erro', 'Não foi possível copiar — selecione manualmente.')
  }
}
</script>

<style scoped>
.platforms-page { display: flex; flex-direction: column; gap: 1.5rem; }
.page-header { display: flex; justify-content: space-between; align-items: flex-end; gap: 1rem; flex-wrap: wrap; }
.header-copy { display: flex; flex-direction: column; gap: 0.25rem; }
.title-row { display: flex; align-items: center; gap: 0.625rem; }
.page-title { font-family: var(--font-heading); font-size: var(--fs-3xl); font-weight: var(--fw-bold); color: var(--text-strong); }
.page-subtitle { font-family: var(--font-sans); color: var(--text-2); font-size: var(--fs-lg); max-width: 680px; }
.search-box { width: 100%; max-width: 320px; }
.platforms-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(275px, 1fr)); gap: 1rem; }
.platform-card { background: var(--surface-2); border: 1px solid var(--border); border-radius: 14px; padding: 1.125rem; display: flex; flex-direction: column; justify-content: space-between; gap: 1rem; box-shadow: var(--shadow-card); transition: border-color 0.18s ease, transform 0.18s ease; cursor: pointer; }
.platform-card:hover { transform: translateY(-2px); border-color: var(--brand-border); }
.platform-header { display: flex; gap: 0.875rem; align-items: flex-start; }
.platform-logo { flex-shrink: 0; width: 48px; height: 48px; border-radius: 12px; display: flex; align-items: center; justify-content: center; font-family: var(--font-heading); font-size: var(--fs-2xl); font-weight: var(--fw-bold); color: var(--text-strong); overflow: hidden; border: 1px solid var(--border); }
.platform-logo img { width: 100%; height: 100%; object-fit: cover; background: var(--surface-1); }
.platform-info { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 0.2rem; }
.platform-name { font-family: var(--font-heading); font-size: var(--fs-lg); font-weight: var(--fw-semibold); color: var(--text-strong); }
.platform-desc { font-family: var(--font-sans); font-size: var(--fs-base); color: var(--text-2); line-height: 1.4; display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; }
.card-footer { display: flex; align-items: center; justify-content: space-between; gap: 0.5rem; padding-top: 0.75rem; border-top: 1px solid var(--border-soft); }
.webhook-pill { font-family: var(--font-mono); font-size: var(--fs-xs); color: var(--text-3); background: var(--surface-soft); padding: 0.2rem 0.5rem; border-radius: 6px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.empty-state { display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 0.75rem; padding: 3rem 1rem; background: var(--surface-2); border: 1px solid var(--border); border-radius: 14px; }
.empty-title { font-family: var(--font-sans); font-size: var(--fs-lg); color: var(--text-2); }
</style>
