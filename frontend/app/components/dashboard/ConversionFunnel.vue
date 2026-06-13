<template>
  <div class="funnel-card">
    <div class="funnel-header">
      <h3 class="funnel-title">Funil de conversão</h3>
      <p class="funnel-subtitle">{{ subtitle }}</p>
    </div>

    <div v-if="!hasData" class="funnel-empty">
      Sem eventos no período.
    </div>

    <div v-else class="funnel-steps">
      <div
        v-for="(step, idx) in steps"
        :key="step.event"
        class="funnel-step"
      >
        <div class="step-header">
          <span class="step-name">{{ step.label }}</span>
          <span class="step-count">{{ formatNumber(step.count) }}</span>
        </div>

        <div class="step-bar-wrap">
          <div
            class="step-bar"
            :style="{
              width: relativeWidth(step.count) + '%',
              background: step.color,
            }"
          />
          <div class="step-bar-bg" />
        </div>

        <div class="step-meta">
          <span v-if="idx > 0" class="step-conversion">
            {{ conversionFromPrevious(idx) }}
            <span class="step-conversion-label">vs. {{ steps[idx - 1].label }}</span>
          </span>
          <span v-if="idx > 0" class="step-conversion total">
            {{ conversionFromTop(idx) }}
            <span class="step-conversion-label">vs. {{ steps[0].label }}</span>
          </span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { CounterSummaryRow } from '#shared/types'

interface Props {
  events: CounterSummaryRow[]
  subtitle?: string
}

const props = withDefaults(defineProps<Props>(), {
  subtitle: '',
})

const FUNNEL_DEF: Array<{ event: string; label: string; color: string }> = [
  { event: 'PageView', label: 'Visitas', color: '#1C87C1' },
  { event: 'ViewContent', label: 'View Content', color: '#06B6D4' },
  { event: 'AddToCart', label: 'Add to Cart', color: '#F59E0B' },
  { event: 'InitiateCheckout', label: 'Initiate Checkout', color: '#A855F7' },
  { event: 'Purchase', label: 'Purchase', color: '#22C55E' },
]

const eventCounts = computed(() => {
  const map = new Map<string, number>()
  for (const row of props.events) {
    map.set(row.event_name, row.count_total)
  }
  return map
})

const steps = computed(() =>
  FUNNEL_DEF.map((s) => ({
    ...s,
    count: eventCounts.value.get(s.event) ?? 0,
  }))
)

const hasData = computed(() => steps.value.some((s) => s.count > 0))

const maxCount = computed(() =>
  steps.value.reduce((m, s) => Math.max(m, s.count), 0) || 1
)

const relativeWidth = (count: number): number => {
  return Math.max(2, (count / maxCount.value) * 100)
}

const formatNumber = (n: number): string => new Intl.NumberFormat('pt-BR').format(n)

const formatPercent = (n: number, d: number): string => {
  if (!d) return '—'
  return new Intl.NumberFormat('pt-BR', {
    style: 'percent',
    minimumFractionDigits: 1,
    maximumFractionDigits: 1,
  }).format(n / d)
}

const conversionFromPrevious = (idx: number): string => {
  const cur = steps.value[idx].count
  const prev = steps.value[idx - 1].count
  return formatPercent(cur, prev)
}

const conversionFromTop = (idx: number): string => {
  const cur = steps.value[idx].count
  const top = steps.value[0].count
  return formatPercent(cur, top)
}
</script>

<style scoped>
.funnel-card {
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 1.25rem;
  box-shadow: var(--shadow-card);
}

.funnel-header {
  margin-bottom: 1rem;
}

.funnel-title {
  font-family: 'Instrument Sans', sans-serif;
  font-size: 1.125rem;
  font-weight: 600;
  color: var(--text-strong);
}

.funnel-subtitle {
  font-size: 0.8125rem;
  color: var(--text-3);
  margin-top: 0.125rem;
}

.funnel-empty {
  padding: 2rem 1rem;
  text-align: center;
  color: var(--text-3);
  font-size: 0.875rem;
  font-style: italic;
}

.funnel-steps {
  display: flex;
  flex-direction: column;
  gap: 0.875rem;
}

.funnel-step {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
}

.step-header {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
}

.step-name {
  font-size: 0.8125rem;
  color: var(--text-2);
  font-weight: 500;
}

.step-count {
  font-family: 'Instrument Sans', sans-serif;
  font-size: 1.25rem;
  font-weight: 700;
  color: var(--text-strong);
}

.step-bar-wrap {
  position: relative;
  height: 10px;
  border-radius: 6px;
  overflow: hidden;
}

.step-bar-bg {
  position: absolute;
  inset: 0;
  background: var(--surface-soft);
  border-radius: 6px;
  z-index: 0;
}

.step-bar {
  position: relative;
  z-index: 1;
  height: 100%;
  border-radius: 6px;
  transition: width 0.5s ease;
}

.step-meta {
  display: flex;
  align-items: center;
  gap: 1rem;
  flex-wrap: wrap;
  font-size: 0.75rem;
  color: var(--text-2);
}

.step-conversion {
  display: inline-flex;
  align-items: baseline;
  gap: 0.375rem;
  font-weight: 600;
}

.step-conversion.total {
  color: var(--brand);
}

.step-conversion-label {
  color: var(--text-3);
  font-weight: 400;
}
</style>
