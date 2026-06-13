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
  Chart,
  LineController,
  LineElement,
  PointElement,
  LinearScale,
  TimeSeriesScale,
  CategoryScale,
  Tooltip,
  Legend,
  Filler,
} from 'chart.js'
import type { CounterSeriesPoint } from '#shared/types'

interface Props {
  points: CounterSeriesPoint[]
  granularity: 'hour' | 'day'
  subtitle?: string
}

const props = withDefaults(defineProps<Props>(), {
  subtitle: '',
})

Chart.register(
  LineController,
  LineElement,
  PointElement,
  LinearScale,
  TimeSeriesScale,
  CategoryScale,
  Tooltip,
  Legend,
  Filler
)

const canvasRef = ref<HTMLCanvasElement | null>(null)
let chart: Chart<'line'> | null = null

const hasData = computed(() => props.points.length > 0)

// Cores por evento
const eventColors: Record<string, string> = {
  PageView: '#1C87C1',
  ViewContent: '#06B6D4',
  AddToCart: '#F59E0B',
  InitiateCheckout: '#A855F7',
  Lead: '#10B981',
  Purchase: '#22C55E',
  AddPaymentInfo: '#EAB308',
  Contact: '#EC4899',
  CompleteRegistration: '#8B5CF6',
}
const fallbackColor = '#737373'

const formatBucket = (iso: string, daily: boolean): string => {
  const d = new Date(iso)
  if (daily) {
    return d.toLocaleDateString('pt-BR', { day: '2-digit', month: '2-digit' })
  }
  return d.toLocaleString('pt-BR', {
    day: '2-digit',
    month: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

const buildDatasets = () => {
  const buckets = new Map<string, Map<string, number>>()
  for (const p of props.points) {
    const evt = p.event_name
    if (!buckets.has(evt)) buckets.set(evt, new Map())
    buckets.get(evt)!.set(p.bucket_at, p.count_total)
  }

  const allTimestamps = Array.from(
    new Set(props.points.map((p) => p.bucket_at))
  ).sort()

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
  const cssText = (val: string) =>
    getComputedStyle(document.documentElement).getPropertyValue(val).trim() || '#777'
  const textColor = cssText('--text-2')
  const gridColor = cssText('--border')

  return {
    responsive: true,
    maintainAspectRatio: false,
    interaction: { mode: 'index' as const, intersect: false },
    plugins: {
      legend: {
        position: 'top' as const,
        labels: {
          color: textColor,
          usePointStyle: true,
          boxWidth: 8,
          padding: 12,
          font: { size: 12 },
        },
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
        ticks: {
          color: textColor,
          precision: 0,
        },
        grid: { color: gridColor, drawTicks: false },
      },
    },
  }
}

const render = () => {
  if (!canvasRef.value) return
  if (!hasData.value) return

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
watch(() => [props.points, props.granularity], render, { deep: true })
onBeforeUnmount(() => {
  if (chart) chart.destroy()
})
</script>

<style scoped>
.events-chart {
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 1.25rem;
  box-shadow: var(--shadow-card);
}

.chart-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 1rem;
}

.chart-title {
  font-family: 'Instrument Sans', sans-serif;
  font-size: 1.125rem;
  font-weight: 600;
  color: var(--text-strong);
}

.chart-subtitle {
  font-size: 0.8125rem;
  color: var(--text-3);
  margin-top: 0.125rem;
}

.chart-canvas-wrap {
  position: relative;
  height: 300px;
}

.chart-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: var(--text-3);
  font-size: 0.875rem;
  font-style: italic;
}
</style>
