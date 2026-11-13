<template>
  <div class="kpi-card" :class="`variant-${variant}`">
    <div class="kpi-side-accent" aria-hidden="true" />

    <div class="kpi-header">
      <p class="kpi-label">{{ label }}</p>
      <div class="kpi-header-right">
        <span
          v-if="hasChange && !blurred"
          class="kpi-change-pill"
          :class="changeToneClass"
          :title="previousLabel ? `Período anterior: ${previousLabel}` : undefined"
        >
          <TrendingUp v-if="(changePercent ?? 0) > 0" :size="11" />
          <TrendingDown v-else-if="(changePercent ?? 0) < 0" :size="11" />
          <span>{{ formattedChange }}</span>
        </span>
        <div class="kpi-icon">
          <component :is="resolvedIcon" :size="15" />
        </div>
      </div>
    </div>

    <div class="kpi-content">
      <p class="kpi-value" :class="{ 'is-blurred': blurred }">{{ formatted }}</p>

      <div v-if="comparisons?.length" class="kpi-compare-row">
        <span
          v-for="(chip, idx) in comparisons"
          :key="`${chip.label}-${idx}`"
          class="compare-chip"
          :class="`tone-${chip.tone}`"
        >
          <MetaAdsIcon v-if="chip.tone === 'meta'" :size="11" class="chip-brand-icon" />
          <GoogleAdsIcon v-else-if="chip.tone === 'google'" :size="11" class="chip-brand-icon" />
          <span class="compare-lbl">{{ chip.label }} =</span>
          <strong class="compare-val">{{ chip.value }}</strong>
        </span>
      </div>

      <div v-if="delivery" class="kpi-delivery-row">
        <span class="delivery-item is-sent">
          <span class="status-dot dot-sent" />
          {{ delivery.sentLabel }}: <strong>{{ formatInt(delivery.sent) }}</strong>
        </span>
        <span class="delivery-sep">·</span>
        <span class="delivery-item" :class="delivery.notSent > 0 ? 'is-pending' : 'is-clean'">
          <span class="status-dot" :class="delivery.notSent > 0 ? 'dot-pending' : 'dot-clean'" />
          {{ delivery.notSentLabel }}: <strong>{{ formatInt(delivery.notSent) }}</strong>
        </span>
      </div>

      <p v-else-if="hint" class="kpi-hint">{{ hint }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Component } from 'vue'
import { TrendingDown, TrendingUp } from 'lucide-vue-next'
import { MetaAdsIcon, GoogleAdsIcon } from '~/utils/adIcons'
import { formatMoney, type CardComparisonChip, type CardDeliveryAudit } from '~/utils/dashboardCardDetails'

interface Props {
  label: string
  value: number | string
  hint?: string
  icon: Component
  variant?: 'primary' | 'success' | 'warning' | 'danger' | 'info'
  format?: 'number' | 'currency' | 'percent' | 'roas'
  currency?: string
  changePercent?: number | null
  changeInverse?: boolean
  previousLabel?: string
  blurred?: boolean
  comparisons?: CardComparisonChip[]
  delivery?: CardDeliveryAudit
  channel?: 'meta' | 'google' | null
}

const props = withDefaults(defineProps<Props>(), {
  variant: 'primary',
  format: 'number',
  currency: 'BRL',
  changePercent: null,
  changeInverse: false,
  blurred: false,
  channel: null,
})

const resolvedIcon = computed<Component>(() => {
  if (props.channel === 'meta') return MetaAdsIcon
  if (props.channel === 'google') return GoogleAdsIcon
  const lower = (props.label || '').toLowerCase()
  if (lower.includes('meta ads')) return MetaAdsIcon
  if (lower.includes('google ads')) return GoogleAdsIcon
  return props.icon
})

const formatInt = (n: number) => new Intl.NumberFormat('pt-BR').format(n || 0)

const formatted = computed(() => {
  if (typeof props.value === 'string') return props.value
  if (props.format === 'roas') {
    return `${new Intl.NumberFormat('pt-BR', { minimumFractionDigits: 2, maximumFractionDigits: 2 }).format(props.value)}x`
  }
  if (props.format === 'currency') {
    return formatMoney(props.value, props.currency)
  }
  if (props.format === 'percent') {
    return new Intl.NumberFormat('pt-BR', { style: 'percent', minimumFractionDigits: 1, maximumFractionDigits: 1 }).format(props.value)
  }
  return new Intl.NumberFormat('pt-BR').format(props.value)
})

const hasChange = computed(() => typeof props.changePercent === 'number' && Number.isFinite(props.changePercent))
const formattedChange = computed(() => {
  const v = props.changePercent ?? 0
  return `${v > 0 ? '+' : ''}${v.toFixed(1).replace('.', ',')}%`
})
const changeToneClass = computed(() => {
  const v = props.changePercent ?? 0
  if (Math.abs(v) < 0.05) return 'change-neutral'
  return (props.changeInverse ? v < 0 : v > 0) ? 'change-up' : 'change-down'
})
</script>

<style scoped>
.kpi-card { position: relative; display: flex; flex-direction: column; justify-content: space-between; gap: 0.55rem; padding: 0.95rem 1.05rem 0.95rem 1.15rem; background: var(--surface-2); border: 1px solid var(--border); border-radius: 12px; transition: border-color 0.18s ease, transform 0.18s ease; height: 100%; box-sizing: border-box; overflow: hidden; }
.kpi-card:hover { border-color: var(--brand-border); transform: translateY(-1px); }
.kpi-side-accent { position: absolute; left: 0; top: 12px; bottom: 12px; width: 3px; border-radius: 0 3px 3px 0; background: var(--brand); opacity: 0.9; }
.variant-danger .kpi-side-accent { background: var(--danger); }
.variant-warning .kpi-side-accent { background: var(--warning); }
.variant-danger .kpi-icon { background: var(--danger-soft); color: var(--danger); }
.variant-warning .kpi-icon { background: var(--warning-soft); color: var(--warning); }

.kpi-header { display: flex; align-items: center; justify-content: space-between; gap: 0.5rem; }
.kpi-label { font-size: var(--fs-xs); color: var(--text-3); font-weight: var(--fw-semibold); text-transform: uppercase; letter-spacing: 0.04em; margin: 0; line-height: 1.25; }
.kpi-header-right { display: flex; align-items: center; gap: 0.4rem; flex-shrink: 0; }
.kpi-change-pill { display: inline-flex; align-items: center; gap: 0.2rem; padding: 0.12rem 0.42rem; border-radius: 999px; font-size: var(--fs-2xs); font-weight: var(--fw-semibold); font-family: var(--font-mono); white-space: nowrap; }
.change-up { background: var(--brand-soft); color: var(--brand); }
.change-down { background: var(--danger-soft); color: var(--danger); border: 1px solid var(--danger-border); }
.change-neutral { background: var(--surface-soft); color: var(--text-3); }

.kpi-icon { flex-shrink: 0; width: 26px; height: 26px; border-radius: 7px; display: flex; align-items: center; justify-content: center; background: var(--brand-soft); color: var(--brand); }

.kpi-content { min-width: 0; display: flex; flex-direction: column; gap: 0.42rem; }
.kpi-value { font-family: var(--font-heading); font-size: var(--fs-2xl); font-weight: var(--fw-bold); color: var(--text-strong); line-height: 1.15; letter-spacing: -0.02em; margin: 0; transition: filter 0.2s ease; }
.kpi-value.is-blurred { filter: blur(7px); user-select: none; }

.kpi-compare-row { display: flex; flex-wrap: wrap; align-items: center; gap: 0.32rem; }
.compare-chip { display: inline-flex; align-items: center; gap: 0.22rem; padding: 0.14rem 0.45rem; border-radius: 6px; font-size: var(--fs-2xs); background: var(--surface-deep); border: 1px solid var(--border-soft); color: var(--text-2); }
.chip-brand-icon { flex-shrink: 0; opacity: 0.85; }
.compare-lbl { color: var(--text-3); font-weight: var(--fw-medium); }
.compare-val { font-family: var(--font-mono); font-weight: var(--fw-semibold); color: var(--text-strong); }
.tone-panel { border-color: var(--brand-border); background: var(--brand-soft-2); }
.tone-panel .compare-val { color: var(--brand); }

.kpi-delivery-row { display: flex; flex-wrap: wrap; align-items: center; gap: 0.38rem; padding-top: 0.35rem; border-top: 1px solid var(--border-soft); font-size: var(--fs-2xs); }
.delivery-item { display: inline-flex; align-items: center; gap: 0.26rem; color: var(--text-3); }
.delivery-item strong { font-family: var(--font-mono); font-weight: var(--fw-semibold); }
.is-sent strong { color: var(--brand); }
.is-pending, .is-pending strong { color: var(--text-strong); }
.is-clean strong { color: var(--text-2); }
.delivery-sep { color: var(--text-3); opacity: 0.5; }
.status-dot { width: 6px; height: 6px; border-radius: 999px; flex-shrink: 0; }
.dot-sent { background: var(--brand); }
.dot-pending { background: var(--text-2); }
.dot-clean { background: var(--text-3); opacity: 0.5; }
.kpi-hint { font-size: var(--fs-xs); color: var(--text-3); margin: 0; }
</style>
