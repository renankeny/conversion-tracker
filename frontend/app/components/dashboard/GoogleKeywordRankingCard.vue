<template>
  <div class="ranking-card">
    <div class="ranking-header">
      <div>
        <h3 class="ranking-title">Top 5 Palavras-Chave — Google Ads</h3>
        <p class="ranking-sub">Termos de pesquisa com maior volume de cliques no período selecionado</p>
      </div>
      <span class="platform-badge badge-google">Google Ads</span>
    </div>

    <div v-if="!topKeywords.length" class="empty-state">
      <Search :size="20" class="empty-icon" />
      <p>Nenhuma palavra-chave com cliques registrada neste período para a conta Google Ads vinculada.</p>
    </div>

    <div v-else class="kw-list">
      <div v-for="(item, idx) in topKeywords" :key="`${item.keyword}-${idx}`" class="kw-row">
        <div class="kw-left">
          <span class="kw-rank" :class="{ 'rank-gold': idx === 0 }">#{{ idx + 1 }}</span>
          <div class="kw-info">
            <div class="kw-title-row">
              <span class="kw-text">{{ item.keyword }}</span>
              <span v-if="item.match_type" class="match-pill">{{ formatMatchType(item.match_type) }}</span>
            </div>
            <span v-if="item.campaign_name" class="kw-camp">{{ item.campaign_name }}</span>
          </div>
        </div>

        <div class="kw-metrics">
          <div class="metric-col">
            <span class="metric-lbl">Cliques</span>
            <span class="metric-val val-strong">{{ formatInt(item.clicks) }}</span>
          </div>
          <div class="metric-col hide-xs">
            <span class="metric-lbl">Impr.</span>
            <span class="metric-val">{{ formatInt(item.impressions) }}</span>
          </div>
          <div class="metric-col hide-xs">
            <span class="metric-lbl">CPC</span>
            <span class="metric-val">{{ formatBRL(item.cpc) }}</span>
          </div>
          <div class="metric-col">
            <span class="metric-lbl">Gasto</span>
            <span class="metric-val val-brand">{{ formatBRL(item.spend) }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Search } from 'lucide-vue-next'
import type { GoogleKeywordInsight } from '#shared/types'

const props = defineProps<{
  keywords: GoogleKeywordInsight[]
  loading?: boolean
}>()

const topKeywords = computed(() => (props.keywords || []).slice(0, 5))

const formatInt = (n: number) => new Intl.NumberFormat('pt-BR').format(n || 0)
const formatBRL = (n: number) =>
  new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(n || 0)

const formatMatchType = (mt: string) => {
  switch (mt?.toUpperCase()) {
    case 'EXACT': return 'Exata'
    case 'PHRASE': return 'Frase'
    case 'BROAD': return 'Ampla'
    default: return mt
  }
}
</script>

<style scoped>
.ranking-card { background: var(--surface-2); border: 1px solid var(--border); border-radius: 12px; padding: 1.25rem; display: flex; flex-direction: column; gap: 1rem; height: 100%; }
.ranking-header { display: flex; align-items: flex-start; justify-content: space-between; gap: 0.75rem; }
.ranking-title { font-family: var(--font-heading); font-size: var(--fs-md); font-weight: var(--fw-semibold); color: var(--text-strong); margin: 0; }
.ranking-sub { font-size: var(--fs-xs); color: var(--text-3); margin: 0.2rem 0 0; }
.platform-badge { font-size: var(--fs-2xs); font-weight: var(--fw-semibold); padding: 0.2rem 0.55rem; border-radius: 999px; white-space: nowrap; }
.badge-google { background: var(--warning-soft); color: var(--warning); border: 1px solid var(--warning); }
.empty-state { display: flex; flex-direction: column; align-items: center; justify-content: center; text-align: center; gap: 0.5rem; padding: 2rem 1rem; color: var(--text-3); font-size: var(--fs-sm); background: var(--surface-deep); border: 1px dashed var(--border-soft); border-radius: 10px; }
.empty-icon { color: var(--text-3); opacity: 0.7; }
.kw-list { display: flex; flex-direction: column; gap: 0.5rem; }
.kw-row { display: flex; align-items: center; justify-content: space-between; gap: 0.75rem; padding: 0.65rem 0.85rem; background: var(--surface-deep); border: 1px solid var(--border-soft); border-radius: 9px; transition: border-color 0.15s ease; }
.kw-row:hover { border-color: var(--border); }
.kw-left { display: flex; align-items: center; gap: 0.65rem; min-width: 0; flex: 1; }
.kw-rank { width: 26px; height: 26px; border-radius: 6px; background: var(--surface-2); border: 1px solid var(--border); display: inline-flex; align-items: center; justify-content: center; font-size: var(--fs-2xs); font-weight: var(--fw-bold); color: var(--text-2); flex-shrink: 0; }
.rank-gold { background: var(--brand-soft); border-color: var(--brand-border); color: var(--brand); }
.kw-info { min-width: 0; display: flex; flex-direction: column; gap: 0.12rem; }
.kw-title-row { display: flex; align-items: center; gap: 0.45rem; flex-wrap: wrap; }
.kw-text { font-size: var(--fs-sm); font-weight: var(--fw-semibold); color: var(--text-strong); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.match-pill { font-size: var(--fs-2xs); color: var(--text-3); background: var(--surface-2); border: 1px solid var(--border-soft); padding: 0.08rem 0.38rem; border-radius: 4px; }
.kw-camp { font-size: var(--fs-2xs); color: var(--text-3); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.kw-metrics { display: flex; align-items: center; gap: 1rem; flex-shrink: 0; }
.metric-col { display: flex; flex-direction: column; align-items: flex-end; gap: 0.1rem; }
.metric-lbl { font-size: var(--fs-2xs); color: var(--text-3); text-transform: uppercase; letter-spacing: 0.03em; }
.metric-val { font-size: var(--fs-xs); font-weight: var(--fw-semibold); color: var(--text-2); font-family: var(--font-mono); }
.val-strong { color: var(--text-strong); }
.val-brand { color: var(--brand); }
@media (max-width: 640px) { .hide-xs { display: none; } }
</style>
