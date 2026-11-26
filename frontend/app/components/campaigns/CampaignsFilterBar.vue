<template>
  <section class="filter-bar">
    <div class="filter-row-top">
      <div class="platform-tabs">
        <button
          v-for="opt in platformTabs"
          :key="opt.value"
          type="button"
          class="platform-pill"
          :class="{ active: platform === opt.value }"
          @click="emit('update:platform', opt.value)"
        >
          <component :is="opt.icon" v-if="opt.icon" :size="14" />
          <span>{{ opt.label }}</span>
        </button>
      </div>

      <div class="top-actions">
        <BaseButton variant="outline" size="sm" @click="emit('open-financials')">
          <Percent :size="14" />
          <span>Taxas &amp; Custos</span>
        </BaseButton>
        <BaseButton variant="primary" size="sm" :loading="loading" @click="emit('refresh')">
          <RefreshCw :size="14" />
          <span>Atualizar</span>
        </BaseButton>
      </div>
    </div>

    <div class="filter-controls">
      <div class="ctrl-item">
        <PixelDropdown
          :model-value="pixelName"
          label="Pixel"
          :allow-all="true"
          @update:model-value="emit('update:pixelName', $event)"
        />
      </div>

      <div class="ctrl-item ctrl-account">
        <BaseAdAccountSelect
          :model-value="selectedAdAccount"
          label="Conta de Anúncios"
          placeholder="Automático (do Pixel ou conta ativa)"
          :options="adAccountOptions"
          :loading="loadingAccounts"
          @update:model-value="emit('update:selectedAdAccount', String($event ?? ''))"
        />
      </div>

      <div class="ctrl-item">
        <BaseDropdown
          :model-value="periodPreset"
          label="Período"
          :options="periodOptions"
          @update:model-value="emit('update:periodPreset', String($event ?? '7d'))"
        />
      </div>

      <div v-if="periodPreset === 'custom'" class="ctrl-item ctrl-dates">
        <BaseInput
          :model-value="customSince"
          type="date"
          label="Início"
          @update:model-value="emit('update:customSince', $event)"
        />
        <BaseInput
          :model-value="customUntil"
          type="date"
          label="Fim"
          @update:model-value="emit('update:customUntil', $event)"
        />
      </div>

      <div class="ctrl-item">
        <BaseDropdown
          :model-value="attributionMode"
          label="Atribuição"
          :options="attributionOptions"
          @update:model-value="emit('update:attributionMode', ($event as CampaignAttributionMode) || 'last_click')"
        />
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import type { Component } from 'vue'
import { Percent, RefreshCw } from 'lucide-vue-next'
import type { AdAccountSelectOption } from '~/components/BaseAdAccountSelect.vue'
import type { CampaignAttributionMode, CampaignPlatform } from '#shared/types'
import { MetaAdsIcon, GoogleAdsIcon } from '~/utils/adIcons'

defineProps<{
  platform: CampaignPlatform
  pixelName: string
  selectedAdAccount: string
  adAccountOptions: AdAccountSelectOption[]
  loadingAccounts: boolean
  periodPreset: string
  customSince: string
  customUntil: string
  attributionMode: CampaignAttributionMode
  loading: boolean
}>()

const emit = defineEmits<{
  'update:platform': [value: CampaignPlatform]
  'update:pixelName': [value: string]
  'update:selectedAdAccount': [value: string]
  'update:periodPreset': [value: string]
  'update:customSince': [value: string]
  'update:customUntil': [value: string]
  'update:attributionMode': [value: CampaignAttributionMode]
  'open-financials': []
  refresh: []
}>()

const platformTabs: Array<{ value: CampaignPlatform; label: string; icon?: Component }> = [
  { value: 'all', label: 'Todas' },
  { value: 'meta', label: 'Meta Ads', icon: MetaAdsIcon },
  { value: 'google', label: 'Google Ads', icon: GoogleAdsIcon },
]

const periodOptions = [
  { value: 'today', label: 'Hoje' },
  { value: 'yesterday', label: 'Ontem' },
  { value: '7d', label: 'Últimos 7 dias' },
  { value: '30d', label: 'Últimos 30 dias' },
  { value: 'month', label: 'Este mês' },
  { value: 'custom', label: 'Personalizado' },
]

const attributionOptions = [
  { value: 'last_click', label: 'Último clique (Last Click)' },
  { value: 'first_click', label: 'Primeiro clique (First Click)' },
]
</script>

<style scoped>
.filter-bar {
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.9rem;
}
.filter-row-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  flex-wrap: wrap;
}
.platform-tabs {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  background: var(--surface-1);
  border: 1px solid var(--border);
  border-radius: 9px;
  padding: 0.2rem;
}
.platform-pill {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.42rem 0.78rem;
  border-radius: 7px;
  border: none;
  background: transparent;
  color: var(--text-2);
  font-size: var(--fs-base);
  font-weight: var(--fw-medium);
  cursor: pointer;
  transition: all 0.15s ease;
}
.platform-pill:hover {
  color: var(--text-1);
}
.platform-pill.active {
  background: var(--brand-soft);
  color: var(--brand);
  font-weight: var(--fw-semibold);
}
.top-actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}
.filter-controls {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(190px, 1fr));
  gap: 0.75rem;
  align-items: end;
}
.ctrl-dates {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.5rem;
}
</style>
