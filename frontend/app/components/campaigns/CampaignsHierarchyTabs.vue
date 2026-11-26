<template>
  <div class="hierarchy-bar">
    <div class="tabs-left">
      <button
        v-for="tab in tabs"
        :key="tab.key"
        type="button"
        class="hierarchy-tab"
        :class="{ active: activeTab === tab.key }"
        @click="emit('update:activeTab', tab.key)"
      >
        <span>{{ tab.label }}</span>
        <span class="tab-count">{{ tab.count }}</span>
      </button>

      <div v-if="selectedCampaignIds.length || selectedAdSetIds.length" class="selection-chips">
        <button
          v-if="selectedCampaignIds.length"
          type="button"
          class="sel-chip"
          @click="emit('clear-campaigns')"
        >
          <span>{{ selectedCampaignIds.length }} campanha(s)</span>
          <X :size="12" />
        </button>
        <button
          v-if="selectedAdSetIds.length"
          type="button"
          class="sel-chip"
          @click="emit('clear-adsets')"
        >
          <span>{{ selectedAdSetIds.length }} conjunto(s)</span>
          <X :size="12" />
        </button>
      </div>
    </div>

    <div class="tabs-right">
      <div class="status-select">
        <BaseDropdown
          :model-value="statusFilter"
          :options="statusOptions"
          @update:model-value="emit('update:statusFilter', String($event ?? 'all'))"
        />
      </div>
      <div class="search-wrap">
        <BaseInput
          :model-value="searchQuery"
          placeholder="Buscar por nome ou ID..."
          @update:model-value="emit('update:searchQuery', $event)"
        >
          <template #icon>
            <Search :size="14" />
          </template>
        </BaseInput>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Search, X } from 'lucide-vue-next'

export type HierarchyTabKey = 'campaigns' | 'adsets' | 'ads'

const props = defineProps<{
  activeTab: HierarchyTabKey
  campaignsCount: number
  adSetsCount: number
  adsCount: number
  selectedCampaignIds: string[]
  selectedAdSetIds: string[]
  statusFilter: string
  searchQuery: string
}>()

const emit = defineEmits<{
  'update:activeTab': [value: HierarchyTabKey]
  'update:statusFilter': [value: string]
  'update:searchQuery': [value: string]
  'clear-campaigns': []
  'clear-adsets': []
}>()

const tabs = computed<Array<{ key: HierarchyTabKey; label: string; count: number }>>(() => [
  { key: 'campaigns', label: 'Campanhas', count: props.campaignsCount },
  { key: 'adsets', label: 'Conjuntos / Grupos', count: props.adSetsCount },
  { key: 'ads', label: 'Anúncios', count: props.adsCount },
])

const statusOptions = [
  { value: 'all', label: 'Todos os status' },
  { value: 'ACTIVE', label: 'Apenas ativos' },
  { value: 'PAUSED', label: 'Apenas pausados' },
]
</script>

<style scoped>
.hierarchy-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  flex-wrap: wrap;
}
.tabs-left {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  flex-wrap: wrap;
}
.hierarchy-tab {
  display: inline-flex;
  align-items: center;
  gap: 0.45rem;
  padding: 0.52rem 0.85rem;
  border-radius: 9px;
  border: 1px solid var(--border);
  background: var(--surface-2);
  color: var(--text-2);
  font-size: var(--fs-md);
  font-weight: var(--fw-medium);
  cursor: pointer;
  transition: all 0.15s ease;
}
.hierarchy-tab:hover {
  border-color: var(--brand-border);
  color: var(--text-1);
}
.hierarchy-tab.active {
  background: var(--brand-soft);
  border-color: var(--brand-border);
  color: var(--brand);
  font-weight: var(--fw-semibold);
}
.tab-count {
  font-size: var(--fs-xs);
  font-weight: var(--fw-semibold);
  padding: 0.08rem 0.4rem;
  border-radius: 999px;
  background: var(--surface-soft);
  color: inherit;
}
.selection-chips {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  margin-left: 0.35rem;
}
.sel-chip {
  display: inline-flex;
  align-items: center;
  gap: 0.3rem;
  padding: 0.25rem 0.55rem;
  border-radius: 999px;
  border: 1px solid var(--brand-border);
  background: var(--brand-soft);
  color: var(--brand);
  font-size: var(--fs-xs);
  font-weight: var(--fw-semibold);
  cursor: pointer;
}
.tabs-right {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
}
.status-select {
  width: 165px;
}
.search-wrap {
  width: 230px;
}
@media (max-width: 640px) {
  .tabs-right, .status-select, .search-wrap {
    width: 100%;
  }
}
</style>
