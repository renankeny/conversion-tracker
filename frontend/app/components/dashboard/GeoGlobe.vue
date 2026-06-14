<template>
  <div class="geo-globe-card">
    <div class="card-header">
      <div>
        <h3 class="card-title">Distribuição Geográfica</h3>
        <p class="card-subtitle">{{ subtitle || 'Acessos e conversões por estado/país' }}</p>
      </div>
      <div v-if="loadingGeo" class="loading-badge">
        Carregando mapa...
      </div>
    </div>

    <div class="globe-container">
      <!-- Elemento Canvas Principal -->
      <canvas
        ref="canvasRef"
        class="globe-canvas"
        @mousedown="onMouseDown"
        @mousemove="onMouseMove"
        @mouseup="onMouseUp"
        @mouseleave="onMouseLeave"
        @touchstart="onTouchStart"
        @touchmove="onTouchMove"
        @touchend="onTouchEnd"
      />

      <!-- Tooltip Premium com Glassmorphism -->
      <Transition name="fade">
        <div
          v-if="hoveredMarker"
          class="globe-tooltip"
          :style="{ left: tooltipX + 'px', top: tooltipY + 'px' }"
        >
          <div class="tooltip-city">{{ hoveredMarker.name }}</div>
          <div class="tooltip-value">
            <span class="pulse-dot"></span>
            <strong>{{ hoveredMarker.count }}</strong>
            {{ hoveredMarker.count === 1 ? 'usuário rastreado' : 'usuários rastreados' }}
          </div>
        </div>
      </Transition>

      <!-- Legenda de Cores -->
      <div class="globe-legend">
        <div class="legend-item">
          <span class="color-dot land-dot"></span>
          <span>Cobertura Global</span>
        </div>
        <div class="legend-item">
          <span class="color-dot active-dot pulsating"></span>
          <span>Regiões com Eventos</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, watch, computed } from 'vue'

interface GeoStat {
  country: string
  state: string
  city: string
  count: number
}

interface Props {
  stats: GeoStat[]
  subtitle?: string
}

const props = withDefaults(defineProps<Props>(), {
  stats: () => [],
  subtitle: '',
})

// === Referências e Estado ===
const canvasRef = ref<HTMLCanvasElement | null>(null)
const loadingGeo = ref(true)
const hoveredMarker = ref<{ name: string; count: number; x: number; y: number } | null>(null)

// Coordenadas calculadas da Tooltip
const tooltipX = ref(0)
const tooltipY = ref(0)

// Estado da Câmera / Rotação
let yaw = 0.5 // Rotação Y
let pitch = -0.2 // Rotação X (leve inclinação)
let targetYaw = 0.5
let targetPitch = -0.2

// Controle de Drag/Interação
let isDragging = false
let startX = 0
let startY = 0
let lastMouseX = 0
let lastMouseY = 0
let velocityYaw = 0
let velocityPitch = 0
let lastInteractionTime = Date.now()

// Dados do Mapa
interface GridDot {
  lat: number
  lon: number
}
let landDots: GridDot[] = []

// Coordenadas estáticas de Capitais do BR e Países
const geoCoordinates: Record<string, { lat: number; lon: number; name: string }> = {
  // Estados Brasileiros
  'SP': { lat: -23.5505, lon: -46.6333, name: 'São Paulo' },
  'RJ': { lat: -22.9068, lon: -43.1729, name: 'Rio de Janeiro' },
  'MG': { lat: -19.9167, lon: -43.9345, name: 'Minas Gerais' },
  'BA': { lat: -12.9777, lon: -38.5016, name: 'Bahia' },
  'RS': { lat: -30.0346, lon: -51.2177, name: 'Rio Grande do Sul' },
  'PR': { lat: -25.4284, lon: -49.2733, name: 'Paraná' },
  'PE': { lat: -8.0578, lon: -34.8829, name: 'Pernambuco' },
  'CE': { lat: -3.7319, lon: -38.5267, name: 'Ceará' },
  'PA': { lat: -1.4558, lon: -48.4902, name: 'Pará' },
  'SC': { lat: -27.5954, lon: -48.5480, name: 'Santa Catarina' },
  'MA': { lat: -2.5307, lon: -44.3068, name: 'Maranhão' },
  'GO': { lat: -16.6869, lon: -49.2648, name: 'Goiás' },
  'AM': { lat: -3.1190, lon: -60.0217, name: 'Amazonas' },
  'PB': { lat: -7.1198, lon: -34.8450, name: 'Paraíba' },
  'ES': { lat: -20.3155, lon: -40.3128, name: 'Espírito Santo' },
  'RN': { lat: -5.7945, lon: -35.2110, name: 'Rio Grande do Norte' },
  'AL': { lat: -9.6658, lon: -35.7350, name: 'Alagoas' },
  'PI': { lat: -5.0920, lon: -42.8038, name: 'Piauí' },
  'MT': { lat: -15.6010, lon: -56.0974, name: 'Mato Grosso' },
  'DF': { lat: -15.7975, lon: -47.8919, name: 'Distrito Federal' },
  'MS': { lat: -20.4697, lon: -54.6201, name: 'Mato Grosso do Sul' },
  'SE': { lat: -10.9472, lon: -37.0731, name: 'Sergipe' },
  'RO': { lat: -8.7612, lon: -63.9039, name: 'Rondônia' },
  'TO': { lat: -10.1844, lon: -48.3336, name: 'Tocantins' },
  'AC': { lat: -9.9754, lon: -67.8080, name: 'Acre' },
  'AP': { lat: 0.0340, lon: -51.0694, name: 'Amapá' },
  'RR': { lat: 2.8235, lon: -60.6758, name: 'Roraima' },

  // Países Comuns
  'BR': { lat: -14.2350, lon: -51.9253, name: 'Brasil' },
  'US': { lat: 37.0902, lon: -95.7129, name: 'Estados Unidos' },
  'PT': { lat: 39.3999, lon: -8.2245, name: 'Portugal' },
  'AR': { lat: -38.4161, lon: -63.6167, name: 'Argentina' },
  'CL': { lat: -35.6751, lon: -71.5430, name: 'Chile' },
  'ES_COUNTRY': { lat: 40.4637, lon: -3.7492, name: 'Espanha' },
  'UY': { lat: -32.5228, lon: -55.7658, name: 'Uruguai' },
  'PY': { lat: -23.4425, lon: -58.4438, name: 'Paraguai' },
  'BO': { lat: -16.2902, lon: -63.5887, name: 'Bolívia' },
  'CO': { lat: 4.5709, lon: -72.9301, name: 'Colômbia' },
  'PE_COUNTRY': { lat: -9.1900, lon: -75.0152, name: 'Peru' },
  'MX': { lat: 23.6345, lon: -102.5528, name: 'México' },
  'GB': { lat: 55.3781, lon: -3.4360, name: 'Reino Unido' },
  'FR': { lat: 46.2276, lon: 2.2137, name: 'França' },
  'DE': { lat: 51.1657, lon: 10.4515, name: 'Alemanha' },
  'IT': { lat: 41.8719, lon: 12.5674, name: 'Itália' },
  'JP': { lat: 36.2048, lon: 138.2529, name: 'Japão' },
}

// Resolução dos dados para a localização de usuários
interface MappedUser {
  lat: number
  lon: number
  name: string
  count: number
  // Coordenadas 3D originais
  x0: number
  y0: number
  z0: number
  // Projetados em tela no frame atual
  projX: number
  projY: number
  projZ: number
}

const activeMarkers = ref<MappedUser[]>([])

// Mapeia estatísticas para coordenadas reais com jitter para evitar sobreposição total
const updateActiveMarkers = () => {
  activeMarkers.value = props.stats.map((s) => {
    const isBR = s.country === 'BR' || !s.country
    let baseCoord = { lat: -14.235, lon: -51.925, name: s.city || s.country || 'Desconhecido' }

    if (isBR && s.state) {
      const stateCoord = geoCoordinates[s.state.toUpperCase()]
      if (stateCoord) {
        baseCoord = stateCoord
      }
    } else {
      const code = s.country?.toUpperCase() || 'BR'
      let lookupCode = code === 'PE' ? 'PE_COUNTRY' : code
      if (lookupCode === 'ES') lookupCode = 'ES_COUNTRY'
      const countryCoord = geoCoordinates[lookupCode]
      if (countryCoord) {
        baseCoord = countryCoord
      }
    }

    // Gerar jitter determinístico baseado no nome da cidade
    let offsetLat = 0
    let offsetLon = 0
    let displayName = baseCoord.name
    if (s.city) {
      let hash = 0
      const seed = s.city + (s.state || '')
      for (let i = 0; i < seed.length; i++) {
        hash = seed.charCodeAt(i) + ((hash << 5) - hash)
      }
      // Offset de +/- 1.2 graus
      offsetLat = ((hash % 100) / 100) * 2.4 - 1.2
      offsetLon = (((hash >> 8) % 100) / 100) * 2.4 - 1.2
      displayName = `${s.city} (${s.state || s.country || 'BR'})`
    } else if (s.state) {
      displayName = `${s.state} (BR)`
    }

    const lat = baseCoord.lat + offsetLat
    const lon = baseCoord.lon + offsetLon

    // Pré-calcula coordenadas 3D unitárias
    const radLat = (lat * Math.PI) / 180
    const radLon = (lon * Math.PI) / 180
    const x0 = Math.cos(radLat) * Math.cos(radLon)
    const y0 = Math.sin(radLat)
    const z0 = Math.cos(radLat) * Math.sin(radLon)

    return {
      lat,
      lon,
      name: displayName,
      count: s.count,
      x0,
      y0,
      z0,
      projX: 0,
      projY: 0,
      projZ: 0,
    }
  })
}

// Watcher para atualizar dados reativos
watch(() => props.stats, updateActiveMarkers, { immediate: true, deep: true })

// Algoritmo Ray-Casting para detectar pontos em terra
const isPointInPolygon = (point: [number, number], polygon: [number, number][]) => {
  const x = point[0], y = point[1]
  let inside = false
  for (let i = 0, j = polygon.length - 1; i < polygon.length; j = i++) {
    const pi = polygon[i]
    const pj = polygon[j]
    if (pi && pj) {
      const xi = pi[0], yi = pi[1]
      const xj = pj[0], yj = pj[1]
      const intersect = ((yi > y) !== (yj > y)) && (x < ((xj - xi) * (y - yi)) / (yj - yi) + xi)
      if (intersect) inside = !inside
    }
  }
  return inside
}

// === Inicialização do Mapa ===
const loadGeoJson = async () => {
  try {
    const res = await fetch('/world-land.json')
    const geoJson = await res.json()

    // Extrair polígonos
    const landPolygons: [number, number][][] = []
    for (const feature of geoJson.features) {
      const geom = feature.geometry
      if (!geom) continue
      if (geom.type === 'Polygon') {
        landPolygons.push(geom.coordinates[0])
      } else if (geom.type === 'MultiPolygon') {
        for (const coords of geom.coordinates) {
          landPolygons.push(coords[0])
        }
      }
    }

    // Gerar malha de pontos do Globo (Dotted style)
    const dots: GridDot[] = []
    const latStep = 3.2 // Passo de latitude para otimizar densidade
    const lonStep = 3.2

    for (let lat = -80; lat <= 80; lat += latStep) {
      // Ajusta quantidade de pontos longitude por latitude para manter uniformidade
      const radLat = (lat * Math.PI) / 180
      const numLon = Math.round((360 * Math.cos(radLat)) / lonStep)
      if (numLon <= 0) continue
      const step = 360 / numLon

      for (let lon = -180; lon < 180; lon += step) {
        let isLand = false
        // Verifica se ponto está em terra
        for (const poly of landPolygons) {
          if (isPointInPolygon([lon, lat], poly)) {
            isLand = true
            break
          }
        }
        if (isLand) {
          dots.push({ lat, lon })
        }
      }
    }

    landDots = dots
    loadingGeo.value = false
  } catch (err) {
    console.error('Falha ao processar world-land.json:', err)
  }
}

// === Render Loop (Canvas) ===
let animationFrameId = 0

const render = () => {
  const canvas = canvasRef.value
  if (!canvas) {
    animationFrameId = requestAnimationFrame(render)
    return
  }

  const ctx = canvas.getContext('2d')
  if (!ctx) return

  // Ajuste de DPI/Retina
  const dpr = window.devicePixelRatio || 1
  const width = canvas.clientWidth
  const height = canvas.clientHeight
  if (canvas.width !== width * dpr || canvas.height !== height * dpr) {
    canvas.width = width * dpr
    canvas.height = height * dpr
  }
  ctx.resetTransform()
  ctx.scale(dpr, dpr)

  // Configurações do Globo
  const cx = width / 2
  const cy = height / 2
  const r = Math.min(width, height) * 0.42 // Raio do globo

  // Limpar
  ctx.clearRect(0, 0, width, height)

  // Desenhar atmosfera externa / Glow de fundo (Space/Ocean effect)
  const isLight = document.documentElement.getAttribute('data-theme') === 'light'
  
  // Gradiente do oceano (globo)
  const oceanGrad = ctx.createRadialGradient(cx - r/4, cy - r/4, r/8, cx, cy, r)
  if (isLight) {
    oceanGrad.addColorStop(0, '#f8fafc')
    oceanGrad.addColorStop(0.5, '#e2e8f0')
    oceanGrad.addColorStop(1, '#cbd5e1')
  } else {
    oceanGrad.addColorStop(0, '#1a1f29')
    oceanGrad.addColorStop(0.5, '#11141b')
    oceanGrad.addColorStop(1, '#07080a')
  }

  ctx.beginPath()
  ctx.arc(cx, cy, r, 0, Math.PI * 2)
  ctx.fillStyle = oceanGrad
  ctx.fill()

  // Sombra interna e borda 3D
  ctx.beginPath()
  ctx.arc(cx, cy, r, 0, Math.PI * 2)
  const glowGrad = ctx.createRadialGradient(cx - r/3, cy - r/3, r*0.7, cx, cy, r)
  if (isLight) {
    glowGrad.addColorStop(0, 'rgba(0, 0, 0, 0)')
    glowGrad.addColorStop(0.85, 'rgba(15, 23, 42, 0.05)')
    glowGrad.addColorStop(1, 'rgba(15, 23, 42, 0.3)')
  } else {
    glowGrad.addColorStop(0, 'rgba(0, 0, 0, 0)')
    glowGrad.addColorStop(0.85, 'rgba(28, 135, 193, 0.05)')
    glowGrad.addColorStop(1, 'rgba(0, 0, 0, 0.95)')
  }
  ctx.fillStyle = glowGrad
  ctx.fill()

  // --- Rotações ---
  // Inércia e atrito para drag suave
  if (!isDragging) {
    yaw += velocityYaw
    pitch += velocityPitch
    velocityYaw *= 0.95
    velocityPitch *= 0.95

    // Rotação automática devagar após 5s sem interação
    if (Date.now() - lastInteractionTime > 5000) {
      yaw += 0.0018
      // Retorna o pitch suavemente para o centro horizontal se estiver muito inclinado
      pitch = pitch * 0.98 + (-0.22) * 0.02
    }
  } else {
    // Segue o alvo suavizado
    yaw = yaw * 0.7 + targetYaw * 0.3
    pitch = pitch * 0.7 + targetPitch * 0.3
  }

  // Clampar pitch para não dar cambalhota completa nos polos (máx 80 graus)
  pitch = Math.max(-1.4, Math.min(1.4, pitch))

  const cosY = Math.cos(yaw)
  const sinY = Math.sin(yaw)
  const cosP = Math.cos(pitch)
  const sinP = Math.sin(pitch)

  // --- Desenhar a Malha de Terra (Pontos) ---
  const dotColor = isLight ? 'rgba(15, 23, 42, 0.15)' : 'rgba(255, 255, 255, 0.15)'
  const dotActiveColor = isLight ? 'rgba(28, 135, 193, 0.25)' : 'rgba(28, 135, 193, 0.3)'

  ctx.fillStyle = dotColor

  // Desenhar pontos de terra
  for (let i = 0; i < landDots.length; i++) {
    const dot = landDots[i]
    if (!dot) continue
    const radLat = (dot.lat * Math.PI) / 180
    const radLon = (dot.lon * Math.PI) / 180

    // Coordenadas 3D unitárias
    const x0 = Math.cos(radLat) * Math.cos(radLon)
    const y0 = Math.sin(radLat)
    const z0 = Math.cos(radLat) * Math.sin(radLon)

    // Rotacionar em Y (Yaw)
    const x1 = x0 * cosY - z0 * sinY
    const z1 = x0 * sinY + z0 * cosY

    // Rotacionar em X (Pitch)
    const y2 = y0 * cosP - z1 * sinP
    const z2 = y0 * sinP + z1 * cosP

    // Apenas desenhar o hemisfério visível (z2 > 0)
    if (z2 > 0) {
      const px = cx + x1 * r
      const py = cy - y2 * r

      // Sombreado nas bordas (cosseno do ângulo com a câmera para opacidade suave)
      const borderFade = z2 * z2 // decaimento quadrático para visual esférico incrível
      
      ctx.beginPath()
      ctx.arc(px, py, 1.4 * borderFade + 0.4, 0, Math.PI * 2)
      
      // Cor ligeiramente diferente se estiver perto de algum marcador ativo
      let markerHighlight = false
      for (const m of activeMarkers.value) {
        const dx = m.x0 - x0
        const dy = m.y0 - y0
        const dz = m.z0 - z0
        const dist = Math.sqrt(dx*dx + dy*dy + dz*dz)
        if (dist < 0.1) {
          markerHighlight = true
          break
        }
      }
      ctx.fillStyle = markerHighlight ? dotActiveColor : dotColor
      ctx.fill()
    }
  }

  // --- Desenhar Marcadores de Usuários (Verdes) ---
  const time = Date.now() * 0.003
  let closestMarker: MappedUser | null = null
  let minDist = 12 // Limiar de pixels para hover do mouse

  activeMarkers.value.forEach((m) => {
    // Rotacionar em Y (Yaw)
    const x1 = m.x0 * cosY - m.z0 * sinY
    const z1 = m.x0 * sinY + m.z0 * cosY

    // Rotacionar em X (Pitch)
    const y2 = m.y0 * cosP - z1 * sinP
    const z2 = m.y0 * sinP + z1 * cosP

    m.projX = cx + x1 * r
    m.projY = cy - y2 * r
    m.projZ = z2

    if (z2 > 0.05) {
      // Shading proporcional à curvatura
      const sizeScale = z2
      const baseRadius = Math.max(3, Math.min(8, 2.5 + Math.log2(m.count) * 1.5)) * sizeScale

      // 1. Efeito Pulsante de Glow Externo (Verde)
      const pulse = Math.sin(time + m.lat + m.lon) * 0.4 + 0.6 // 0.2 a 1.0
      ctx.beginPath()
      ctx.arc(m.projX, m.projY, baseRadius * (1.2 + pulse * 1.3), 0, Math.PI * 2)
      ctx.fillStyle = `rgba(34, 197, 94, ${0.15 * z2 * (1 - pulse * 0.7)})`
      ctx.fill()

      // 2. Anel Verde Fino Interno
      ctx.beginPath()
      ctx.arc(m.projX, m.projY, baseRadius * 1.2, 0, Math.PI * 2)
      ctx.strokeStyle = `rgba(34, 197, 94, ${0.4 * z2})`
      ctx.lineWidth = 1
      ctx.stroke()

      // 3. Ponto Central Forte
      ctx.beginPath()
      ctx.arc(m.projX, m.projY, baseRadius * 0.6, 0, Math.PI * 2)
      ctx.fillStyle = `rgba(34, 197, 94, ${0.9 * z2})`
      ctx.fill()

      // Detectar se está sob o mouse
      if (isDragging) return
      const dx = lastMouseX - m.projX
      const dy = lastMouseY - m.projY
      const dist = Math.sqrt(dx * dx + dy * dy)
      if (dist < minDist) {
        minDist = dist
        closestMarker = m
      }
    }
  })

  // --- Lógica de Hover / Tooltip ---
  if (closestMarker) {
    canvas.style.cursor = 'pointer'
    const m = closestMarker as MappedUser
    hoveredMarker.value = {
      name: m.name,
      count: m.count,
      x: m.projX,
      y: m.projY,
    }
    // Posiciona a tooltip com compensação do container do canvas
    const rect = canvas.getBoundingClientRect()
    tooltipX.value = m.projX
    tooltipY.value = m.projY - 12
  } else {
    canvas.style.cursor = isDragging ? 'grabbing' : 'grab'
    // Se o mouse se distanciou muito, fecha a tooltip
    if (hoveredMarker.value) {
      const dx = lastMouseX - hoveredMarker.value.x
      const dy = lastMouseY - hoveredMarker.value.y
      if (Math.sqrt(dx*dx + dy*dy) > 18) {
        hoveredMarker.value = null
      }
    }
  }

  // Desenhar efeito de atmosfera/brilho no limbo da Terra (Borda)
  ctx.beginPath()
  ctx.arc(cx, cy, r, 0, Math.PI * 2)
  const atmosphericGlow = ctx.createRadialGradient(cx, cy, r * 0.96, cx, cy, r * 1.01)
  if (isLight) {
    atmosphericGlow.addColorStop(0, 'rgba(255, 255, 255, 0)')
    atmosphericGlow.addColorStop(0.5, 'rgba(28, 135, 193, 0.12)')
    atmosphericGlow.addColorStop(1, 'rgba(28, 135, 193, 0)')
  } else {
    atmosphericGlow.addColorStop(0, 'rgba(28, 135, 193, 0)')
    atmosphericGlow.addColorStop(0.5, 'rgba(28, 135, 193, 0.25)')
    atmosphericGlow.addColorStop(1, 'rgba(28, 135, 193, 0)')
  }
  ctx.fillStyle = atmosphericGlow
  ctx.fill()

  // Borda fina externa do limbo
  ctx.beginPath()
  ctx.arc(cx, cy, r, 0, Math.PI * 2)
  ctx.strokeStyle = isLight ? 'rgba(15, 23, 42, 0.08)' : 'rgba(255, 255, 255, 0.08)'
  ctx.lineWidth = 1
  ctx.stroke()

  animationFrameId = requestAnimationFrame(render)
}

// === Eventos de Interação (Mouse) ===
const onMouseDown = (e: MouseEvent) => {
  isDragging = true
  startX = e.clientX
  startY = e.clientY
  targetYaw = yaw
  targetPitch = pitch
  velocityYaw = 0
  velocityPitch = 0
  lastInteractionTime = Date.now()
}

const onMouseMove = (e: MouseEvent) => {
  const canvas = canvasRef.value
  if (!canvas) return

  // Coordenadas locais do mouse no canvas
  const rect = canvas.getBoundingClientRect()
  lastMouseX = e.clientX - rect.left
  lastMouseY = e.clientY - rect.top

  if (!isDragging) return

  const dx = e.clientX - startX
  const dy = e.clientY - startY

  // Sensibilidade da rotação
  const sensitivity = 0.0045
  targetYaw = yaw + dx * sensitivity
  targetPitch = pitch - dy * sensitivity

  // Grava velocidades para inércia
  velocityYaw = dx * sensitivity * 0.15
  velocityPitch = -dy * sensitivity * 0.15

  // Avança âncora para cálculo incremental continuo
  startX = e.clientX
  startY = e.clientY
  lastInteractionTime = Date.now()
}

const onMouseUp = () => {
  isDragging = false
  lastInteractionTime = Date.now()
}

const onMouseLeave = () => {
  isDragging = false
  hoveredMarker.value = null
}

// === Eventos de Interação (Touch/Mobile) ===
const onTouchStart = (e: TouchEvent) => {
  if (e.touches.length !== 1) return
  const touch = e.touches[0]
  if (!touch) return
  isDragging = true
  startX = touch.clientX
  startY = touch.clientY
  targetYaw = yaw
  targetPitch = pitch
  velocityYaw = 0
  velocityPitch = 0
  lastInteractionTime = Date.now()
}

const onTouchMove = (e: TouchEvent) => {
  const canvas = canvasRef.value
  if (!canvas || !isDragging || e.touches.length !== 1) return
  const touch = e.touches[0]
  if (!touch) return

  const rect = canvas.getBoundingClientRect()
  lastMouseX = touch.clientX - rect.left
  lastMouseY = touch.clientY - rect.top

  const dx = touch.clientX - startX
  const dy = touch.clientY - startY

  const sensitivity = 0.007 // Um pouco mais sensível no touch
  targetYaw = yaw + dx * sensitivity
  targetPitch = pitch - dy * sensitivity

  velocityYaw = dx * sensitivity * 0.15
  velocityPitch = -dy * sensitivity * 0.15

  startX = touch.clientX
  startY = touch.clientY
  lastInteractionTime = Date.now()
}

const onTouchEnd = () => {
  isDragging = false
  lastInteractionTime = Date.now()
}

// === Lifecycle Hooks ===
onMounted(async () => {
  await loadGeoJson()
  render()
})

onBeforeUnmount(() => {
  cancelAnimationFrame(animationFrameId)
})
</script>

<style scoped>
.geo-globe-card {
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 1.25rem;
  box-shadow: var(--shadow-card);
  display: flex;
  flex-direction: column;
  position: relative;
  overflow: hidden;
  height: 100%;
}

.card-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: 0.5rem;
}

.card-title {
  font-family: 'Instrument Sans', sans-serif;
  font-size: 1.125rem;
  font-weight: 600;
  color: var(--text-strong);
}

.card-subtitle {
  font-size: 0.8125rem;
  color: var(--text-3);
  margin-top: 0.125rem;
}

.loading-badge {
  font-size: 0.75rem;
  background: var(--brand-soft);
  color: var(--brand);
  padding: 0.25rem 0.625rem;
  border-radius: 20px;
  border: 1px solid var(--brand-border);
  animation: pulse 1.8s infinite ease-in-out;
}

.globe-container {
  flex: 1;
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 320px;
  user-select: none;
}

.globe-canvas {
  width: 100%;
  height: 100%;
  max-height: 400px;
  aspect-ratio: 1.2 / 1;
  display: block;
}

/* Tooltip Premium Overlay */
.globe-tooltip {
  position: absolute;
  transform: translate(-50%, -100%);
  background: rgba(17, 20, 27, 0.85);
  backdrop-filter: blur(10px);
  -webkit-backdrop-filter: blur(10px);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 8px;
  padding: 0.625rem 0.875rem;
  box-shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.4), 0 8px 10px -6px rgba(0, 0, 0, 0.4);
  pointer-events: none;
  z-index: 100;
  white-space: nowrap;
}

[data-theme='light'] .globe-tooltip {
  background: rgba(255, 255, 255, 0.88);
  border: 1px solid rgba(15, 23, 42, 0.08);
  box-shadow: 0 10px 25px -5px rgba(15, 23, 42, 0.12);
}

.tooltip-city {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--text-strong);
  margin-bottom: 0.25rem;
}

.tooltip-value {
  font-size: 0.75rem;
  color: var(--text-2);
  display: flex;
  align-items: center;
  gap: 0.375rem;
}

.tooltip-value strong {
  color: var(--success);
  font-weight: 700;
}

/* Bolinha piscante na tooltip */
.pulse-dot {
  width: 6px;
  height: 6px;
  background: var(--success);
  border-radius: 50%;
  display: inline-block;
  box-shadow: 0 0 8px var(--success);
  animation: pulse-glow 1.4s infinite ease-in-out;
}

/* Legenda discreta */
.globe-legend {
  position: absolute;
  bottom: 0px;
  left: 0px;
  right: 0px;
  display: flex;
  justify-content: center;
  gap: 1.25rem;
  font-size: 0.75rem;
  color: var(--text-3);
  pointer-events: none;
}

.legend-item {
  display: flex;
  align-items: center;
  gap: 0.375rem;
}

.color-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  display: inline-block;
}

.land-dot {
  background: var(--text-3);
  opacity: 0.4;
}

.active-dot {
  background: var(--success);
  box-shadow: 0 0 6px var(--success);
}

/* Keyframes de animação */
@keyframes pulse {
  0%, 100% {
    opacity: 0.6;
  }
  50% {
    opacity: 1;
  }
}

@keyframes pulse-glow {
  0%, 100% {
    transform: scale(0.85);
    opacity: 0.6;
  }
  50% {
    transform: scale(1.2);
    opacity: 1;
    box-shadow: 0 0 12px var(--success);
  }
}

/* Transições da Tooltip */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s cubic-bezier(0.4, 0, 0.2, 1);
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
