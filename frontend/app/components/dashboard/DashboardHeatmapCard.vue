<template>
  <div class="heatmap-card">
    <div class="heatmap-head">
      <div class="head-left">
        <div class="head-icon"><Flame :size="15" /></div>
        <div>
          <h3 class="head-title">Mapa de Calor — Dia × Hora</h3>
          <p class="head-sub">{{ summaryText }}</p>
        </div>
      </div>

      <div class="mode-switch">
        <button
          v-for="tab in tabs"
          :key="tab.key"
          type="button"
          class="mode-btn"
          :class="{ 'is-active': activeMode === tab.key }"
          @click="activeMode = tab.key"
        >
          {{ tab.label }}
        </button>
      </div>
    </div>

    <div class="heatmap-scroll">
      <div class="heatmap-grid">
        <div class="corner-cell" />
        <div v-for="h in 24" :key="`hr-${h - 1}`" class="hour-label">
          {{ (h - 1) % 3 === 0 ? `${String(h - 1).padStart(2, '0')}h` : '' }}
        </div>

        <template v-for="(day, dIdx) in orderedDays" :key="day.dow">
          <div class="day-label">{{ day.label }}</div>
          <div
            v-for="h in 24"
            :key="`c-${dIdx}-${h - 1}`"
            class="heat-cell"
            :class="cellIntensityClass(cellCount(day.dow, h - 1))"
            :title="`${day.full} às ${String(h - 1).padStart(2, '0')}h: ${cellCount(day.dow, h - 1)} ${unitLabel}`"
          />
        </template>
      </div>
    </div>

    <div class="heatmap-legend">
      <span class="legend-txt">Menor volume</span>
      <span class="heat-swatch heat-0" />
      <span class="heat-swatch heat-1" />
      <span class="heat-swatch heat-2" />
      <span class="heat-swatch heat-3" />
      <span class="heat-swatch heat-4" />
      <span class="legend-txt">Pico</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Flame } from 'lucide-vue-next'
import { useDashboardStore } from '~/stores/useDashboardStore'
import type { DashboardHeatmap, HeatmapBucket } from '#shared/types'

const props = defineProps<{
  heatmap?: DashboardHeatmap | null
}>()

type ModeKey = 'messages' | 'leads' | 'sales'
const dashboardStore = useDashboardStore()
const activeMode = computed<ModeKey>({
  get: () => dashboardStore.heatmapMode,
  set: (v) => { dashboardStore.heatmapMode = v },
})

const tabs: Array<{ key: ModeKey; label: string }> = [
  { key: 'messages', label: 'Mensagens' },
  { key: 'leads', label: 'Leads' },
  { key: 'sales', label: 'Vendas' },
]

// Postgres EXTRACT(DOW): 0 = Dom, 1 = Seg ... 6 = Sáb -> Exibimos Seg..Dom
const orderedDays = [
  { dow: 1, label: 'Seg', full: 'Segunda-feira' },
  { dow: 2, label: 'Ter', full: 'Terça-feira' },
  { dow: 3, label: 'Qua', full: 'Quarta-feira' },
  { dow: 4, label: 'Qui', full: 'Quinta-feira' },
  { dow: 5, label: 'Sex', full: 'Sexta-feira' },
  { dow: 6, label: 'Sáb', full: 'Sábado' },
  { dow: 0, label: 'Dom', full: 'Domingo' },
]

const activeBuckets = computed<HeatmapBucket[]>(() => {
  if (!props.heatmap) return []
  return props.heatmap[activeMode.value] || []
})

const unitLabel = computed(() => {
  if (activeMode.value === 'messages') return 'mensagens'
  if (activeMode.value === 'sales') return 'vendas'
  return 'leads'
})

const lookup = computed(() => {
  const map = new Map<string, number>()
  for (const b of activeBuckets.value) {
    map.set(`${b.weekday}:${b.hour}`, (map.get(`${b.weekday}:${b.hour}`) || 0) + b.count)
  }
  return map
})

const maxCount = computed(() => {
  let max = 0
  for (const v of lookup.value.values()) {
    if (v > max) max = v
  }
  return max
})

const cellCount = (dow: number, hour: number): number => lookup.value.get(`${dow}:${hour}`) || 0

const cellIntensityClass = (count: number): string => {
  if (count <= 0 || maxCount.value <= 0) return 'heat-0'
  const ratio = count / maxCount.value
  if (ratio <= 0.25) return 'heat-1'
  if (ratio <= 0.5) return 'heat-2'
  if (ratio <= 0.75) return 'heat-3'
  return 'heat-4'
}

const summaryText = computed(() => {
  let total = 0
  let bestKey = ''
  let bestVal = 0
  for (const [k, v] of lookup.value.entries()) {
    total += v
    if (v > bestVal) { bestVal = v; bestKey = k }
  }
  if (total === 0) return `Sem ${unitLabel.value} registrados neste período (horário de Brasília)`
  const [dowStr = '1', hrStr = '0'] = bestKey.split(':')
  const dayObj = orderedDays.find((d) => d.dow === Number(dowStr))
  return `${new Intl.NumberFormat('pt-BR').format(total)} ${unitLabel.value} · Pico: ${dayObj?.label || ''} às ${hrStr.padStart(2, '0')}h (${bestVal})`
})
</script>

<style scoped>
.heatmap-card { background: var(--surface-2); border: 1px solid var(--border); border-radius: 12px; padding: 1.25rem; display: flex; flex-direction: column; gap: 0.9rem; }
.heatmap-head { display: flex; align-items: center; justify-content: space-between; gap: 0.75rem; flex-wrap: wrap; }
.head-left { display: flex; align-items: center; gap: 0.65rem; }
.head-icon { width: 30px; height: 30px; border-radius: 8px; background: var(--brand-soft); color: var(--brand); display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
.head-title { font-family: var(--font-heading); font-size: var(--fs-md); font-weight: var(--fw-semibold); color: var(--text-strong); margin: 0; }
.head-sub { font-size: var(--fs-xs); color: var(--text-3); margin: 0.12rem 0 0; }

.mode-switch { display: inline-flex; padding: 0.2rem; border-radius: 8px; background: var(--surface-deep); border: 1px solid var(--border-soft); gap: 0.15rem; }
.mode-btn { padding: 0.28rem 0.65rem; border-radius: 6px; font-size: var(--fs-xs); font-weight: var(--fw-medium); color: var(--text-3); background: transparent; border: none; cursor: pointer; transition: all 0.15s ease; }
.mode-btn:hover { color: var(--text-1); }
.mode-btn.is-active { background: var(--brand-soft); color: var(--brand); font-weight: var(--fw-semibold); }

.heatmap-scroll { overflow-x: auto; padding-bottom: 0.2rem; }
.heatmap-grid { display: grid; grid-template-columns: 34px repeat(24, minmax(18px, 1fr)); gap: 4px; min-width: 540px; align-items: center; }
.hour-label { font-size: var(--fs-2xs); color: var(--text-3); font-family: var(--font-mono); text-align: center; height: 14px; }
.day-label { font-size: var(--fs-xs); color: var(--text-2); font-weight: var(--fw-medium); padding-right: 0.35rem; }

.heat-cell { aspect-ratio: 1.35 / 1; min-height: 16px; border-radius: 4px; transition: transform 0.12s ease; }
.heat-cell:hover { transform: scale(1.14); z-index: 1; outline: 1px solid var(--brand); }
.heat-0, .heat-swatch.heat-0 { background: var(--surface-deep); border: 1px solid var(--border-soft); }
.heat-1, .heat-swatch.heat-1 { background: color-mix(in srgb, var(--brand) 22%, var(--surface-deep)); }
.heat-2, .heat-swatch.heat-2 { background: color-mix(in srgb, var(--brand) 46%, var(--surface-deep)); }
.heat-3, .heat-swatch.heat-3 { background: color-mix(in srgb, var(--brand) 72%, var(--surface-deep)); }
.heat-4, .heat-swatch.heat-4 { background: var(--brand); }

.heatmap-legend { display: flex; align-items: center; justify-content: flex-end; gap: 0.35rem; }
.legend-txt { font-size: var(--fs-2xs); color: var(--text-3); margin: 0 0.15rem; }
.heat-swatch { width: 12px; height: 12px; border-radius: 3px; display: inline-block; }
</style>

