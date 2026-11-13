export interface ProcessedGeoMarker {
  lat: number
  lon: number
  name: string
  flag: string
  count: number
}

let cachedWorldGeoJson: unknown = null
let worldGeoJsonPromise: Promise<unknown> | null = null

export async function loadWorldGeoJson(): Promise<unknown> {
  if (cachedWorldGeoJson) return cachedWorldGeoJson
  if (!worldGeoJsonPromise) {
    worldGeoJsonPromise = fetch('/world-land.json')
      .then((res) => res.json())
      .then((data) => { cachedWorldGeoJson = data; return data })
      .catch((err) => { worldGeoJsonPromise = null; throw err })
  }
  return worldGeoJsonPromise
}

export const getCssVar = (val: string, fallback: string) => {
  if (typeof document === 'undefined') return fallback
  return getComputedStyle(document.documentElement).getPropertyValue(val).trim() || fallback
}

export function projectPoint(lat: number, lon: number, r: number, cx: number, cy: number, rY: number, rX: number) {
  const phi = (lat * Math.PI) / 180
  const lambda = (lon * Math.PI) / 180 + rY

  const x = Math.cos(phi) * Math.sin(lambda)
  const y = Math.sin(phi)
  const z = Math.cos(phi) * Math.cos(lambda)

  const y2 = y * Math.cos(rX) - z * Math.sin(rX)
  const z2 = y * Math.sin(rX) + z * Math.cos(rX)

  return {
    screenX: cx + r * x,
    screenY: cy - r * y2,
    visible: z2 > 0,
    z: z2,
  }
}

export function draw3DGlobeFrame(params: {
  canvas: HTMLCanvasElement
  globeRadius: number
  rotY: number
  rotX: number
  worldGeoJson: any
  markers: ProcessedGeoMarker[]
  isPurchase: boolean
}): number {
  const { canvas, rotY, rotX, worldGeoJson, markers, isPurchase } = params
  let { globeRadius } = params
  const ctx = canvas.getContext('2d')
  if (!ctx) return globeRadius

  const dpr = window.devicePixelRatio || 1
  const displayWidth = canvas.width / dpr || 450
  const displayHeight = canvas.height / dpr || 380

  if (globeRadius < 50) {
    globeRadius = Math.max(110, Math.min(displayWidth, displayHeight) * 0.38)
  }

  ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
  ctx.clearRect(0, 0, displayWidth, displayHeight)

  const cx = displayWidth / 2
  const cy = displayHeight / 2
  const brandColor = getCssVar('--brand', '#1C87C1')
  const brandLight = getCssVar('--brand-light', '#38BDF8')
  const brandSoft = getCssVar('--brand-soft', 'rgba(28, 135, 193, 0.15)')
  const brandBorder = getCssVar('--brand-border', 'rgba(28, 135, 193, 0.35)')

  // 1. Atmosfera e brilho ao redor do globo
  const atmosphere = ctx.createRadialGradient(cx, cy, globeRadius * 0.9, cx, cy, globeRadius * 1.25)
  atmosphere.addColorStop(0, brandSoft)
  atmosphere.addColorStop(0.5, getCssVar('--brand-soft-2', 'rgba(28, 135, 193, 0.05)'))
  atmosphere.addColorStop(1, 'rgba(0, 0, 0, 0)')

  ctx.beginPath()
  ctx.arc(cx, cy, globeRadius * 1.25, 0, Math.PI * 2)
  ctx.fillStyle = atmosphere
  ctx.fill()

  // 2. Base da Esfera (Oceano escuro 3D com iluminação)
  ctx.save()
  ctx.beginPath()
  ctx.arc(cx, cy, globeRadius, 0, Math.PI * 2)
  ctx.clip()

  const oceanGradient = ctx.createRadialGradient(
    cx - globeRadius * 0.35, cy - globeRadius * 0.35, globeRadius * 0.1,
    cx, cy, globeRadius
  )
  oceanGradient.addColorStop(0, '#1a2333')
  oceanGradient.addColorStop(0.7, '#0f172a')
  oceanGradient.addColorStop(1, '#080d1a')
  ctx.fillStyle = oceanGradient
  ctx.fill()

  // 3. Linhas de Grade Esférica
  ctx.strokeStyle = 'rgba(255, 255, 255, 0.04)'
  ctx.lineWidth = 0.75

  for (let lat = -60; lat <= 60; lat += 30) {
    ctx.beginPath()
    let first = true
    for (let lon = -180; lon <= 180; lon += 5) {
      const p = projectPoint(lat, lon, globeRadius, cx, cy, rotY, rotX)
      if (p.visible) {
        if (first) { ctx.moveTo(p.screenX, p.screenY); first = false }
        else ctx.lineTo(p.screenX, p.screenY)
      } else first = true
    }
    ctx.stroke()
  }

  for (let lon = -180; lon <= 180; lon += 45) {
    ctx.beginPath()
    let first = true
    for (let lat = -80; lat <= 80; lat += 5) {
      const p = projectPoint(lat, lon, globeRadius, cx, cy, rotY, rotX)
      if (p.visible) {
        if (first) { ctx.moveTo(p.screenX, p.screenY); first = false }
        else ctx.lineTo(p.screenX, p.screenY)
      } else first = true
    }
    ctx.stroke()
  }

  // 4. Desenhar Continentes a partir do GeoJSON
  if (worldGeoJson?.features) {
    ctx.strokeStyle = brandBorder
    ctx.lineWidth = 1.1
    for (const feature of worldGeoJson.features) {
      const geom = feature.geometry
      if (!geom || geom.type !== 'Polygon') continue
      for (const ring of geom.coordinates) {
        ctx.beginPath()
        let anyVisible = false
        let first = true
        for (let i = 0; i < ring.length; i++) {
          const pt = ring[i]
          const p = projectPoint(pt[1], pt[0], globeRadius, cx, cy, rotY, rotX)
          if (p.visible) {
            anyVisible = true
            if (first) { ctx.moveTo(p.screenX, p.screenY); first = false }
            else ctx.lineTo(p.screenX, p.screenY)
          } else first = true
        }
        if (anyVisible) ctx.stroke()
      }
    }
  }

  // 5. Brilho na borda interna do globo
  const innerShadow = ctx.createRadialGradient(cx, cy, globeRadius * 0.85, cx, cy, globeRadius)
  innerShadow.addColorStop(0, 'rgba(0, 0, 0, 0)')
  innerShadow.addColorStop(1, 'rgba(0, 0, 0, 0.65)')
  ctx.fillStyle = innerShadow
  ctx.fill()
  ctx.restore()

  // 6. Borda sutil da esfera
  ctx.beginPath()
  ctx.arc(cx, cy, globeRadius, 0, Math.PI * 2)
  ctx.strokeStyle = brandBorder
  ctx.lineWidth = 1.2
  ctx.stroke()

  // 7. Renderizar Marcadores de Eventos
  const dotColor = isPurchase ? brandColor : brandLight
  for (const m of markers) {
    const p = projectPoint(m.lat, m.lon, globeRadius, cx, cy, rotY, rotX)
    if (!p.visible || p.z < 0.05) continue
    const baseRadius = Math.max(3, Math.min(8, Math.log10(m.count + 1) * 2.8))
    ctx.beginPath()
    ctx.arc(p.screenX, p.screenY, baseRadius, 0, Math.PI * 2)
    ctx.fillStyle = dotColor
    ctx.fill()
    ctx.strokeStyle = '#ffffff'
    ctx.lineWidth = 1
    ctx.stroke()
  }

  return globeRadius
}

export function build2DChartOption(markers: ProcessedGeoMarker[], eventType: 'PageView' | 'Purchase') {
  const brandColor = getCssVar('--brand', '#10B981')
  const brandStrong = getCssVar('--brand-strong', '#059669')
  const mapAreaColor = getCssVar('--surface-3', '#242b35')
  const mapBorderColor = getCssVar('--border', '#334155')

  return {
    backgroundColor: 'transparent',
    tooltip: {
      trigger: 'item',
      backgroundColor: 'rgba(15,23,42,0.95)',
      borderColor: 'rgba(255,255,255,0.1)',
      textStyle: { color: '#fff' },
      formatter: (params: any) => {
        if (params.seriesType === 'scatter') {
          return `<div style="font-weight:var(--fw-bold);margin-bottom:2px;">${params.name}</div>
                  <div style="font-size:var(--fs-sm);color:var(--brand);">${params.value[2].toLocaleString('pt-BR')} ${eventType === 'Purchase' ? 'compras' : 'visitas'}</div>`
        }
        return params.name
      },
    },
    geo: {
      map: 'world',
      roam: true,
      zoom: 1.25,
      scaleLimit: { min: 1, max: 14 },
      itemStyle: { areaColor: mapAreaColor, borderColor: mapBorderColor, borderWidth: 0.6 },
      emphasis: { itemStyle: { areaColor: getCssVar('--brand-soft', 'rgba(28, 135, 193, 0.15)') }, label: { show: false } },
    },
    series: [{
      name: 'Distribuição',
      type: 'scatter',
      coordinateSystem: 'geo',
      data: markers.map(d => ({ name: `${d.flag} ${d.name}`, value: [d.lon, d.lat, d.count] })),
      symbolSize: (val: any) => Math.max(6, Math.min(22, Math.log10(val[2] + 1) * 7.5)),
      itemStyle: { color: eventType === 'PageView' ? brandColor : brandStrong, opacity: 0.85 },
    }],
  }
}
