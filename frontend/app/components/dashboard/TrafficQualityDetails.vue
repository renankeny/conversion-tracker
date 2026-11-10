<template>
  <div class="traffic-details">
    <!-- Estatísticas Principais (Grid 3 colunas) -->
    <div class="stats-grid">
      <div class="stat-box human">
        <div class="stat-top">
          <span class="indicator-dot dot-human" />
          <span class="stat-label">Provavelmente Humano</span>
        </div>
        <div class="stat-value">{{ humanPercent }}%</div>
        <div class="stat-count">{{ humanCount.toLocaleString('pt-BR') }} visitantes</div>
      </div>

      <div class="stat-box bot">
        <div class="stat-top">
          <span class="indicator-dot dot-bot" />
          <span class="stat-label">Bots & Crawlers</span>
        </div>
        <div class="stat-value">{{ botPercent }}%</div>
        <div class="stat-count">{{ botCount.toLocaleString('pt-BR') }} acessos</div>
      </div>

      <div class="stat-box inconclusive">
        <div class="stat-top">
          <span class="indicator-dot dot-inconclusive" />
          <span class="stat-label">Inconclusivo</span>
        </div>
        <div class="stat-value">{{ inconclusivePercent }}%</div>
        <div class="stat-count">{{ inconclusiveCount.toLocaleString('pt-BR') }} acessos</div>
      </div>
    </div>

    <!-- Detalhamento de Bots Conhecidos (se houver) -->
    <div v-if="hasBotCategories" class="bot-breakdown">
      <div class="breakdown-title">Tipos de Acessos Automatizados Detectados:</div>
      <div class="breakdown-chips">
        <div v-if="adVerifierCount > 0" class="chip chip-ad">
          <Eye :size="12" />
          <span>Verificação de Anúncios (Meta/Google Ads): <strong>{{ adVerifierCount }}</strong></span>
        </div>
        <div v-if="searchEngineCount > 0" class="chip chip-search">
          <Search :size="12" />
          <span>Motores de Busca (Googlebot/Bing): <strong>{{ searchEngineCount }}</strong></span>
        </div>
        <div v-if="scraperCount > 0" class="chip chip-scraper">
          <Bot :size="12" />
          <span>Headless / Automação / Scrapers: <strong>{{ scraperCount }}</strong></span>
        </div>
        <div v-if="genericBotCount > 0" class="chip chip-generic">
          <Cpu :size="12" />
          <span>Robôs Genéricos: <strong>{{ genericBotCount }}</strong></span>
        </div>
      </div>
    </div>

    <!-- Rodapé com Aviso de Indício Heurístico -->
    <div class="traffic-footer">
      <Info :size="13" class="info-icon" style="color: var(--text-3)" />
      <span class="disclaimer-text">
        Classificação probabilística baseada em cabeçalhos HTTP, User-Agent e referrer. Indícios técnicos, não confirmação definitiva.
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Bot, Eye, Search, Cpu, Info } from 'lucide-vue-next'
import type { TrafficQualityStats } from '#shared/types'

const props = defineProps<{
  stats?: TrafficQualityStats
  humanPercent: number
  botPercent: number
  inconclusivePercent: number
  humanCount: number
  botCount: number
  inconclusiveCount: number
}>()

const categories = computed(() => props.stats?.bot_categories || {})
const adVerifierCount = computed(() => categories.value['ad_verifier'] || 0)
const searchEngineCount = computed(() => categories.value['search_engine'] || 0)
const scraperCount = computed(() => categories.value['headless_scraper'] || 0)
const genericBotCount = computed(() => categories.value['generic_bot'] || 0)

const hasBotCategories = computed(() => {
  return adVerifierCount.value > 0 || searchEngineCount.value > 0 || scraperCount.value > 0 || genericBotCount.value > 0
})
</script>

<style scoped>
.traffic-details {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.75rem;
}

@media (max-width: 640px) {
  .stats-grid {
    grid-template-columns: 1fr;
  }
}

.stat-box {
  background: var(--surface-soft);
  border: 1px solid var(--border-soft);
  border-radius: 8px;
  padding: 0.75rem;
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.stat-top {
  display: flex;
  align-items: center;
  gap: 0.375rem;
}

.indicator-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
}

.dot-human { background: var(--brand); }
.dot-bot { background: var(--danger); }
.dot-inconclusive { background: var(--warning); }

.stat-label {
  font-size: var(--fs-xs);
  color: var(--text-2);
  font-weight: var(--fw-medium);
}

.stat-value {
  font-size: var(--fs-2xl);
  font-weight: var(--fw-bold);
  color: var(--text-1);
  letter-spacing: -0.02em;
}

.stat-count {
  font-size: var(--fs-xs);
  color: var(--text-2);
}

.bot-breakdown {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  padding-top: 0.25rem;
}

.breakdown-title {
  font-size: var(--fs-xs);
  font-weight: var(--fw-semibold);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--text-2);
}

.breakdown-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.chip {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  font-size: var(--fs-xs);
  padding: 0.25rem 0.5rem;
  border-radius: 6px;
  background: var(--surface-soft-2);
  border: 1px solid var(--border-soft);
  color: var(--text-2);
}

.chip strong { color: var(--text-1); }
.chip-ad { border-color: rgba(56, 189, 248, 0.25); color: var(--info); }
.chip-search { border-color: rgba(168, 85, 247, 0.25); color: var(--brand); }
.chip-scraper { border-color: rgba(239, 68, 68, 0.25); color: var(--danger); }
.chip-generic { border-color: rgba(251, 191, 36, 0.25); color: var(--warning); }

.traffic-footer {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  font-size: var(--fs-xs);
  color: var(--text-2);
  opacity: 0.8;
  border-top: 1px solid var(--border-soft);
  padding-top: 0.625rem;
}

.disclaimer-text {
  line-height: 1.3;
}
</style>
