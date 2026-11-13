<template>
  <div class="geo-card">
    <div class="card-header">
      <div class="title-group">
        <h3 class="card-title">Distribuição Geográfica Global</h3>
        <span v-if="activeCountryCount > 0" class="card-badge">
          {{ activeCountryCount }} {{ activeCountryCount === 1 ? 'país ativo' : 'países ativos' }}
        </span>
      </div>

      <div class="header-controls">
        <div class="view-toggle">
          <button class="toggle-btn" :class="{ active: viewMode === '3d' }" title="Visualização Globo 3D Interativo" @click="viewMode = '3d'">🌐 3D</button>
          <button class="toggle-btn" :class="{ active: viewMode === '2d' }" title="Visualização Mapa 2D" @click="viewMode = '2d'">🗺️ 2D</button>
        </div>

        <div class="filter-buttons">
          <button class="filter-btn" :class="{ active: eventType === 'PageView' }" @click="eventType = 'PageView'">Visitas</button>
          <button class="filter-btn" :class="{ active: eventType === 'Purchase' }" @click="eventType = 'Purchase'">Compras</button>
        </div>
      </div>
    </div>

    <div ref="containerRef" class="globe-container">
      <div v-show="viewMode === '3d'" class="globe-3d-wrapper">
        <canvas
          ref="canvasRef"
          class="globe-canvas"
          @pointerdown="onPointerDown"
          @pointermove="onPointerMove"
          @pointerup="onPointerUp"
          @pointercancel="onPointerUp"
          @wheel.prevent="onWheel"
        />

        <div class="globe-overlay-controls">
          <div class="focus-pills">
            <button
              v-for="focus in regionFocusList"
              :key="focus.id"
              class="focus-pill"
              :class="{ active: activeFocus === focus.id }"
              @click="focusRegion(focus)"
            >
              {{ focus.label }}
            </button>
          </div>

          <div class="globe-tools">
            <button class="tool-btn" :title="isAutoRotating ? 'Pausar rotação' : 'Girar automaticamente'" @click="isAutoRotating = !isAutoRotating">
              {{ isAutoRotating ? '⏸' : '▶' }}
            </button>
            <button class="tool-btn" title="Aumentar zoom" @click="zoomIn">+</button>
            <button class="tool-btn" title="Diminuir zoom" @click="zoomOut">−</button>
            <button class="tool-btn" title="Resetar posição" @click="resetOrientation">⟲</button>
          </div>
        </div>

        <div
          v-if="hoveredMarker"
          class="globe-tooltip"
          :style="{ left: hoveredMarker.x + 'px', top: hoveredMarker.y + 'px' }"
        >
          <div class="tooltip-header">
            <span class="tooltip-flag">{{ hoveredMarker.flag }}</span>
            <span class="tooltip-title">{{ hoveredMarker.name }}</span>
          </div>
          <div class="tooltip-val">
            <strong>{{ hoveredMarker.count.toLocaleString('pt-BR') }}</strong>
            {{ eventType === 'Purchase' ? 'compras' : 'visitas' }}
          </div>
        </div>
      </div>

      <div v-if="viewMode === '2d'" class="globe-2d-wrapper">
        <ClientOnly fallback-tag="div" fallback="Carregando mapa...">
          <v-chart class="echarts-map" :option="chartOption" autoresize />
        </ClientOnly>
      </div>
    </div>

    <GeoRankingList :countries="topCountries" @focus-country="focusCountry" />
  </div>
</template>

<script setup lang="ts">
import { use } from 'echarts/core'
import { MapChart, ScatterChart } from 'echarts/charts'
import { GeoComponent, TooltipComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import VChart from 'vue-echarts'
import GeoRankingList from '~/components/dashboard/GeoRankingList.vue'
import type { GeoStat } from '~/utils/geoCoordinates'

use([MapChart, ScatterChart, GeoComponent, TooltipComponent, CanvasRenderer])

interface Props {
  stats: GeoStat[]
  subtitle?: string
  from?: string
  to?: string
}

const props = withDefaults(defineProps<Props>(), {
  stats: () => [],
  subtitle: '',
  from: '',
  to: '',
})

const {
  viewMode, eventType, isAutoRotating, activeFocus,
  containerRef, canvasRef, chartOption, hoveredMarker,
  topCountries, activeCountryCount, regionFocusList,
  onPointerDown, onPointerMove, onPointerUp, onWheel,
  zoomIn, zoomOut, resetOrientation, focusRegion, focusCountry,
} = useGeoGlobeRenderer(() => props.stats)
</script>

<style scoped>
.geo-card {
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 1.125rem;
  box-shadow: var(--shadow-card);
  height: 100%;
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  position: relative;
  overflow: hidden;
}

.card-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 0.75rem; gap: 0.75rem; flex-wrap: wrap; z-index: 10; }
.title-group { display: flex; align-items: center; gap: 0.625rem; }
.card-title { font-family: var(--font-sans); font-size: var(--fs-lg); font-weight: var(--fw-semibold); color: var(--text-strong); letter-spacing: -0.01em; margin: 0; }
.card-badge { font-size: var(--fs-xs); font-weight: var(--fw-semibold); background: var(--brand-soft); color: var(--brand); padding: 0.125rem 0.5rem; border-radius: 9999px; border: 1px solid var(--brand-border); }
.header-controls { display: flex; align-items: center; gap: 0.5rem; }

.view-toggle, .filter-buttons { display: flex; background: var(--surface-1); border: 1px solid var(--border); border-radius: 6px; padding: 2px; gap: 2px; }
.toggle-btn, .filter-btn { padding: 0.2rem 0.55rem; font-size: var(--fs-sm); font-weight: var(--fw-medium); border: none; background: transparent; color: var(--text-2); border-radius: 4px; cursor: pointer; transition: all 0.15s ease; }
.toggle-btn.active { background: var(--surface-3); color: var(--text-strong); font-weight: var(--fw-semibold); }
.filter-btn:hover { color: var(--text-1); }
.filter-btn.active { background: var(--brand-soft); color: var(--brand); font-weight: var(--fw-semibold); }

.globe-container {
  flex: 1;
  min-height: 380px;
  height: 380px;
  width: 100%;
  border-radius: 8px;
  position: relative;
  background: radial-gradient(circle at 50% 50%, var(--globe-bg-start) 0%, var(--globe-bg-end) 100%);
  border: 1px solid var(--border-soft);
  overflow: hidden;
}

.globe-3d-wrapper, .globe-2d-wrapper { position: absolute; inset: 0; width: 100%; height: 100%; }
.globe-3d-wrapper { cursor: grab; }
.globe-3d-wrapper:active { cursor: grabbing; }
.globe-canvas { position: absolute; inset: 0; width: 100%; height: 100%; display: block; }
.echarts-map { width: 100%; height: 100%; }

.globe-overlay-controls {
  position: absolute;
  top: 0.75rem;
  left: 0.75rem;
  right: 0.75rem;
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  pointer-events: none;
  z-index: 5;
}

.focus-pills { display: flex; gap: 0.35rem; flex-wrap: wrap; pointer-events: auto; }
.focus-pill {
  background: var(--globe-bg-start);
  backdrop-filter: blur(8px);
  border: 1px solid var(--globe-border);
  color: var(--globe-text);
  font-size: var(--fs-xs);
  font-weight: var(--fw-medium);
  padding: 0.25rem 0.5rem;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s ease;
}
.focus-pill:hover { background: var(--globe-hover); border-color: var(--border-hover); }
.focus-pill.active { background: var(--brand-soft); border-color: var(--brand-border); color: var(--brand); font-weight: var(--fw-semibold); }

.globe-tools { display: flex; flex-direction: column; gap: 0.35rem; pointer-events: auto; }
.tool-btn {
  width: 28px; height: 28px; border-radius: 6px;
  background: var(--globe-bg-start); backdrop-filter: blur(8px);
  border: 1px solid var(--globe-border); color: var(--globe-text);
  font-size: var(--fs-base); font-weight: var(--fw-bold); display: flex;
  align-items: center; justify-content: center; cursor: pointer; transition: all 0.15s ease;
}
.tool-btn:hover { background: var(--globe-hover); border-color: var(--border-hover); color: var(--text-on-solid); }

.globe-tooltip {
  position: absolute; transform: translate(-50%, -100%);
  background: var(--globe-bg-start); backdrop-filter: blur(8px);
  border: 1px solid var(--globe-border); border-radius: 6px;
  padding: 0.4rem 0.65rem; pointer-events: none;
  box-shadow: var(--shadow-elevated); z-index: 20; white-space: nowrap;
}
.tooltip-header { display: flex; align-items: center; gap: 0.35rem; font-size: var(--fs-sm); font-weight: var(--fw-semibold); color: var(--text-on-solid); }
.tooltip-flag { font-size: var(--fs-md); }
.tooltip-val { font-size: var(--fs-xs); color: var(--globe-muted); margin-top: 2px; }
.tooltip-val strong { color: var(--brand); }
</style>
