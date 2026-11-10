<template>
  <div class="events-chart">
    <div class="chart-header">
      <div>
        <h3 class="chart-title">Eventos por dia</h3>
        <p class="chart-subtitle">{{ subtitle }}</p>
      </div>
    </div>
    <div class="chart-canvas-wrap">
      <canvas v-if="hasData" ref="canvasRef" />
      <div v-else class="chart-empty">
        Sem dados no período selecionado.
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import {
  Chart, LineController, LineElement, PointElement,
  LinearScale, TimeSeriesScale, CategoryScale, Tooltip, Legend, Filler,
} from 'chart.js'
import type { CounterSeriesPoint } from '#shared/types'

interface Props {
  points: CounterSeriesPoint[]
  granularity: 'hour' | 'day'
  subtitle?: string
}

const props = withDefaults(defineProps<Props>(), { subtitle: '' })

Chart.register(
  LineController, LineElement, PointElement,
  LinearScale, TimeSeriesScale, CategoryScale, Tooltip, Legend, Filler
)

const canvasRef = ref<HTMLCanvasElement | null>(null)
let chart: Chart<'line'> | null = null
const { isDark } = useTheme()

const hasData = computed(() => props.points.length > 0)

const getCssVar = (val: string, fallback: string) => {
  if (typeof document === 'undefined') return fallback
  return getComputedStyle(document.documentElement).getPropertyValue(val).trim() || fallback
}

const formatBucket = (iso: string, daily: boolean): string => {
  const d = new Date(iso)
  if (daily) return d.toLocaleDateString('pt-BR', { day: '2-digit', month: '2-digit' })
  return d.toLocaleString('pt-BR', { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' })
}

const buildDatasets = () => {
  const eventColors: Record<string, string> = {
    PageView: getCssVar('--brand', '#A2F07C'),
    ViewContent: getCssVar('--brand-strong', '#8EE85F'),
    AddToCart: getCssVar('--warning', '#F59E0B'),
    InitiateCheckout: getCssVar('--info', '#0B4B4D'),
    Lead: getCssVar('--success', '#10B981'),
    Purchase: getCssVar('--success', '#22C55E'),
    AddPaymentInfo: getCssVar('--warning', '#EAB308'),
    Contact: getCssVar('--info', '#EC4899'),
    CompleteRegistration: getCssVar('--brand', '#8B5CF6'),
  }
  const fallbackColor = getCssVar('--text-3', '#737373')

  const buckets = new Map<string, Map<string, number>>()
  for (const p of props.points) {
    if (!buckets.has(p.event_name)) buckets.set(p.event_name, new Map())
    buckets.get(p.event_name)!.set(p.bucket_at, p.count_total)
  }

  const allTimestamps = Array.from(new Set(props.points.map((p) => p.bucket_at))).sort()
  const labels = allTimestamps.map((t) => formatBucket(t, props.granularity === 'day'))

  const datasets = Array.from(buckets.entries()).map(([eventName, bucketMap]) => {
    const data = allTimestamps.map((t) => bucketMap.get(t) ?? 0)
    const color = eventColors[eventName] ?? fallbackColor
    return {
      label: eventName,
      data,
      borderColor: color,
      backgroundColor: color + '22',
      borderWidth: 2,
      pointRadius: 3,
      pointHoverRadius: 5,
      tension: 0.3,
      fill: false,
    }
  })

  return { labels, datasets }
}

const buildOptions = () => {
  const cssText = (val: string) => getCssVar(val, '#777')
  const textColor = cssText('--text-2')
  const gridColor = cssText('--border')

  return {
    responsive: true,
    maintainAspectRatio: false,
    interaction: { mode: 'index' as const, intersect: false },
    plugins: {
      legend: {
        position: 'top' as const,
        labels: { color: textColor, usePointStyle: true, boxWidth: 8, padding: 12, font: { size: 12 } },
      },
      tooltip: {
        backgroundColor: cssText('--surface-2'),
        titleColor: cssText('--text-strong'),
        bodyColor: cssText('--text-1'),
        borderColor: cssText('--border'),
        borderWidth: 1,
        padding: 10,
      },
    },
    scales: {
      x: {
        ticks: { color: textColor, maxRotation: 0, autoSkipPadding: 16 },
        grid: { color: gridColor, drawTicks: false },
      },
      y: {
        beginAtZero: true,
        ticks: { color: textColor, precision: 0 },
        grid: { color: gridColor, drawTicks: false },
      },
    },
  }
}

const render = () => {
  if (!canvasRef.value || !hasData.value) return
  if (chart) {
    chart.destroy()
    chart = null
  }
  const { labels, datasets } = buildDatasets()
  chart = new Chart(canvasRef.value, {
    type: 'line',
    data: { labels, datasets },
    options: buildOptions(),
  })
}

onMounted(render)

watch(
  () => [props.points, props.granularity, isDark.value],
  async () => {
    await nextTick()
    render()
  },
  { deep: true }
)

onBeforeUnmount(() => {
  if (chart) chart.destroy()
})
</script>

<style scoped>
.events-chart {
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 1.125rem;
  box-shadow: var(--shadow-card);
  height: 100%;
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
}

.chart-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 0.875rem;
}

.chart-title {
  font-family: 'Instrument Sans', sans-serif;
  font-size: var(--fs-lg);
  font-weight: var(--fw-semibold);
  color: var(--text-strong);
  letter-spacing: -0.01em;
}

.chart-subtitle {
  font-size: var(--fs-sm);
  color: var(--text-3);
  margin-top: 0.125rem;
}

.chart-canvas-wrap {
  position: relative;
  flex: 1;
  min-height: 280px;
}

.chart-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: var(--text-3);
  font-size: var(--fs-base);
}
</style>
