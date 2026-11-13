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
        v-for="(step, idx) in enrichedSteps"
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
            <span class="step-conversion-label">vs. {{ enrichedSteps[idx - 1]?.label }}</span>
          </span>
          <span v-if="idx > 0" class="step-conversion total">
            {{ conversionFromTop(idx) }}
            <span class="step-conversion-label">vs. {{ enrichedSteps[0]?.label }}</span>
          </span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { CounterSummaryRow } from '#shared/types'

interface FunnelStepDef {
  event: string
  label: string
  color: string
}

interface Props {
  events: CounterSummaryRow[]
  steps: FunnelStepDef[]
  subtitle?: string
}

const props = withDefaults(defineProps<Props>(), {
  subtitle: '',
})

const eventCounts = computed(() => {
  const map = new Map<string, number>()
  for (const row of props.events) {
    map.set(row.event_name, row.count_total)
  }
  return map
})

const enrichedSteps = computed(() =>
  props.steps.map((s) => ({
    ...s,
    count: eventCounts.value.get(s.event) ?? 0,
  }))
)

const hasData = computed(() => enrichedSteps.value.some((s) => s.count > 0))

const maxCount = computed(() =>
  enrichedSteps.value.reduce((m, s) => Math.max(m, s.count), 0) || 1
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
  const cur = enrichedSteps.value[idx]?.count ?? 0
  const prev = enrichedSteps.value[idx - 1]?.count ?? 0
  return formatPercent(cur, prev)
}

const conversionFromTop = (idx: number): string => {
  const cur = enrichedSteps.value[idx]?.count ?? 0
  const top = enrichedSteps.value[0]?.count ?? 0
  return formatPercent(cur, top)
}
</script>

<style scoped>
.funnel-card {
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 1.125rem;
  box-shadow: var(--shadow-card);
  height: 100%;
  box-sizing: border-box;
}

.funnel-header {
  margin-bottom: 0.875rem;
}

.funnel-title {
  font-family: var(--font-sans);
  font-size: var(--fs-lg);
  font-weight: var(--fw-semibold);
  color: var(--text-strong);
  letter-spacing: -0.01em;
}

.funnel-subtitle {
  font-size: var(--fs-sm);
  color: var(--text-3);
  margin-top: 0.125rem;
}

.funnel-empty {
  padding: 2.5rem 1rem;
  text-align: center;
  color: var(--text-3);
  font-size: var(--fs-base);
}

.funnel-steps {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.funnel-step {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.step-header {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
}

.step-name {
  font-size: var(--fs-sm);
  color: var(--text-2);
  font-weight: var(--fw-medium);
}

.step-count {
  font-family: var(--font-sans);
  font-size: var(--fs-xl);
  font-weight: var(--fw-bold);
  color: var(--text-strong);
}

.step-bar-wrap {
  position: relative;
  height: 8px;
  border-radius: 4px;
  overflow: hidden;
}

.step-bar-bg {
  position: absolute;
  inset: 0;
  background: var(--surface-soft);
  border-radius: 4px;
  z-index: 0;
}

.step-bar {
  position: relative;
  z-index: 1;
  height: 100%;
  border-radius: 4px;
  transition: width 0.4s cubic-bezier(0.16, 1, 0.3, 1);
}

.step-meta {
  display: flex;
  align-items: center;
  gap: 0.875rem;
  flex-wrap: wrap;
  font-size: var(--fs-xs);
  color: var(--text-2);
  margin-top: 0.125rem;
}

.step-conversion {
  display: inline-flex;
  align-items: baseline;
  gap: 0.25rem;
  font-weight: var(--fw-semibold);
}

.step-conversion.total {
  color: var(--brand);
}

.step-conversion-label {
  color: var(--text-3);
  font-weight: var(--fw-normal);
}
</style>
