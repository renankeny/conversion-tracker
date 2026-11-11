<template>
  <div class="ranking-card">
    <div class="ranking-header">
      <div>
        <h3 class="ranking-title">Top 3 Criativos — Meta Ads</h3>
        <p class="ranking-sub">Miniatura, posicionamento, leads, vendas e investimento por criativo</p>
      </div>
      <span class="platform-badge badge-meta">Meta Ads</span>
    </div>

    <div v-if="!topCreatives.length" class="empty-state">
      <Layers :size="20" class="empty-icon" />
      <p>Nenhum anúncio com investimento ativo neste período para a conta Meta Ads vinculada.</p>
    </div>

    <div v-else class="cr-list">
      <div v-for="(item, idx) in topCreatives" :key="item.ad_id || idx" class="cr-row">
        <div class="cr-left">
          <span class="cr-rank" :class="{ 'rank-gold': idx === 0 }">#{{ idx + 1 }}</span>
          <img
            v-if="item.thumbnail_url || item.image_url"
            :src="item.thumbnail_url || item.image_url"
            :alt="item.ad_name"
            class="cr-thumb"
          >
          <div v-else class="cr-thumb-placeholder">
            <Layers :size="16" />
          </div>
          <div class="cr-info">
            <span class="cr-name" :title="item.ad_name">{{ item.ad_name || `Anúncio #${item.ad_id}` }}</span>
            <div class="cr-sub-row">
              <span class="placement-pill">{{ item.placement || 'Posicionamento Automático' }}</span>
              <span class="cr-id">ID: {{ item.ad_id }}</span>
            </div>
          </div>
        </div>

        <div class="cr-metrics">
          <div class="metric-col">
            <span class="metric-lbl">Leads</span>
            <span class="metric-val val-strong">{{ formatInt(Number(item.leads || 0)) }}</span>
          </div>
          <div class="metric-col">
            <span class="metric-lbl">Vendas</span>
            <span class="metric-val val-success">{{ formatInt(Number(item.purchases || 0)) }}</span>
          </div>
          <div class="metric-col hide-xs">
            <span class="metric-lbl">Gasto</span>
            <span class="metric-val val-brand">{{ formatBRL(Number(item.spend || 0)) }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Layers } from 'lucide-vue-next'
import type { MetaCreativeInsight } from '~/composables/useDashboard'

const props = defineProps<{
  creatives: MetaCreativeInsight[]
  loading?: boolean
}>()

const topCreatives = computed(() => (props.creatives || []).slice(0, 3))

const formatInt = (n: number) => new Intl.NumberFormat('pt-BR').format(n || 0)
const formatBRL = (n: number) =>
  new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(n || 0)
</script>

<style scoped>
.ranking-card { background: var(--surface-2); border: 1px solid var(--border); border-radius: 12px; padding: 1.25rem; display: flex; flex-direction: column; gap: 1rem; height: 100%; }
.ranking-header { display: flex; align-items: flex-start; justify-content: space-between; gap: 0.75rem; }
.ranking-title { font-family: var(--font-heading); font-size: var(--fs-md); font-weight: var(--fw-semibold); color: var(--text-strong); margin: 0; }
.ranking-sub { font-size: var(--fs-xs); color: var(--text-3); margin: 0.2rem 0 0; }
.platform-badge { font-size: var(--fs-2xs); font-weight: var(--fw-semibold); padding: 0.2rem 0.55rem; border-radius: 999px; white-space: nowrap; }
.badge-meta { background: var(--info-soft); color: var(--info); border: 1px solid var(--info); }
.empty-state { display: flex; flex-direction: column; align-items: center; justify-content: center; text-align: center; gap: 0.5rem; padding: 2rem 1rem; color: var(--text-3); font-size: var(--fs-sm); background: var(--surface-deep); border: 1px dashed var(--border-soft); border-radius: 10px; }
.empty-icon { color: var(--text-3); opacity: 0.7; }
.cr-list { display: flex; flex-direction: column; gap: 0.6rem; }
.cr-row { display: flex; align-items: center; justify-content: space-between; gap: 0.85rem; padding: 0.65rem 0.85rem; background: var(--surface-deep); border: 1px solid var(--border-soft); border-radius: 10px; transition: border-color 0.15s ease; }
.cr-row:hover { border-color: var(--border); }
.cr-left { display: flex; align-items: center; gap: 0.7rem; min-width: 0; flex: 1; }
.cr-rank { width: 26px; height: 26px; border-radius: 6px; background: var(--surface-2); border: 1px solid var(--border); display: inline-flex; align-items: center; justify-content: center; font-size: var(--fs-2xs); font-weight: var(--fw-bold); color: var(--text-2); flex-shrink: 0; }
.rank-gold { background: var(--brand-soft); border-color: var(--brand-border); color: var(--brand); }
.cr-thumb, .cr-thumb-placeholder { width: 46px; height: 46px; border-radius: 8px; object-fit: cover; flex-shrink: 0; border: 1px solid var(--border); background: var(--surface-2); display: flex; align-items: center; justify-content: center; color: var(--text-3); }
.cr-info { min-width: 0; display: flex; flex-direction: column; gap: 0.22rem; }
.cr-name { font-size: var(--fs-sm); font-weight: var(--fw-semibold); color: var(--text-strong); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.cr-sub-row { display: flex; align-items: center; gap: 0.45rem; flex-wrap: wrap; }
.placement-pill { font-size: var(--fs-2xs); font-weight: var(--fw-medium); color: var(--info); background: var(--info-soft); padding: 0.1rem 0.45rem; border-radius: 5px; white-space: nowrap; }
.cr-id { font-size: var(--fs-2xs); color: var(--text-3); font-family: var(--font-mono); }
.cr-metrics { display: flex; align-items: center; gap: 1.1rem; flex-shrink: 0; }
.metric-col { display: flex; flex-direction: column; align-items: flex-end; gap: 0.1rem; }
.metric-lbl { font-size: var(--fs-2xs); color: var(--text-3); text-transform: uppercase; letter-spacing: 0.03em; }
.metric-val { font-size: var(--fs-xs); font-weight: var(--fw-semibold); color: var(--text-2); font-family: var(--font-mono); }
.val-strong { color: var(--text-strong); }
.val-success { color: var(--success); }
.val-brand { color: var(--brand); }
@media (max-width: 640px) { .hide-xs { display: none; } }
</style>
