import { ref, computed, watch, onMounted, onUnmounted, shallowRef } from 'vue'
import * as echarts from 'echarts/core'
import { useDashboardStore } from '~/stores/useDashboardStore'
import { countryInfo, regionFocusList, resolveCoordinates, type GeoStat, type RegionFocus } from '~/utils/geoCoordinates'
import { loadWorldGeoJson, projectPoint, draw3DGlobeFrame, build2DChartOption, type ProcessedGeoMarker } from '~/utils/geoGlobeScene'

let worldMapRegistered = false

export const useGeoGlobeRenderer = (getStats: () => GeoStat[]) => {
  const dashboardStore = useDashboardStore()
  const { isDark } = useTheme()

  const viewMode = computed<'3d' | '2d'>({
    get: () => dashboardStore.globeViewMode,
    set: (v) => { dashboardStore.globeViewMode = v },
  })
  const eventType = computed<'PageView' | 'Purchase'>({
    get: () => dashboardStore.globeEventType,
    set: (v) => { dashboardStore.globeEventType = v },
  })
  const isAutoRotating = ref(true)
  const activeFocus = ref<string>('all')

  const containerRef = ref<HTMLDivElement | null>(null)
  const canvasRef = ref<HTMLCanvasElement | null>(null)
  const chartOption = shallowRef<Record<string, unknown>>({})
  const hoveredMarker = ref<{ name: string; flag: string; count: number; x: number; y: number } | null>(null)

  let mapLoaded = false
  let worldGeoJson: unknown = null
  let animationFrameId: number | null = null
  let resizeObserver: ResizeObserver | null = null
  let rotY = -0.9, rotX = -0.2
  let targetRotY: number | null = null, targetRotX: number | null = null
  let globeRadius = 140, isDragging = false
  let lastPointerX = 0, lastPointerY = 0, dragVelocityX = 0, dragVelocityY = 0

  const activeStats = computed(() =>
    (getStats() || []).filter((s) => (s.event_name || 'PageView') === eventType.value),
  )

  const processedMarkers = computed<ProcessedGeoMarker[]>(() => {
    const map = new Map<string, ProcessedGeoMarker>()
    for (const s of activeStats.value) {
      const count = s.count ?? 0
      if (count <= 0 || (!s.country && !s.state && !s.city)) continue
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
      const code = (s.country || '').trim().toUpperCase()
      const count = s.count ?? 0
      if (!code || count <= 0) continue
      const meta = countryInfo[code] || { name: code, flag: '🌐' }
      if (!counts[code]) counts[code] = { count: 0, name: meta.name, flag: meta.flag }
      counts[code].count += count
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

  function stopLoop() {
    if (animationFrameId !== null) { cancelAnimationFrame(animationFrameId); animationFrameId = null }
  }

  function animate() {
    if (viewMode.value !== '3d' || (typeof document !== 'undefined' && document.hidden)) {
      animationFrameId = null
      return
    }
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
    animationFrameId = requestAnimationFrame(animate)
  }

  function startLoop() {
    if (animationFrameId === null && viewMode.value === '3d' && (typeof document === 'undefined' || !document.hidden)) {
      animationFrameId = requestAnimationFrame(animate)
    }
  }

  function onVisibilityChange() {
    if (typeof document !== 'undefined' && document.hidden) stopLoop()
    else startLoop()
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
      const dx = e.clientX - lastPointerX, dy = e.clientY - lastPointerY, sensitivity = 0.006
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
    let found: { name: string; flag: string; count: number; x: number; y: number } | null = null
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

  watch([getStats, eventType, isDark], () => { if (viewMode.value === '2d') update2DChart() }, { deep: true })
  watch(viewMode, (v) => {
    if (v === '2d') { stopLoop(); setTimeout(update2DChart, 50) }
    else { setTimeout(() => { resizeCanvas(); startLoop() }, 50) }
  })

  watch(topCountries, (countries) => {
    const first = countries[0]
    if (first && first.code === 'JP' && activeFocus.value === 'all') {
      const jpRegion = regionFocusList.find((r) => r.id === 'jp') || regionFocusList[0]
      if (jpRegion) focusRegion(jpRegion)
    }
  }, { immediate: true })

  onMounted(async () => {
    try {
      worldGeoJson = await loadWorldGeoJson()
      if (!worldMapRegistered) {
        echarts.registerMap('world', worldGeoJson as Parameters<typeof echarts.registerMap>[1])
        worldMapRegistered = true
      }
      mapLoaded = true; update2DChart()
    } catch (e) { console.error('Falha ao carregar mapa mundial', e) }
    resizeCanvas()
    if (containerRef.value) { resizeObserver = new ResizeObserver(resizeCanvas); resizeObserver.observe(containerRef.value) }
    document.addEventListener('visibilitychange', onVisibilityChange)
    startLoop()
  })

  onUnmounted(() => {
    stopLoop()
    resizeObserver?.disconnect()
    document.removeEventListener('visibilitychange', onVisibilityChange)
  })

  return {
    viewMode, eventType, isAutoRotating, activeFocus, containerRef, canvasRef, chartOption,
    hoveredMarker, topCountries, activeCountryCount, regionFocusList,
    onPointerDown, onPointerMove, onPointerUp, onWheel, zoomIn, zoomOut, resetOrientation, focusRegion, focusCountry,
  }
}
