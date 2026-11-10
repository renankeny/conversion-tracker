import { ref, computed, watch, onMounted, onUnmounted, shallowRef } from 'vue'
import * as echarts from 'echarts/core'
import { countryInfo, regionFocusList, resolveCoordinates, type GeoStat, type RegionFocus } from '~/utils/geoCoordinates'
import { projectPoint, draw3DGlobeFrame, build2DChartOption, type ProcessedGeoMarker } from '~/utils/geoGlobeScene'

export const useGeoGlobeRenderer = (getStats: () => GeoStat[]) => {
  const viewMode = ref<'3d' | '2d'>('3d')
  const eventType = ref<'PageView' | 'Purchase'>('PageView')
  const isAutoRotating = ref(true)
  const activeFocus = ref<string>('all')

  const containerRef = ref<HTMLDivElement | null>(null)
  const canvasRef = ref<HTMLCanvasElement | null>(null)
  const chartOption = shallowRef<any>({})
  const hoveredMarker = ref<{ name: string; flag: string; count: number; x: number; y: number } | null>(null)

  let mapLoaded = false
  let worldGeoJson: any = null
  let animationFrameId: number | null = null
  let resizeObserver: ResizeObserver | null = null
  let rotY = -0.9
  let rotX = -0.2
  let targetRotY: number | null = null
  let targetRotX: number | null = null
  let globeRadius = 140
  let isDragging = false
  let lastPointerX = 0
  let lastPointerY = 0
  let dragVelocityX = 0
  let dragVelocityY = 0

  const activeStats = computed(() =>
    (getStats() || []).filter(s => (s.event_name || 'PageView') === eventType.value)
  )

  const processedMarkers = computed<ProcessedGeoMarker[]>(() => {
    const map = new Map<string, ProcessedGeoMarker>()
    for (const s of activeStats.value) {
      const count = s.count || 1
      const res = resolveCoordinates(s)
      const key = `${res.lat.toFixed(2)}_${res.lon.toFixed(2)}`
      const existing = map.get(key)
      if (existing) existing.count += count
      else map.set(key, { lat: res.lat, lon: res.lon, name: res.resolvedName, flag: res.flag, count })
    }
    return Array.from(map.values())
  })

  const topCountries = computed(() => {
    const counts: Record<string, { count: number; name: string; flag: string }> = {}
    for (const s of activeStats.value) {
      const code = (s.country || 'BR').toUpperCase()
      const meta = countryInfo[code] || { name: code, flag: '🌐' }
      if (!counts[code]) counts[code] = { count: 0, name: meta.name, flag: meta.flag }
      counts[code].count += (s.count || 1)
    }
    return Object.entries(counts).map(([code, data]) => ({ code, ...data })).sort((a, b) => b.count - a.count).slice(0, 6)
  })

  const activeCountryCount = computed(() => topCountries.value.length)

  function render3DGlobe() {
    if (!canvasRef.value) return
    globeRadius = draw3DGlobeFrame({
      canvas: canvasRef.value, globeRadius, rotY, rotX, worldGeoJson,
      markers: processedMarkers.value, isPurchase: eventType.value === 'Purchase',
    })
  }

  function animate() {
    if (viewMode.value === '3d') {
      if (targetRotY !== null && targetRotX !== null) {
        rotY += (targetRotY - rotY) * 0.08
        rotX += (targetRotX - rotX) * 0.08
        if (Math.abs(targetRotY - rotY) < 0.005 && Math.abs(targetRotX - rotX) < 0.005) {
          rotY = targetRotY; rotX = targetRotX; targetRotY = null; targetRotX = null
        }
      } else if (isAutoRotating.value && !isDragging) {
        rotY += 0.0035
      }
      if (!isDragging && (Math.abs(dragVelocityX) > 0.0001 || Math.abs(dragVelocityY) > 0.0001)) {
        rotY += dragVelocityX
        rotX = Math.max(-1.1, Math.min(1.1, rotX + dragVelocityY))
        dragVelocityX *= 0.92; dragVelocityY *= 0.92
      }
      render3DGlobe()
    }
    animationFrameId = requestAnimationFrame(animate)
  }

  function onPointerDown(e: PointerEvent) {
    isDragging = true; targetRotY = null; targetRotX = null
    lastPointerX = e.clientX; lastPointerY = e.clientY; dragVelocityX = 0; dragVelocityY = 0
    ;(e.target as HTMLElement)?.setPointerCapture(e.pointerId)
  }

  function onPointerMove(e: PointerEvent) {
    const canvas = canvasRef.value
    if (!canvas) return
    if (isDragging) {
      const dx = e.clientX - lastPointerX
      const dy = e.clientY - lastPointerY
      const sensitivity = 0.006
      rotY += dx * sensitivity
      rotX = Math.max(-1.1, Math.min(1.1, rotX - dy * sensitivity))
      dragVelocityX = dx * sensitivity; dragVelocityY = -dy * sensitivity
      lastPointerX = e.clientX; lastPointerY = e.clientY; hoveredMarker.value = null
      return
    }
    const rect = canvas.getBoundingClientRect()
    const mouseX = (e.clientX - rect.left) * (canvas.width / rect.width)
    const mouseY = (e.clientY - rect.top) * (canvas.height / rect.height)
    const cx = canvas.width / 2, cy = canvas.height / 2
    let found: any = null
    for (const m of processedMarkers.value) {
      const p = projectPoint(m.lat, m.lon, globeRadius, cx, cy, rotY, rotX)
      if (!p.visible || p.z < 0.05) continue
      if (Math.hypot(p.screenX - mouseX, p.screenY - mouseY) < 15) {
        found = { name: m.name, flag: m.flag, count: m.count, x: e.clientX - rect.left, y: e.clientY - rect.top - 12 }
        break
      }
    }
    hoveredMarker.value = found
  }

  function onPointerUp(e: PointerEvent) {
    isDragging = false
    try { ;(e.target as HTMLElement)?.releasePointerCapture(e.pointerId) } catch {}
  }

  function zoomIn() { globeRadius = Math.min(220, globeRadius + 15) }
  function zoomOut() { globeRadius = Math.max(90, globeRadius - 15) }
  function onWheel(e: WheelEvent) { e.deltaY < 0 ? zoomIn() : zoomOut() }

  function focusRegion(region: RegionFocus) {
    activeFocus.value = region.id
    targetRotY = ((-region.targetLon + 90) * Math.PI) / 180
    targetRotX = (-region.targetLat * Math.PI) / 180
  }

  function resetOrientation() { if (regionFocusList[0]) focusRegion(regionFocusList[0]) }

  function focusCountry(code: string) {
    const meta = countryInfo[code]
    if (!meta) return
    activeFocus.value = code.toLowerCase()
    targetRotY = ((-meta.lon + 90) * Math.PI) / 180
    targetRotX = (-meta.lat * Math.PI) / 180
  }

  function resizeCanvas() {
    const canvas = canvasRef.value, container = containerRef.value
    if (!canvas || !container) return
    const dpr = window.devicePixelRatio || 1
    const rect = container.getBoundingClientRect()
    const displayWidth = Math.floor(rect.width) || 450
    const displayHeight = Math.floor(rect.height) || 380
    canvas.width = Math.floor(displayWidth * dpr)
    canvas.height = Math.floor(displayHeight * dpr)
    canvas.style.width = displayWidth + 'px'; canvas.style.height = displayHeight + 'px'
    globeRadius = Math.max(110, Math.min(displayWidth, displayHeight) * 0.38)
  }

  function update2DChart() {
    if (mapLoaded) chartOption.value = build2DChartOption(processedMarkers.value, eventType.value)
  }

  watch(getStats, () => { if (viewMode.value === '2d') update2DChart() }, { deep: true })
  watch(eventType, () => { if (viewMode.value === '2d') update2DChart() })
  watch(viewMode, (v) => { setTimeout(v === '2d' ? update2DChart : resizeCanvas, 50) })

  watch(topCountries, (countries) => {
    const first = countries[0]
    if (first && first.code === 'JP' && activeFocus.value === 'all') {
      const jpRegion = regionFocusList.find(r => r.id === 'jp') || regionFocusList[0]
      if (jpRegion) focusRegion(jpRegion)
    }
  }, { immediate: true })

  onMounted(async () => {
    try {
      const resWorld = await fetch('/world-land.json')
      worldGeoJson = await resWorld.json()
      echarts.registerMap('world', worldGeoJson)
      mapLoaded = true
      update2DChart()
    } catch (e) { console.error('Falha ao carregar mapa mundial', e) }
    resizeCanvas()
    if (containerRef.value) {
      resizeObserver = new ResizeObserver(resizeCanvas)
      resizeObserver.observe(containerRef.value)
    }
    animate()
  })

  onUnmounted(() => {
    if (animationFrameId) cancelAnimationFrame(animationFrameId)
    if (resizeObserver) resizeObserver.disconnect()
  })

  return {
    viewMode, eventType, isAutoRotating, activeFocus, containerRef, canvasRef, chartOption,
    hoveredMarker, topCountries, activeCountryCount, regionFocusList,
    onPointerDown, onPointerMove, onPointerUp, onWheel, zoomIn, zoomOut, resetOrientation, focusRegion, focusCountry,
  }
}
