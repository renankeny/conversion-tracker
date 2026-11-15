<template>
  <div class="platforms-page">
    <div class="page-header">
      <div class="header-copy">
        <div class="title-row">
          <h1 class="page-title">Plataformas</h1>
          <BaseBadge variant="default">{{ PLATFORMS.length }} homologadas</BaseBadge>
          <BaseBadge v-if="activeCount > 0" variant="success">
            {{ activeCount }} {{ activeCount === 1 ? 'ativa' : 'ativas' }}
          </BaseBadge>
        </div>
        <p class="page-subtitle">
          Gateways e checkouts de infoprodutos homologados. Copie a URL do webhook, escolha opcionalmente o Pixel do produto e valide via token seguro.
        </p>
      </div>

      <div class="toolbar-controls">
        <div v-if="isGlobalTenantMode && accountOptions.length" class="account-filter">
          <BaseDropdown
            :model-value="localAccountId"
            :options="accountOptions"
            placeholder="Empresa de destino"
            @update:model-value="localAccountId = String($event ?? '')"
          />
        </div>

        <div class="status-filter">
          <BaseDropdown
            :model-value="statusFilter"
            :options="filterOptions"
            placeholder="Todas as plataformas"
            @update:model-value="statusFilter = $event === 'active' ? 'active' : 'all'"
          />
        </div>

        <div class="search-box">
          <BaseInput
            v-model="searchQuery"
            placeholder="Buscar plataforma (ex: Hotmart, Kiwify, Eduzz...)"
          />
        </div>
      </div>
    </div>

    <div v-if="!filteredPlatforms.length" class="empty-state">
      <p class="empty-title">
        {{
          statusFilter === 'active' && !searchQuery
            ? 'Nenhuma plataforma com token ativo ou vendas registradas nesta empresa.'
            : `Nenhuma plataforma encontrada para "${searchQuery}"`
        }}
      </p>
      <BaseButton variant="ghost" size="sm" @click="resetFilters">
        Limpar filtros
      </BaseButton>
    </div>

    <div v-else class="platforms-grid">
      <PlatformCard
        v-for="p in filteredPlatforms"
        :key="p.id"
        :platform="p"
        :summary="summaryItems[p.id]"
        :copied="copiedKey === `card:${p.id}`"
        :can-manage="canManagePixels"
        @copy="copyCardUrl"
        @configure="openModal"
      />
    </div>

    <PlatformIntegrateModal
      v-model="showModal"
      v-model:token-value="tokenInput"
      v-model:selected-pixel="selectedPixelSlug"
      v-model:account-id="localAccountId"
      :platform="selectedPlatform"
      :webhook-url="modalWebhookUrl"
      :copied="Boolean(selectedPlatform && copiedKey === `modal:${selectedPlatform.id}`)"
      :has-saved-token="hasSavedToken"
      :masked-token="maskedToken"
      :loading-token="loadingToken"
      :saving-token="savingToken"
      :can-manage="canManagePixels"
      :pixel-options="pixelOptions"
      :has-multiple-pixels="hasMultiplePixels"
      :is-global-tenant-mode="isGlobalTenantMode"
      :account-options="accountOptions"
      @copy="copyModalUrl"
      @save-token="saveToken"
      @clear-token="clearToken"
    />
  </div>
</template>

<script setup lang="ts">
import { PLATFORMS } from '#shared/constants/platforms'
import PlatformCard from '~/components/integrations/PlatformCard.vue'
import PlatformIntegrateModal from '~/components/integrations/PlatformIntegrateModal.vue'
import { usePlatformsPage } from '~/composables/usePlatformsPage'

definePageMeta({ layout: 'app', middleware: 'auth' })

const { formatTitle } = useBranding()
useHead({ title: computed(() => formatTitle('Plataformas')) })

const {
  canManagePixels,
  accountOptions,
  isGlobalTenantMode,
  localAccountId,
  searchQuery,
  statusFilter,
  summaryItems,
  activeCount,
  filteredPlatforms,
  pixelOptions,
  hasMultiplePixels,
  selectedPixelSlug,
  showModal,
  selectedPlatform,
  modalWebhookUrl,
  tokenInput,
  hasSavedToken,
  maskedToken,
  loadingToken,
  savingToken,
  copiedKey,
  openModal,
  saveToken,
  clearToken,
  copyCardUrl,
  copyModalUrl,
} = usePlatformsPage()

const filterOptions = [
  { value: 'all', label: 'Todas as plataformas' },
  { value: 'active', label: 'Ativas ou com token' },
]

const resetFilters = () => {
  searchQuery.value = ''
  statusFilter.value = 'all'
}
</script>

<style scoped>
.platforms-page { display: flex; flex-direction: column; gap: 1.5rem; }
.page-header { display: flex; justify-content: space-between; align-items: flex-end; gap: 1rem; flex-wrap: wrap; }
.header-copy { display: flex; flex-direction: column; gap: 0.25rem; }
.title-row { display: flex; align-items: center; gap: 0.5rem; flex-wrap: wrap; }
.page-title { font-family: var(--font-heading); font-size: var(--fs-3xl); font-weight: var(--fw-bold); color: var(--text-strong); }
.page-subtitle { font-family: var(--font-sans); color: var(--text-2); font-size: var(--fs-lg); max-width: 680px; }
.toolbar-controls { display: flex; align-items: center; gap: 0.625rem; flex-wrap: wrap; }
.account-filter { min-width: 190px; }
.status-filter { min-width: 195px; }
.search-box { width: 280px; max-width: 100%; }
.platforms-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 1rem; }
.empty-state { display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 0.75rem; padding: 3rem 1rem; background: var(--surface-2); border: 1px solid var(--border); border-radius: 14px; text-align: center; }
.empty-title { font-family: var(--font-sans); font-size: var(--fs-lg); color: var(--text-2); }

@media (max-width: 640px) {
  .toolbar-controls { width: 100%; }
  .account-filter, .status-filter, .search-box { width: 100%; }
}
</style>
