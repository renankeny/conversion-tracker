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
            <span v-if="user.traffic_score !== undefined" class="text-xs font-mono" style="color: var(--text-3)">
              Score: {{ user.traffic_score > 0 ? '+' : '' }}{{ user.traffic_score }}
            </span>
          </div>
          <p class="text-xs mt-0.5" style="color: var(--text-2)">{{ trafficTypeSummary }}</p>
        </div>
      </div>

      <button
        type="button"
        class="text-xs flex items-center gap-1 font-medium transition-colors"
        style="color: var(--text-2)"
        @click="showRules = !showRules"
      >
        <span>{{ showRules ? 'Ocultar regras' : 'Ver regras disparadas' }}</span>
        <ChevronDown :size="14" class="transition-transform duration-200" :class="{ 'rotate-180': showRules }" />
      </button>
    </div>

    <!-- Lista de Regras Acionadas -->
    <div v-show="showRules" class="mt-3 pt-3 flex flex-col gap-2" style="border-top: 1px solid var(--border-soft)">
      <div v-if="parsedRules.length" class="flex flex-col gap-1.5">
        <div
          v-for="rule in parsedRules"
          :key="rule.id"
          class="flex items-start gap-2 p-2 rounded text-xs"
          style="background: var(--surface-deep); border: 1px solid var(--border-soft)"
        >
          <span
            class="w-2 h-2 rounded-full mt-1 flex-shrink-0"
            :class="{
              'bg-brand': rule.impact === 'human',
              'bg-red-500': rule.impact === 'automated',
              'bg-amber-400': rule.impact === 'neutral'
            }"
          />
          <div class="flex-1 min-w-0">
            <div class="flex items-center justify-between gap-2">
              <span class="font-semibold" style="color: var(--text-1)">{{ rule.label }}</span>
              <span class="font-mono text-[10px]" style="color: var(--text-3)">{{ rule.weight > 0 ? '+' : '' }}{{ rule.weight }}</span>
            </div>
            <p class="text-[11px] mt-0.5" style="color: var(--text-2)">{{ rule.description }}</p>
          </div>
        </div>
      </div>
      <p v-else class="text-xs italic" style="color: var(--text-3)">Nenhuma regra específica registrada.</p>

      <div v-if="user.referrer" class="text-[11px] mt-1" style="color: var(--text-3)">
        <strong>Referrer:</strong> <span class="font-mono">{{ user.referrer }}</span>
      </div>

      <p class="text-[10px] mt-1 italic" style="color: var(--text-3)">
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
.traffic-audit-box {
  border-radius: 8px;
  padding: 0.75rem 1rem;
  border: 1px solid var(--border-soft);
  background: var(--surface-soft-2);
}

.traffic-audit-box.is-human {
  border-color: var(--brand-border);
  background: var(--brand-soft-2);
}
.traffic-audit-box.is-human .audit-icon-wrapper {
  color: var(--brand);
}

.traffic-audit-box.is-bot {
  border-color: rgba(239, 68, 68, 0.25);
  background: rgba(239, 68, 68, 0.04);
}
.traffic-audit-box.is-bot .audit-icon-wrapper {
  color: #ef4444;
}

.traffic-audit-box.is-inconclusive {
  border-color: rgba(251, 191, 36, 0.25);
  background: rgba(251, 191, 36, 0.04);
}
.traffic-audit-box.is-inconclusive .audit-icon-wrapper {
  color: #fbbf24;
}

.audit-badge {
  font-size: var(--fs-xs);
  font-weight: var(--fw-bold);
  text-transform: uppercase;
  letter-spacing: 0.04em;
  padding: 0.125rem 0.5rem;
  border-radius: 9999px;
}
.audit-badge.is-human {
  background: var(--brand-soft);
  color: var(--brand);
  border: 1px solid var(--brand-border);
}
.audit-badge.is-bot {
  background: rgba(239, 68, 68, 0.15);
  color: #f87171;
  border: 1px solid rgba(239, 68, 68, 0.3);
}
.audit-badge.is-inconclusive {
  background: rgba(251, 191, 36, 0.15);
  color: #fbbf24;
  border: 1px solid rgba(251, 191, 36, 0.3);
}
</style>
