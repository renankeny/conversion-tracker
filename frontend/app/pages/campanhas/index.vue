<template>
  <div class="campaigns-page">
    <header class="page-header">
      <div>
        <h1 class="page-title">Campanhas</h1>
        <p class="page-subtitle">
          Atribuição real multi-nível (Meta Graph API v26.0 + Google Ads API v26) com receita líquida, reembolsos e lucro real.
        </p>
      </div>
      <div class="header-right">
        <span v-if="overview?.cached" class="cache-pill">Cache ativo (3 min)</span>
        <NuxtLink to="/campanhas/regras" class="rules-shortcut">
          <Bot :size="14" />
          <span>Regras Automáticas</span>
        </NuxtLink>
      </div>
    </header>

    <CampaignsFilterBar
      v-model:platform="platform"
      v-model:pixel-name="pixelName"
      v-model:selected-ad-account="selectedAdAccount"
      v-model:period-preset="periodPreset"
      v-model:custom-since="customSince"
      v-model:custom-until="customUntil"
      v-model:attribution-mode="attributionMode"
      :ad-account-options="adAccountOptions"
      :loading-accounts="loadingAccounts"
      :loading="loading"
      @open-financials="financialsModalOpen = true"
      @refresh="fetchOverview(true)"
    />

    <div v-if="error" class="alert-box is-danger">
      <AlertCircle :size="16" />
      <span>{{ error }}</span>
    </div>

    <div v-if="overview?.warnings?.length" class="alert-box is-warning">
      <AlertTriangle :size="16" />
      <span>{{ overview.warnings.join(' · ') }}</span>
    </div>

    <CampaignsKpiStrip :summary="summary" :currency="currency" />

    <CampaignsHierarchyTabs
      v-model:active-tab="activeTab"
      v-model:status-filter="statusFilter"
      v-model:search-query="searchQuery"
      :campaigns-count="visibleCampaigns.length"
      :ad-sets-count="visibleAdSets.length"
      :ads-count="visibleAds.length"
      :selected-campaign-ids="selectedCampaignIds"
      :selected-ad-set-ids="selectedAdSetIds"
      @clear-campaigns="selectedCampaignIds = []"
      @clear-adsets="selectedAdSetIds = []"
    />

    <CampaignsTable
      :rows="activeRows"
      :active-tab="activeTab"
      :currency="currency"
      :loading="loading"
      :selected-ids="activeTab === 'campaigns' ? selectedCampaignIds : selectedAdSetIds"
      :mutating-ids="mutatingIds"
      @toggle-select="toggleSelectRow"
      @toggle-status="toggleRowStatus"
      @edit-budget="budgetRow = $event"
      @drill-down="drillDownRow"
    />

    <CampaignBudgetModal
      :row="budgetRow"
      :currency="currency"
      :saving="savingBudget"
      @close="budgetRow = null"
      @save="onSaveBudget"
    />

    <PixelFinancialConfigModal
      v-model="financialsModalOpen"
      :pixels="pixelStore.pixels"
      :active-pixel-name="pixelName"
      :saving="savingFinancials"
      @save="onSaveFinancials"
    />
  </div>
</template>

<script setup lang="ts">
import { AlertCircle, AlertTriangle, Bot } from 'lucide-vue-next'
import type { CampaignRow, PixelFinancialConfig } from '#shared/types'
import CampaignsFilterBar from '~/components/campaigns/CampaignsFilterBar.vue'
import CampaignsKpiStrip from '~/components/campaigns/CampaignsKpiStrip.vue'
import CampaignsHierarchyTabs from '~/components/campaigns/CampaignsHierarchyTabs.vue'
import CampaignsTable from '~/components/campaigns/CampaignsTable.vue'
import CampaignBudgetModal from '~/components/campaigns/CampaignBudgetModal.vue'
import PixelFinancialConfigModal from '~/components/campaigns/PixelFinancialConfigModal.vue'

definePageMeta({ layout: 'app', middleware: 'auth' })

const { formatTitle } = useBranding()
useHead({ title: computed(() => formatTitle('Campanhas')) })

const {
  pixelStore, platform, pixelName, selectedAdAccount, periodPreset, customSince, customUntil,
  attributionMode, activeTab, statusFilter, searchQuery, selectedCampaignIds, selectedAdSetIds,
  adAccountOptions, loadingAccounts, overview, loading, error, mutatingIds, savingBudget, savingFinancials,
  visibleCampaigns, visibleAdSets, visibleAds, activeRows, summary, currency,
  loadAccounts, fetchOverview, toggleSelectRow, drillDownRow, toggleRowStatus, saveRowBudget, savePixelFinancials,
} = useCampaignsManager()

const budgetRow = ref<CampaignRow | null>(null)
const financialsModalOpen = ref(false)

onMounted(async () => {
  await Promise.all([pixelStore.fetch(), loadAccounts()])
  await fetchOverview(false)
})

const onSaveBudget = async (row: CampaignRow, dailyBudget: number) => {
  if (await saveRowBudget(row, dailyBudget)) {
    budgetRow.value = null
  }
}

const onSaveFinancials = async (pixelId: number, cfg: PixelFinancialConfig) => {
  if (await savePixelFinancials(pixelId, cfg)) {
    financialsModalOpen.value = false
  }
}
</script>

<style scoped>
.campaigns-page { display: flex; flex-direction: column; gap: 1rem; padding-bottom: 2rem; }
.page-header { display: flex; align-items: flex-start; justify-content: space-between; gap: 1rem; flex-wrap: wrap; }
.header-right { display: flex; align-items: center; gap: 0.5rem; flex-wrap: wrap; }
.page-title { margin: 0; font-size: var(--fs-2xl); font-weight: var(--fw-bold); color: var(--text-strong); }
.page-subtitle { margin: 0.25rem 0 0; font-size: var(--fs-base); color: var(--text-3); }
.cache-pill {
  font-size: var(--fs-xs); font-weight: var(--fw-medium); padding: 0.25rem 0.6rem;
  border-radius: 999px; background: var(--surface-2); border: 1px solid var(--border); color: var(--text-3);
}
.rules-shortcut {
  display: inline-flex; align-items: center; gap: 0.4rem; padding: 0.375rem 0.75rem; border-radius: 8px;
  background: var(--brand-soft); border: 1px solid var(--brand-border); color: var(--brand);
  font-size: var(--fs-sm); font-weight: var(--fw-semibold); text-decoration: none; transition: all 0.15s ease;
}
.rules-shortcut:hover { background: var(--brand); color: var(--brand-text); }
.alert-box {
  display: flex; align-items: center; gap: 0.55rem; padding: 0.7rem 0.9rem;
  border-radius: 10px; font-size: var(--fs-base);
}
.alert-box.is-danger { background: var(--danger-soft); border: 1px solid var(--danger-border); color: var(--danger); }
.alert-box.is-warning { background: var(--warning-soft); border: 1px solid var(--warning-border); color: var(--warning); }
</style>
