<template>
  <div class="goal-progress-card" :class="{ 'is-reached': isReached }">
    <div class="goal-top-row">
      <div class="goal-title-group">
        <div class="goal-icon-box">
          <Trophy v-if="isReached" :size="16" />
          <Target v-else :size="16" />
        </div>
        <div>
          <div class="goal-heading-line">
            <h3 class="goal-heading">Meta de Vendas</h3>
            <span class="goal-mode-pill">
              {{ goalType === 'count' ? 'Por volume de vendas' : 'Por faturamento' }}
            </span>
            <span v-if="isReached" class="reached-pill">Meta atingida</span>
          </div>
          <p class="goal-sub">
            <span :class="{ blurred: isBlurredValue }">{{ formattedCurrent }}</span>
            <span> de </span>
            <strong :class="{ blurred: isBlurredValue }">{{ formattedTarget }}</strong>
            <span v-if="periodLabel"> • {{ periodLabel }}</span>
          </p>
        </div>
      </div>

      <div class="goal-percent-wrap">
        <span class="goal-percent-value">{{ formattedPercent }}%</span>
        <span class="goal-remaining" :class="{ blurred: isBlurredValue }">
          {{ remainingText }}
        </span>
      </div>
    </div>

    <div class="progress-track" role="progressbar" :aria-valuenow="clampedPercent" aria-valuemin="0" aria-valuemax="100">
      <div class="progress-fill" :style="{ width: `${clampedPercent}%` }" />
    </div>

    <div class="goal-ticks">
      <span>0%</span>
      <span>25%</span>
      <span>50%</span>
      <span>75%</span>
      <span>100%</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Target, Trophy } from 'lucide-vue-next'
import type { SalesGoalType } from '#shared/types'

const props = withDefaults(defineProps<{
  goalType: SalesGoalType
  target: number
  current: number
  currency?: string
  periodLabel?: string
  blurred?: boolean
}>(), {
  currency: 'BRL',
  periodLabel: '',
  blurred: false,
})

const safeTarget = computed(() => Math.max(0, Number(props.target) || 0))
const safeCurrent = computed(() => Math.max(0, Number(props.current) || 0))

const rawPercent = computed(() => {
  if (safeTarget.value <= 0) return 0
  return (safeCurrent.value / safeTarget.value) * 100
})

const clampedPercent = computed(() => Math.min(100, Math.max(0, rawPercent.value)))
const isReached = computed(() => safeTarget.value > 0 && safeCurrent.value >= safeTarget.value)
const isBlurredValue = computed(() => Boolean(props.blurred && props.goalType === 'revenue'))

const formattedPercent = computed(() => {
  const p = rawPercent.value
  if (p <= 0) return '0'
  if (p < 10) return p.toFixed(1).replace('.', ',')
  return String(Math.round(p))
})

const formatMoneyVal = (val: number) => {
  try {
    return new Intl.NumberFormat('pt-BR', { style: 'currency', currency: props.currency || 'BRL' }).format(val)
  } catch {
    return `R$ ${val.toFixed(2)}`
  }
}

const formatGoalUnit = (val: number) => {
  if (props.goalType === 'count') {
    const n = Math.round(val)
    return `${n} ${n === 1 ? 'venda' : 'vendas'}`
  }
  return formatMoneyVal(val)
}

const formattedCurrent = computed(() => formatGoalUnit(safeCurrent.value))
const formattedTarget = computed(() => formatGoalUnit(safeTarget.value))

const remainingText = computed(() => {
  const diff = safeTarget.value - safeCurrent.value
  if (diff > 0) {
    return `Faltam ${formatGoalUnit(diff)}`
  }
  if (diff < 0) {
    return `+${formatGoalUnit(Math.abs(diff))} acima da meta`
  }
  return '100% concluída'
})
</script>

<style scoped>
.goal-progress-card {
  background: var(--surface-2);
  border: 1px solid var(--border-soft);
  border-radius: 12px;
  padding: 1rem 1.25rem;
  display: flex;
  flex-direction: column;
  gap: 0.65rem;
}
.goal-progress-card.is-reached {
  border-color: var(--brand-border);
}
.goal-top-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  flex-wrap: wrap;
}
.goal-title-group {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  min-width: 0;
}
.goal-icon-box {
  width: 36px;
  height: 36px;
  border-radius: 9px;
  background: var(--brand-soft);
  border: 1px solid var(--brand-border);
  color: var(--brand);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.goal-heading-line {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
}
.goal-heading {
  font-size: var(--fs-lg);
  font-weight: var(--fw-bold);
  color: var(--text-strong);
  margin: 0;
}
.goal-mode-pill {
  font-size: var(--fs-2xs);
  font-weight: var(--fw-semibold);
  color: var(--text-2);
  background: var(--surface-deep);
  border: 1px solid var(--border);
  padding: 0.12rem 0.48rem;
  border-radius: 999px;
}
.reached-pill {
  font-size: var(--fs-2xs);
  font-weight: var(--fw-bold);
  color: var(--brand-text);
  background: var(--brand);
  padding: 0.12rem 0.5rem;
  border-radius: 999px;
}
.goal-sub {
  font-size: var(--fs-base);
  color: var(--text-2);
  margin: 0.15rem 0 0;
}
.goal-sub strong { color: var(--text-strong); font-weight: var(--fw-semibold); }
.goal-percent-wrap {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 0.1rem;
}
.goal-percent-value {
  font-size: var(--fs-2xl);
  font-weight: var(--fw-bold);
  color: var(--brand);
  line-height: 1.1;
  letter-spacing: -0.02em;
}
.goal-remaining {
  font-size: var(--fs-xs);
  color: var(--text-3);
}
.blurred { filter: blur(6px); user-select: none; }
.progress-track {
  width: 100%;
  height: 10px;
  border-radius: 999px;
  background: var(--surface-deep);
  border: 1px solid var(--border-soft);
  overflow: hidden;
}
.progress-fill {
  height: 100%;
  border-radius: 999px;
  background: linear-gradient(90deg, var(--brand-strong) 0%, var(--brand) 100%);
  transition: width 0.45s cubic-bezier(0.16, 1, 0.3, 1);
}
.goal-ticks {
  display: flex;
  justify-content: space-between;
  font-size: var(--fs-2xs);
  color: var(--text-3);
  font-family: var(--font-mono);
}
@media (max-width: 640px) {
  .goal-percent-wrap { align-items: flex-start; }
}
</style>
