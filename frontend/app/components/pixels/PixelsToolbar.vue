<template>
  <div class="pixels-toolbar">
    <div class="kpi-strip">
      <div class="kpi-card">
        <span class="kpi-label">Total de Pixels</span>
        <strong class="kpi-value">{{ totalCount }}</strong>
      </div>
      <div class="kpi-card">
        <span class="kpi-label">Ativos</span>
        <strong class="kpi-value success">{{ activeCount }}</strong>
      </div>
      <div class="kpi-card">
        <span class="kpi-label">Pausados</span>
        <strong class="kpi-value muted">{{ pausedCount }}</strong>
      </div>
    </div>

    <div class="filter-bar">
      <div class="search-box">
        <BaseInput
          :model-value="search"
          placeholder="Buscar por nome, rótulo, domínio ou Pixel ID..."
          @update:model-value="$emit('update:search', $event)"
        />
      </div>
      <div class="status-pills" role="tablist" aria-label="Filtrar status dos pixels">
        <button
          v-for="opt in filterOptions"
          :key="opt.value"
          type="button"
          class="status-pill"
          :class="{ active: statusFilter === opt.value }"
          @click="$emit('update:statusFilter', opt.value)"
        >
          {{ opt.label }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Pixel } from '#shared/types'

export type PixelStatusFilter = 'all' | 'active' | 'paused'

const props = defineProps<{
  pixels: Pixel[]
  search: string
  statusFilter: PixelStatusFilter
}>()

defineEmits<{
  'update:search': [value: string]
  'update:statusFilter': [value: PixelStatusFilter]
}>()

const totalCount = computed(() => props.pixels.length)
const activeCount = computed(() => props.pixels.filter((p) => p.enabled).length)
const pausedCount = computed(() => props.pixels.filter((p) => !p.enabled).length)

const filterOptions: Array<{ value: PixelStatusFilter; label: string }> = [
  { value: 'all', label: 'Todos' },
  { value: 'active', label: 'Ativos' },
  { value: 'paused', label: 'Pausados' },
]
</script>

<style scoped>
.pixels-toolbar { display: flex; flex-direction: column; gap: 1rem; }
.kpi-strip { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 0.75rem; }
@media (max-width: 768px) { .kpi-strip { grid-template-columns: 1fr; } }
.kpi-card { background: var(--surface-2); border: 1px solid var(--border); border-radius: 12px; padding: 0.875rem 1rem; display: flex; flex-direction: column; gap: 0.25rem; }
.kpi-label { font-size: var(--fs-xs); font-weight: var(--fw-medium); color: var(--text-3); text-transform: uppercase; letter-spacing: 0.04em; }
.kpi-value { font-size: var(--fs-2xl); font-weight: var(--fw-bold); color: var(--text-strong); line-height: 1.2; }
.kpi-value.success { color: var(--success); }
.kpi-value.muted { color: var(--text-2); }
.filter-bar { display: flex; align-items: center; justify-content: space-between; gap: 0.75rem; flex-wrap: wrap; }
.search-box { flex: 1; min-width: 240px; max-width: 420px; }
.status-pills { display: inline-flex; align-items: center; gap: 0.25rem; background: var(--surface-2); border: 1px solid var(--border); border-radius: 10px; padding: 0.25rem; overflow-x: auto; }
.status-pill { border: none; background: transparent; color: var(--text-2); font-size: var(--fs-sm); font-weight: var(--fw-medium); padding: 0.375rem 0.75rem; border-radius: 7px; cursor: pointer; transition: all 0.15s; white-space: nowrap; }
.status-pill:hover { color: var(--text-strong); }
.status-pill.active { background: var(--brand-soft); color: var(--brand); font-weight: var(--fw-semibold); }
</style>
