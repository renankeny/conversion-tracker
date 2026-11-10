<template>
  <div class="filter-strip">
    <div class="search-box">
      <Search :size="16" class="search-icon" />
      <input
        type="text"
        :value="searchQuery"
        placeholder="Buscar lead por nome, e-mail ou telefone..."
        class="search-input"
        @input="emit('update:searchQuery', ($event.target as HTMLInputElement).value)"
      />
      <button v-if="searchQuery" type="button" class="clear-search-btn" @click="emit('update:searchQuery', '')">
        <X :size="14" />
      </button>
    </div>

    <div class="filter-dropdowns">
      <div class="filter-dropdown-item">
        <BaseDropdown
          :model-value="selectedTagFilter"
          size="sm"
          placeholder="Todas as Tags"
          :options="[
            { value: '', label: 'Todas as Tags' },
            ...allTags.map((tag) => ({ value: String(tag.id), label: tag.name }))
          ]"
          @update:model-value="emit('update:selectedTagFilter', String($event ?? ''))"
        />
      </div>

      <div class="filter-dropdown-item">
        <BaseDropdown
          :model-value="selectedOriginFilter"
          size="sm"
          placeholder="Todas as Origens"
          :options="originOptions"
          @update:model-value="emit('update:selectedOriginFilter', String($event ?? ''))"
        />
      </div>

      <BaseButton
        v-if="isFilterActive"
        variant="ghost"
        size="sm"
        class="clear-all-filters"
        @click="emit('clear')"
      >
        Limpar Filtros
      </BaseButton>

      <span v-if="isFilterActive" class="filtered-count">
        {{ filteredTotalCount }} de {{ totalLeadsCount }} leads
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Search, X } from 'lucide-vue-next'
import type { CrmTag } from '#shared/types'

defineProps<{
  searchQuery: string
  selectedTagFilter: string
  selectedOriginFilter: string
  allTags: CrmTag[]
  isFilterActive: boolean
  filteredTotalCount: number
  totalLeadsCount: number
}>()

const emit = defineEmits<{
  (e: 'update:searchQuery', val: string): void
  (e: 'update:selectedTagFilter', val: string): void
  (e: 'update:selectedOriginFilter', val: string): void
  (e: 'clear'): void
}>()

const originOptions = [
  { value: '', label: 'Todas as Origens' },
  { value: 'meta', label: 'Meta Ads & CTWA' },
  { value: 'google', label: 'Google Ads' },
  { value: 'tiktok', label: 'TikTok' },
  { value: 'direct', label: 'Direto / Orgânico' },
  { value: 'other', label: 'Outras Origens' },
]
</script>

<style scoped>
.filter-strip {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  flex-wrap: wrap;
  background: var(--surface-deep);
  padding: 0.625rem 1rem;
  border-radius: 10px;
  border: 1px solid var(--border-soft);
}

.search-box {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  background: var(--surface-1);
  border: 1px solid var(--border-soft);
  border-radius: 8px;
  padding: 0.375rem 0.75rem;
  flex: 1;
  min-width: 250px;
  max-width: 400px;
}

.search-icon {
  color: var(--text-3);
}

.search-input {
  border: none;
  outline: none;
  background: transparent;
  color: var(--text-strong);
  font-size: var(--fs-base);
  width: 100%;
}

.clear-search-btn {
  background: none;
  border: none;
  color: var(--text-2);
  cursor: pointer;
  display: flex;
  align-items: center;
  padding: 2px;
}

.filter-dropdowns {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.filter-dropdown-item {
  min-width: 170px;
}

.clear-all-filters {
  font-size: var(--fs-sm);
  color: var(--danger);
}

.filtered-count {
  font-size: var(--fs-sm);
  color: var(--text-2);
  font-weight: var(--fw-medium);
}

@media (max-width: 768px) {
  .filter-strip {
    padding: 0.625rem;
    gap: 0.625rem;
    flex-direction: column;
    align-items: stretch;
  }

  .search-box {
    min-width: 100%;
    max-width: 100%;
    width: 100%;
  }

  .filter-dropdowns {
    width: 100%;
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
  }

  .filter-dropdown-item {
    flex: 1 1 calc(50% - 0.25rem);
    min-width: 135px;
  }
}
</style>
