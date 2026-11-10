<template>
  <div v-if="countries.length > 0" class="top-countries-strip">
    <span class="strip-label">Principais:</span>
    <div class="country-badges">
      <button
        v-for="item in countries"
        :key="item.code"
        class="country-badge"
        :title="'Girar globo para ' + item.name"
        @click="$emit('focus-country', item.code)"
      >
        <span class="badge-flag">{{ item.flag }}</span>
        <span class="badge-name">{{ item.name }}</span>
        <span class="badge-count">{{ formatCompactNumber(item.count) }}</span>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { formatCompactNumber } from '~/utils/geoCoordinates'

export interface TopCountryItem {
  code: string
  name: string
  flag: string
  count: number
}

defineProps<{
  countries: TopCountryItem[]
}>()

defineEmits<{
  'focus-country': [code: string]
}>()
</script>

<style scoped>
.top-countries-strip {
  display: flex;
  align-items: center;
  gap: 0.625rem;
  margin-top: 0.75rem;
  padding-top: 0.625rem;
  border-top: 1px solid var(--border-soft);
  overflow-x: auto;
  scrollbar-width: none;
}

.top-countries-strip::-webkit-scrollbar {
  display: none;
}

.strip-label {
  font-size: var(--fs-sm);
  font-weight: var(--fw-medium);
  color: var(--text-3);
  white-space: nowrap;
}

.country-badges {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: nowrap;
}

.country-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  background: var(--surface-1);
  border: 1px solid var(--border-soft);
  border-radius: 9999px;
  padding: 0.2rem 0.625rem;
  font-size: var(--fs-sm);
  color: var(--text-1);
  cursor: pointer;
  transition: all 0.15s ease;
  white-space: nowrap;
}

.country-badge:hover {
  background: var(--surface-soft);
  border-color: var(--border);
  transform: translateY(-1px);
}

.badge-flag {
  font-size: var(--fs-base);
}

.badge-name {
  font-weight: var(--fw-medium);
}

.badge-count {
  font-size: var(--fs-xs);
  font-weight: var(--fw-bold);
  color: var(--brand);
  background: var(--brand-soft);
  padding: 0.0625rem 0.375rem;
  border-radius: 9999px;
}
</style>
