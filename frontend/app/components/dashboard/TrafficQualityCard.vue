<template>
  <div class="traffic-card">
    <div class="traffic-header">
      <div class="header-info">
        <div class="icon-badge">
          <ShieldCheck :size="16" class="text-brand" />
        </div>
        <div>
          <h3 class="card-title">Auditoria de Tráfego</h3>
          <p class="card-subtitle">Análise heurística de visitantes humanos vs. bots e crawlers</p>
        </div>
      </div>

      <div class="trust-pill" :class="trafficBadgeClass">
        <span class="trust-dot" />
        <span>{{ trafficStatusLabel }}</span>
      </div>
    </div>

    <!-- Barra de Distribuição Segmentada -->
    <div class="distribution-track">
      <div
        class="track-segment segment-human"
        :style="{ width: `${humanPercent}%` }"
        :title="`Provavelmente Humano: ${humanPercent}% (${humanCount} visitantes)`"
      />
      <div
        class="track-segment segment-bot"
        :style="{ width: `${botPercent}%` }"
        :title="`Bots & Crawlers: ${botPercent}% (${botCount} acessos)`"
      />
      <div
        class="track-segment segment-inconclusive"
        :style="{ width: `${inconclusivePercent}%` }"
        :title="`Inconclusivo: ${inconclusivePercent}% (${inconclusiveCount} acessos)`"
      />
    </div>

    <TrafficQualityDetails
      :stats="stats"
      :human-percent="humanPercent"
      :bot-percent="botPercent"
      :inconclusive-percent="inconclusivePercent"
      :human-count="humanCount"
      :bot-count="botCount"
      :inconclusive-count="inconclusiveCount"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { ShieldCheck } from 'lucide-vue-next'
import TrafficQualityDetails from '~/components/dashboard/TrafficQualityDetails.vue'
import type { TrafficQualityStats } from '#shared/types'

const props = defineProps<{
  stats?: TrafficQualityStats
}>()

const total = computed(() => props.stats?.total_visitors || 0)
const humanCount = computed(() => props.stats?.human_count || 0)
const botCount = computed(() => props.stats?.bot_count || 0)
const inconclusiveCount = computed(() => props.stats?.inconclusive_count || 0)

const humanPercent = computed(() => (total.value === 0 ? 100 : (props.stats?.human_percent ?? 100)))
const botPercent = computed(() => (total.value === 0 ? 0 : (props.stats?.bot_percent ?? 0)))
const inconclusivePercent = computed(() => (total.value === 0 ? 0 : (props.stats?.inconclusive_percent ?? 0)))

const trafficStatusLabel = computed(() => {
  if (humanPercent.value >= 85) return 'Tráfego Altamente Humano'
  if (humanPercent.value >= 65) return 'Qualidade de Tráfego Boa'
  return 'Atenção: Alto Volume de Bots'
})

const trafficBadgeClass = computed(() => {
  if (humanPercent.value >= 85) return 'status-safe'
  if (humanPercent.value >= 65) return 'status-warning'
  return 'status-alert'
})
</script>

<style scoped>
.traffic-card {
  background: var(--surface-card);
  border: 1px solid var(--border-soft);
  border-radius: 12px;
  padding: 1.125rem 1.25rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
  position: relative;
  overflow: hidden;
}

.traffic-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 0.75rem;
}

.header-info {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.icon-badge {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  background: var(--brand-soft);
  border: 1px solid var(--brand-border);
  display: flex;
  align-items: center;
  justify-content: center;
}

.card-title {
  font-size: var(--fs-lg);
  font-weight: var(--fw-bold);
  color: var(--text-1);
  margin: 0;
  line-height: 1.2;
}

.card-subtitle {
  font-size: var(--fs-sm);
  color: var(--text-2);
  margin: 0.2rem 0 0 0;
}

.trust-pill {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  font-size: var(--fs-xs);
  font-weight: var(--fw-semibold);
  padding: 0.25rem 0.625rem;
  border-radius: 9999px;
  letter-spacing: 0.02em;
}

.trust-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
}

.trust-pill.status-safe {
  background: var(--brand-soft);
  color: var(--brand);
  border: 1px solid var(--brand-border);
}
.trust-pill.status-safe .trust-dot {
  background: var(--brand);
  box-shadow: 0 0 6px var(--brand);
}

.trust-pill.status-warning {
  background: var(--warning-soft);
  color: var(--warning);
  border: 1px solid var(--warning);
}
.trust-pill.status-warning .trust-dot {
  background: var(--warning);
  box-shadow: 0 0 6px var(--warning);
}

.trust-pill.status-alert {
  background: var(--danger-soft);
  color: var(--danger);
  border: 1px solid var(--danger);
}
.trust-pill.status-alert .trust-dot {
  background: var(--danger);
  box-shadow: 0 0 6px var(--danger);
}

.distribution-track {
  width: 100%;
  height: 8px;
  border-radius: 9999px;
  background: var(--surface-soft-2);
  overflow: hidden;
  display: flex;
}

.track-segment {
  height: 100%;
  transition: width 0.4s ease;
}

.segment-human { background: var(--brand); }
.segment-bot { background: var(--danger); }
.segment-inconclusive { background: var(--warning); }
</style>
