<template>
  <div v-if="user" class="traffic-audit-box" :class="trafficTypeClass">
    <div class="flex items-center justify-between flex-wrap gap-2">
      <div class="flex items-center gap-2.5">
        <div class="audit-icon-wrapper">
          <ShieldCheck v-if="user.traffic_type !== 'bot'" :size="18" />
          <Bot v-else :size="18" />
        </div>
        <div>
          <div class="flex items-center gap-2">
            <span class="audit-badge" :class="trafficTypeClass">{{ trafficTypeLabel }}</span>
            <span v-if="user.traffic_score !== undefined" class="score-text">
              Score: {{ user.traffic_score > 0 ? '+' : '' }}{{ user.traffic_score }}
            </span>
          </div>
          <p class="summary-text">{{ trafficTypeSummary }}</p>
        </div>
      </div>

      <button type="button" class="rules-toggle-btn" @click="showRules = !showRules">
        <span>{{ showRules ? 'Ocultar regras' : 'Ver regras disparadas' }}</span>
        <ChevronDown :size="14" class="transition-transform duration-200" :class="{ 'rotate-180': showRules }" />
      </button>
    </div>

    <div v-show="showRules" class="rules-panel">
      <div v-if="parsedRules.length" class="flex flex-col gap-1.5">
        <div v-for="rule in parsedRules" :key="rule.id" class="rule-card">
          <span
            class="rule-dot"
            :class="{
              'is-human': rule.impact === 'human',
              'is-bot': rule.impact === 'automated',
              'is-neutral': rule.impact === 'neutral'
            }"
          />
          <div class="flex-1 min-w-0">
            <div class="flex items-center justify-between gap-2">
              <span class="rule-label">{{ rule.label }}</span>
              <span class="rule-weight">{{ rule.weight > 0 ? '+' : '' }}{{ rule.weight }}</span>
            </div>
            <p class="rule-desc">{{ rule.description }}</p>
          </div>
        </div>
      </div>
      <p v-else class="empty-rules">Nenhuma regra específica registrada.</p>

      <div v-if="user.referrer" class="referrer-row">
        <strong>Referrer:</strong> <span class="referrer-val">{{ user.referrer }}</span>
      </div>

      <p class="audit-footnote">
        * Indícios técnicos baseados em cabeçalhos HTTP, User-Agent e referrer.
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { ShieldCheck, Bot, ChevronDown } from 'lucide-vue-next'
import type { TrackedUser, TrafficRule } from '#shared/types'

interface Props {
  user: TrackedUser | null
}

const props = defineProps<Props>()
const showRules = ref(false)

const parsedRules = computed<TrafficRule[]>(() => {
  const rules = props.user?.traffic_rules
  if (!rules) return []
  if (Array.isArray(rules)) return rules
  if (typeof rules === 'string') {
    try {
      return JSON.parse(rules)
    } catch {
      return []
    }
  }
  return []
})

const trafficTypeClass = computed(() => {
  const t = props.user?.traffic_type
  if (t === 'bot') return 'is-bot'
  if (t === 'inconclusive') return 'is-inconclusive'
  return 'is-human'
})

const trafficTypeLabel = computed(() => {
  const t = props.user?.traffic_type
  if (t === 'bot') {
    if (props.user?.traffic_category === 'ad_verifier') return 'Crawler de Anúncio'
    if (props.user?.traffic_category === 'search_engine') return 'Crawler de Busca'
    if (props.user?.traffic_category === 'headless_scraper') return 'Automação / Headless'
    return 'Provável Bot'
  }
  if (t === 'inconclusive') return 'Inconclusivo'
  return 'Provavelmente Humano'
})

const trafficTypeSummary = computed(() => {
  const t = props.user?.traffic_type
  if (t === 'bot') return 'Identificados padrões característicos de requisições automatizadas ou robôs de plataformas.'
  if (t === 'inconclusive') return 'Dados de cabeçalho incompletos ou não padronizados para afirmação conclusiva.'
  return 'Navegador e sistema legítimos reconhecidos, sem traços de controle remoto.'
})
</script>

<style scoped>
.traffic-audit-box { border-radius: 8px; padding: 0.75rem 1rem; border: 1px solid var(--border-soft); background: var(--surface-soft-2); }
.traffic-audit-box.is-human { border-color: var(--brand-border); background: var(--brand-soft-2); }
.traffic-audit-box.is-human .audit-icon-wrapper { color: var(--brand); }
.traffic-audit-box.is-bot { border-color: var(--danger-border); background: var(--danger-soft); }
.traffic-audit-box.is-bot .audit-icon-wrapper { color: var(--danger); }
.traffic-audit-box.is-inconclusive { border-color: var(--warning-border); background: var(--warning-soft); }
.traffic-audit-box.is-inconclusive .audit-icon-wrapper { color: var(--warning-light); }
.audit-badge { font-size: var(--fs-xs); font-weight: var(--fw-bold); text-transform: uppercase; letter-spacing: 0.04em; padding: 0.125rem 0.5rem; border-radius: 9999px; }
.audit-badge.is-human { background: var(--brand-soft); color: var(--brand); border: 1px solid var(--brand-border); }
.audit-badge.is-bot { background: var(--danger-soft); color: var(--danger-light); border: 1px solid var(--danger-border); }
.audit-badge.is-inconclusive { background: var(--warning-soft); color: var(--warning-light); border: 1px solid var(--warning-border); }
.score-text { font-size: var(--fs-base); font-family: var(--font-mono); color: var(--text-3); }
.summary-text { font-size: var(--fs-base); color: var(--text-2); margin-top: 0.125rem; }
.rules-toggle-btn { font-size: var(--fs-base); font-weight: var(--fw-medium); color: var(--text-2); display: flex; align-items: center; gap: 0.25rem; background: transparent; border: none; cursor: pointer; }
.rules-panel { margin-top: 0.75rem; padding-top: 0.75rem; border-top: 1px solid var(--border-soft); display: flex; flex-direction: column; gap: 0.5rem; }
.rule-card { display: flex; align-items: flex-start; gap: 0.5rem; padding: 0.5rem; border-radius: 6px; font-size: var(--fs-base); background: var(--surface-deep); border: 1px solid var(--border-soft); }
.rule-dot { width: 8px; height: 8px; border-radius: 9999px; margin-top: 0.25rem; flex-shrink: 0; }
.rule-dot.is-human { background: var(--brand); }
.rule-dot.is-bot { background: var(--danger); }
.rule-dot.is-neutral { background: var(--warning-light); }
.rule-label { font-weight: var(--fw-semibold); color: var(--text-1); }
.rule-weight { font-family: var(--font-mono); font-size: var(--fs-xs); color: var(--text-3); }
.rule-desc { font-size: var(--fs-sm); color: var(--text-2); margin-top: 0.125rem; }
.empty-rules { font-size: var(--fs-base); font-style: italic; color: var(--text-3); }
.referrer-row { font-size: var(--fs-sm); color: var(--text-3); margin-top: 0.25rem; }
.referrer-val { font-family: var(--font-mono); }
.audit-footnote { font-size: var(--fs-xs); margin-top: 0.25rem; font-style: italic; color: var(--text-3); }
</style>
