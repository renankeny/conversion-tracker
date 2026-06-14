<template>
  <div class="geo-card">
    <div class="card-header">
      <h3 class="card-title">Distribuição Geográfica</h3>
      <div class="filter-buttons">
        <button
          class="filter-btn"
          :class="{ active: eventType === 'PageView' }"
          @click="eventType = 'PageView'"
        >
          Visitas
        </button>
        <button
          class="filter-btn"
          :class="{ active: eventType === 'Purchase' }"
          @click="eventType = 'Purchase'"
        >
          Compras
        </button>
      </div>
    </div>
    <canvas 
      ref="canvas" 
      class="map-canvas" 
      @mousedown="onMouseDown"
      @mousemove="onMouseMove"
      @mouseup="onMouseUp"
      @mouseleave="onMouseLeave"
      @touchstart="onTouchStart"
      @touchmove="onTouchMove"
      @touchend="onTouchEnd"
    />
    <Transition name="fade">
      <div v-if="hoverInfo" class="tooltip" :style="{ left: hoverX + 'px', top: hoverY + 'px' }">
        <div class="tooltip-title">{{ hoverInfo.name }}</div>
        <div class="tooltip-count">{{ hoverInfo.count }} {{ eventType === 'PageView' ? 'visitas' : 'compras' }}</div>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, computed, watch } from 'vue'

interface GeoStat {
  country?: string
  state?: string
  city?: string
  count?: number
  event_name?: string
  [key: string]: any
}

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

const canvas = ref<HTMLCanvasElement | null>(null)
const eventType = ref<'PageView' | 'Purchase'>('PageView')
const hoverInfo = ref<{ name: string; count: number } | null>(null)
const hoverX = ref(0)
const hoverY = ref(0)
const loading = ref(false)
const eventStats = ref<GeoStat[]>([])

let animId = 0
let polygons: [number, number][][] = []
let panX = 0
let panY = 0
let zoom = 5
let isDragging = false
let dragStartX = 0
let dragStartY = 0
let lastX = 0
let lastY = 0
let markers: { lat: number; lon: number; name: string; count: number; event: string; projX: number; projY: number }[] = []
let initialPanSet = false

const geoCoords: Record<string, { lat: number; lon: number }> = {
  'SP': { lat: -23.55, lon: -46.63 },
  'RJ': { lat: -22.91, lon: -43.17 },
  'MG': { lat: -19.92, lon: -43.93 },
  'BA': { lat: -12.98, lon: -38.50 },
  'RS': { lat: -30.03, lon: -51.22 },
  'PR': { lat: -25.43, lon: -49.27 },
  'PE': { lat: -8.06, lon: -34.88 },
  'CE': { lat: -3.73, lon: -38.53 },
  'PA': { lat: -1.46, lon: -48.49 },
  'SC': { lat: -27.60, lon: -48.55 },
  'MA': { lat: -2.53, lon: -44.31 },
  'GO': { lat: -16.69, lon: -49.26 },
  'AM': { lat: -3.12, lon: -60.02 },
  'BR': { lat: -14.24, lon: -51.93 },
  'US': { lat: 37.09, lon: -95.71 },
  'PT': { lat: 39.40, lon: -8.22 },
  'AR': { lat: -38.42, lon: -63.62 },
  'MX': { lat: 23.63, lon: -102.55 },
  'GB': { lat: 55.38, lon: -3.44 },
  'FR': { lat: 46.23, lon: 2.21 },
  'DE': { lat: 51.17, lon: 10.45 },
  'IT': { lat: 41.87, lon: 12.57 },
  'JP': { lat: 36.20, lon: 138.25 },
}

const filteredStats = computed(() => {
  if (!eventStats.value || eventStats.value.length === 0) {
    return []
  }
  
  // Filtra pelo tipo de evento selecionado
  return eventStats.value.filter(s => s.event_name === eventType.value)
})

// Buscar dados quando o tipo de evento mudar
const fetchEventData = async () => {
  if (!props.from || !props.to) return
  
  loading.value = true
  try {
    const dashboard = useDashboard()
    const data = await dashboard.geoStatsByEvent({
      from: props.from,
      to: props.to,
      event: eventType.value,
    })
    eventStats.value = data || []
    updateMarkers()
  } catch (error) {
    console.error('Erro ao buscar dados geográficos:', error)
  } finally {
    loading.value = false
  }
}

watch(eventType, fetchEventData)
watch(() => [props.from, props.to], fetchEventData)
watch(filteredStats, updateMarkers, { deep: true })

function updateMarkers() {
  if (!filteredStats.value || filteredStats.value.length === 0) {
    markers = []
    return
  }
  
  markers = filteredStats.value.map((s) => {
    let coord = geoCoords['BR'] || { lat: -14.24, lon: -51.93 }
    
    if (s.state) {
      const stateCoord = geoCoords[s.state.toUpperCase()]
      if (stateCoord) coord = stateCoord
    } else if (s.country) {
      const countryCoord = geoCoords[s.country.toUpperCase()]
      if (countryCoord) coord = countryCoord
    }

    let hash = 0
    const seed = (s.city || '') + (s.state || '')
    for (let i = 0; i < seed.length; i++) {
      hash = seed.charCodeAt(i) + ((hash << 5) - hash)
    }
    const offsetLat = ((hash % 100) / 100) * 1.2 - 0.6
    const offsetLon = (((hash >> 8) % 100) / 100) * 1.2 - 0.6

    return {
      lat: coord.lat + offsetLat,
      lon: coord.lon + offsetLon,
      name: s.city || s.state || s.country || 'Desconhecido',
      count: s.count || 1,
      event: s.event_name || eventType.value,
      projX: 0,
      projY: 0,
    }
  })
}

const projectPoint = (lat: number, lon: number, w: number, h: number) => {
  // Projeção Mercator 2D (mapa plano)
  const x = (lon + 180) / 360
  const y = (1 - Math.log(Math.tan(Math.PI * (lat + 90) / 360)) / Math.PI) / 2
  return {
    x: x * w * zoom + panX,
    y: y * h * zoom + panY,
    visible: true
  }
}

const render = () => {
  const c = canvas.value
  if (!c) {
    animId = requestAnimationFrame(render)
    return
  }

  const ctx = c.getContext('2d')
  if (!ctx) return

  const dpr = window.devicePixelRatio || 1
  const w = c.clientWidth
  const h = c.clientHeight

  if (c.width !== w * dpr || c.height !== h * dpr) {
    c.width = w * dpr
    c.height = h * dpr
  }

  ctx.resetTransform()
  ctx.scale(dpr, dpr)

  // Definir pan inicial para centralizar no Brasil
  if (!initialPanSet && w > 0 && h > 0) {
    const brazilLat = -14.24
    const brazilLon = -51.93
    const brazilProj = projectPoint(brazilLat, brazilLon, w, h)
    panX = w / 2 - brazilProj.x + panX
    panY = h / 2 - brazilProj.y + panY
    initialPanSet = true
  }

  const isLight = document.documentElement.getAttribute('data-theme') === 'light'
  
  // Fundo do mar
  ctx.fillStyle = isLight ? '#dbeafe' : '#0a1628'
  ctx.fillRect(0, 0, w, h)
  
  // Grid no mar (efeito de água)
  ctx.strokeStyle = isLight ? 'rgba(59, 130, 246, 0.08)' : 'rgba(59, 130, 246, 0.04)'
  ctx.lineWidth = 0.5
  const gridSize = 30
  
  // Linhas verticais
  for (let x = (panX % gridSize) - gridSize; x < w; x += gridSize) {
    ctx.beginPath()
    ctx.moveTo(x, 0)
    ctx.lineTo(x, h)
    ctx.stroke()
  }
  
  // Linhas horizontais
  for (let y = (panY % gridSize) - gridSize; y < h; y += gridSize) {
    ctx.beginPath()
    ctx.moveTo(0, y)
    ctx.lineTo(w, y)
    ctx.stroke()
  }

  // Cores alternadas para países
  const land1 = isLight ? '#e2e8f0' : '#1a2332'
  const land2 = isLight ? '#cbd5e1' : '#0f1419'
  const countryBorder = isLight ? '#94a3b8' : '#374151'
  const stateBorder = isLight ? '#64748b' : '#4b5563'
  
  // Primeira passar: preencher polígonos com cores alternadas
  polygons.forEach((poly, idx) => {
    if (!poly.length) return
    
    // Detectar se é estado ou país pelo tamanho
    const bounds = { minLon: 180, maxLon: -180, minLat: 90, maxLat: -90 }
    for (const [lon, lat] of poly) {
      bounds.minLon = Math.min(bounds.minLon, lon)
      bounds.maxLon = Math.max(bounds.maxLon, lon)
      bounds.minLat = Math.min(bounds.minLat, lat)
      bounds.maxLat = Math.max(bounds.maxLat, lat)
    }
    const width = bounds.maxLon - bounds.minLon
    const height = bounds.maxLat - bounds.minLat
    const isState = width < 20 && height < 20
    
    // Cor alternada baseada no índice
    ctx.fillStyle = idx % 2 === 0 ? land1 : land2
    
    ctx.beginPath()
    let first = true
    for (const [lon, lat] of poly) {
      const p = projectPoint(lat, lon, w, h)
      if (first) {
        ctx.moveTo(p.x, p.y)
        first = false
      } else {
        ctx.lineTo(p.x, p.y)
      }
    }
    ctx.fill()
  })
  
  // Segunda passar: desenhar bordas
  for (const poly of polygons) {
    if (!poly.length) continue
    
    // Detectar tamanho
    const bounds = { minLon: 180, maxLon: -180, minLat: 90, maxLat: -90 }
    for (const [lon, lat] of poly) {
      bounds.minLon = Math.min(bounds.minLon, lon)
      bounds.maxLon = Math.max(bounds.maxLon, lon)
      bounds.minLat = Math.min(bounds.minLat, lat)
      bounds.maxLat = Math.max(bounds.maxLat, lat)
    }
    const width = bounds.maxLon - bounds.minLon
    const height = bounds.maxLat - bounds.minLat
    const isState = width < 20 && height < 20
    
    ctx.strokeStyle = isState ? stateBorder : countryBorder
    ctx.lineWidth = isState ? 1 : 1.2
    
    ctx.beginPath()
    let first = true
    for (const [lon, lat] of poly) {
      const p = projectPoint(lat, lon, w, h)
      if (first) {
        ctx.moveTo(p.x, p.y)
        first = false
      } else {
        ctx.lineTo(p.x, p.y)
      }
    }
    ctx.stroke()
  }

  // Desenhar marcadores
  const time = Date.now() * 0.003
  let closest = null
  let minDist = 20

  markers.forEach((m) => {
    const proj = projectPoint(m.lat, m.lon, w, h)
    m.projX = proj.x
    m.projY = proj.y

    // Verificar se está dentro da tela
    if (proj.x > -50 && proj.x < w + 50 && proj.y > -50 && proj.y < h + 50) {
      const pulse = Math.sin(time + m.lat + m.lon) * 0.4 + 0.6
      const size = Math.max(5, Math.min(20, 3 + Math.log2(Math.max(1, m.count)) * 2.5))

      // Cor baseada no tipo de evento
      const isPageView = m.event === 'PageView'
      const color = isPageView ? '59, 130, 246' : '34, 197, 94' // Azul para visitas, verde para compras

      // Halo
      ctx.beginPath()
      ctx.arc(proj.x, proj.y, size * (1.5 + pulse * 0.6), 0, Math.PI * 2)
      ctx.fillStyle = `rgba(${color}, ${0.1 * (1 - pulse * 0.4)})`
      ctx.fill()

      // Anel
      ctx.beginPath()
      ctx.arc(proj.x, proj.y, size * 1.15, 0, Math.PI * 2)
      ctx.strokeStyle = `rgba(${color}, 0.7)`
      ctx.lineWidth = 2.5
      ctx.stroke()

      // Centro
      ctx.beginPath()
      ctx.arc(proj.x, proj.y, size * 0.6, 0, Math.PI * 2)
      ctx.fillStyle = `rgba(${color}, 1)`
      ctx.fill()

      // Detectar hover
      const dx = lastX - proj.x
      const dy = lastY - proj.y
      const dist = Math.sqrt(dx * dx + dy * dy)
      if (dist < minDist && !isDragging) {
        minDist = dist
        closest = m
        c.style.cursor = 'pointer'
      }
    }
  })

  if (closest && !isDragging) {
    hoverInfo.value = {
      name: closest.name,
      count: closest.count,
    }
  } else if (!isDragging) {
    c.style.cursor = 'grab'
  }

  animId = requestAnimationFrame(render)
}

const onMouseDown = (e: MouseEvent) => {
  isDragging = true
  dragStartX = e.clientX
  dragStartY = e.clientY
}

const onMouseMove = (e: MouseEvent) => {
  const c = canvas.value
  if (!c) return
  
  const rect = c.getBoundingClientRect()
  lastX = e.clientX - rect.left
  lastY = e.clientY - rect.top

  if (isDragging) {
    const dx = e.clientX - dragStartX
    const dy = e.clientY - dragStartY
    panX += dx
    panY += dy
    dragStartX = e.clientX
    dragStartY = e.clientY
    hoverInfo.value = null
    c.style.cursor = 'grabbing'
  } else {
    // Atualizar posição do tooltip usando coordenadas da tela
    hoverX.value = e.clientX
    hoverY.value = e.clientY
  }
}

const onMouseUp = () => {
  isDragging = false
}

const onMouseLeave = () => {
  isDragging = false
  hoverInfo.value = null
}

const onTouchStart = (e: TouchEvent) => {
  if (e.touches.length === 1) {
    isDragging = true
    dragStartX = e.touches[0].clientX
    dragStartY = e.touches[0].clientY
    e.preventDefault()
  }
}

const onTouchMove = (e: TouchEvent) => {
  const c = canvas.value
  if (!c) return
  
  if (e.touches.length === 1) {
    const touch = e.touches[0]
    const rect = c.getBoundingClientRect()
    lastX = touch.clientX - rect.left
    lastY = touch.clientY - rect.top
    
    if (isDragging) {
      const dx = touch.clientX - dragStartX
      const dy = touch.clientY - dragStartY
      panX += dx
      panY += dy
      dragStartX = touch.clientX
      dragStartY = touch.clientY
      hoverInfo.value = null
      e.preventDefault()
    }
  }
}

const onTouchEnd = () => {
  isDragging = false
}

const onWheel = (e: WheelEvent) => {
  e.preventDefault()
  const c = canvas.value
  if (!c) return
  
  const rect = c.getBoundingClientRect()
  const mouseX = e.clientX - rect.left
  const mouseY = e.clientY - rect.top
  
  const oldZoom = zoom
  const delta = e.deltaY > 0 ? 0.9 : 1.1
  zoom = Math.max(0.5, Math.min(8, zoom * delta))
  
  // Ajustar pan para zoom centrado no mouse
  const zoomChange = zoom / oldZoom
  panX = mouseX - (mouseX - panX) * zoomChange
  panY = mouseY - (mouseY - panY) * zoomChange
}

const load = async () => {
  try {
    // Carregar países do mundo
    const resWorld = await fetch('/world-land.json')
    const geoWorld = await resWorld.json()
    for (const feat of geoWorld.features) {
      if (!feat.geometry) continue
      if (feat.geometry.type === 'Polygon') {
        polygons.push(feat.geometry.coordinates[0])
      } else if (feat.geometry.type === 'MultiPolygon') {
        for (const coords of feat.geometry.coordinates) {
          polygons.push(coords[0])
        }
      }
    }
    
    // Tentar carregar GeoJSON do Brasil com estados
    try {
      const resBR = await fetch('https://raw.githubusercontent.com/codeforamerica/click_that_hood/master/public/data/brazil-states.geojson')
      const geoBR = await resBR.json()
      
      for (const feat of geoBR.features) {
        if (!feat.geometry) continue
        if (feat.geometry.type === 'Polygon') {
          polygons.push(feat.geometry.coordinates[0])
        } else if (feat.geometry.type === 'MultiPolygon') {
          for (const coords of feat.geometry.coordinates) {
            polygons.push(coords[0])
          }
        }
      }
    } catch (e) {
      // Estados não disponíveis, usar apenas países
    }
    
    // Buscar dados iniciais
    await fetchEventData()
    render()
  } catch (e) {
    console.error('Erro ao carregar mapa:', e)
  }
}

onMounted(() => {
  load()
  const c = canvas.value
  if (c) {
    c.addEventListener('wheel', onWheel, { passive: false })
  }
})

onBeforeUnmount(() => {
  cancelAnimationFrame(animId)
  const c = canvas.value
  if (c) {
    c.removeEventListener('wheel', onWheel)
  }
})
</script>

<style scoped>
.geo-card {
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 1.25rem;
  box-shadow: var(--shadow-card);
  height: 100%;
  display: flex;
  flex-direction: column;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 0.75rem;
}

.card-title {
  font-family: 'Instrument Sans', sans-serif;
  font-size: 1.125rem;
  font-weight: 600;
  color: var(--text-strong);
}

.filter-buttons {
  display: flex;
  gap: 0.5rem;
}

.filter-btn {
  padding: 0.5rem 1rem;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: transparent;
  color: var(--text-2);
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}

.filter-btn:hover {
  background: var(--surface-soft);
  color: var(--text-1);
}

.filter-btn.active {
  background: var(--brand-soft);
  color: var(--brand);
  border-color: var(--brand);
}

.map-canvas {
  flex: 1;
  min-height: 380px;
  width: 100%;
  display: block;
  border-radius: 8px;
  background: var(--surface-1);
  cursor: grab;
}

.map-canvas:active {
  cursor: grabbing;
}

.tooltip {
  position: fixed;
  background: rgba(17, 20, 27, 0.92);
  backdrop-filter: blur(10px);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 8px;
  padding: 0.625rem 0.875rem;
  box-shadow: 0 10px 25px rgba(0, 0, 0, 0.3);
  pointer-events: none;
  z-index: 100;
  white-space: nowrap;
  transform: translate(-50%, calc(-100% - 12px));
}

[data-theme='light'] .tooltip {
  background: rgba(255, 255, 255, 0.92);
  border: 1px solid rgba(15, 23, 42, 0.08);
}

.tooltip-title {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--text-strong);
  margin-bottom: 0.25rem;
}

.tooltip-count {
  font-size: 0.75rem;
  color: var(--text-2);
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
