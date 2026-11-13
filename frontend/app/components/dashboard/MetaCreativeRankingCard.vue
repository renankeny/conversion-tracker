<template>
  <div class="ranking-card">
    <div class="ranking-header">
      <div>
        <h3 class="ranking-title">Top 3 Criativos — Meta Ads</h3>
        <p class="ranking-sub">Pódio visual com miniatura, posicionamento, leads, vendas e CPL por criativo</p>
      </div>
      <span class="platform-badge badge-meta">
        <MetaAdsIcon :size="12" />
        <span>Meta Ads</span>
      </span>
    </div>

    <div v-if="loading && !topCreatives.length" class="empty-state">
      <div class="loading-spinner" />
      <p>Carregando ranking de criativos da Meta Ads…</p>
    </div>

    <div v-else-if="!topCreatives.length" class="empty-state">
      <Layers :size="20" class="empty-icon" />
      <p>Nenhum anúncio com investimento ativo neste período para a conta Meta Ads vinculada.</p>
    </div>

    <div v-else class="podium-grid">
      <div
        v-for="(item, idx) in topCreatives"
        :key="item.ad_id || idx"
        class="podium-item"
        :class="{ 'is-champion': idx === 0 }"
      >
        <div class="podium-top">
          <span class="medal-pill" :class="`medal-${idx + 1}`">
            <Trophy v-if="idx === 0" :size="11" />
            <span>{{ idx + 1 }}º LUGAR</span>
          </span>
          <span class="placement-pill">{{ item.placement || 'Automático' }}</span>
        </div>

        <div class="media-box">
          <img
            v-if="item.image_url || item.thumbnail_url"
            :src="item.image_url || item.thumbnail_url"
            :alt="item.ad_name"
            class="media-img"
            loading="lazy"
          >
          <div v-else class="media-placeholder">
            <Layers :size="22" />
            <span>Sem prévia</span>
          </div>

          <div class="media-overlay">
            <span class="ov-badge">{{ formatInt(Number(item.leads || 0)) }} leads</span>
            <span v-if="Number(item.purchases || 0) > 0" class="ov-badge ov-sales">
              {{ formatInt(Number(item.purchases || 0)) }} vendas
            </span>
          </div>
        </div>

        <div class="podium-meta">
          <span class="cr-name" :title="item.ad_name">{{ item.ad_name || `Anúncio #${item.ad_id}` }}</span>
          <span class="cr-id">ID: {{ item.ad_id }}</span>
        </div>

        <div class="podium-stats" :class="{ 'is-blurred': blurred }">
          <div class="p-stat">
            <span class="p-lbl">Gasto</span>
            <span class="p-val val-brand blur-target">{{ formatCurrency(Number(item.spend || 0)) }}</span>
          </div>
          <div class="p-stat">
            <span class="p-lbl">CPL</span>
            <span class="p-val blur-target">{{ computeCPL(item) }}</span>
          </div>
          <div class="p-stat">
            <span class="p-lbl">Cliques</span>
            <span class="p-val val-strong">{{ formatInt(Number(item.clicks || 0)) }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Layers, Trophy } from 'lucide-vue-next'
import { MetaAdsIcon } from '~/utils/adIcons'
import { formatMoney } from '~/utils/dashboardCardDetails'
import type { MetaCreativeInsight } from '~/composables/useDashboard'

const props = withDefaults(defineProps<{
  creatives: MetaCreativeInsight[]
  loading?: boolean
  blurred?: boolean
  currency?: string
}>(), {
  currency: 'BRL',
})

const topCreatives = computed(() => (props.creatives || []).slice(0, 3))

const formatInt = (n: number) => new Intl.NumberFormat('pt-BR').format(n || 0)
const formatCurrency = (n: number) => formatMoney(n || 0, props.currency)

const computeCPL = (item: MetaCreativeInsight) => {
  const spend = Number(item.spend || 0)
  const leads = Number(item.leads || 0)
  if (leads <= 0 || spend <= 0) return '—'
  return formatCurrency(spend / leads)
}
</script>

<style scoped>
.ranking-card { background: var(--surface-2); border: 1px solid var(--border); border-radius: 12px; padding: 1.25rem; display: flex; flex-direction: column; gap: 1rem; height: 100%; }
.ranking-header { display: flex; align-items: flex-start; justify-content: space-between; gap: 0.75rem; }
.ranking-title { font-family: var(--font-heading); font-size: var(--fs-md); font-weight: var(--fw-semibold); color: var(--text-strong); margin: 0; }
.ranking-sub { font-size: var(--fs-xs); color: var(--text-3); margin: 0.2rem 0 0; }
.platform-badge { display: inline-flex; align-items: center; gap: 0.3rem; font-size: var(--fs-2xs); font-weight: var(--fw-semibold); padding: 0.2rem 0.55rem; border-radius: 999px; white-space: nowrap; }
.badge-meta { background: var(--brand-soft); color: var(--brand); border: 1px solid var(--brand-border); }
.empty-state { display: flex; flex-direction: column; align-items: center; justify-content: center; text-align: center; gap: 0.5rem; padding: 2.25rem 1rem; color: var(--text-3); font-size: var(--fs-sm); background: var(--surface-deep); border: 1px dashed var(--border); border-radius: 10px; flex: 1; }
.empty-icon { color: var(--text-3); opacity: 0.7; }
.loading-spinner { width: 1.35rem; height: 1.35rem; border: 2px solid var(--border); border-top-color: var(--brand); border-radius: 50%; animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }

.podium-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 0.75rem; }
@media (max-width: 768px) { .podium-grid { grid-template-columns: 1fr; } }

.podium-item { display: flex; flex-direction: column; gap: 0.6rem; padding: 0.75rem; border-radius: 10px; background: var(--surface-deep); border: 1px solid var(--border-soft); transition: border-color 0.18s ease; min-width: 0; }
.podium-item:hover { border-color: var(--border); }
.podium-item.is-champion { border-color: var(--brand-border); }

.podium-top { display: flex; align-items: center; justify-content: space-between; gap: 0.35rem; }
.medal-pill { display: inline-flex; align-items: center; gap: 0.25rem; padding: 0.14rem 0.45rem; border-radius: 6px; font-size: var(--fs-2xs); font-weight: var(--fw-bold); background: var(--surface-2); color: var(--text-2); border: 1px solid var(--border); white-space: nowrap; }
.medal-1 { background: var(--brand-soft); color: var(--brand); border-color: var(--brand-border); }
.placement-pill { font-size: var(--fs-2xs); color: var(--text-3); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; max-width: 55%; }

.media-box { position: relative; width: 100%; aspect-ratio: 1 / 1; border-radius: 8px; overflow: hidden; background: var(--surface-2); border: 1px solid var(--border-soft); }
.media-img { width: 100%; height: 100%; object-fit: cover; display: block; }
.media-placeholder { width: 100%; height: 100%; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 0.3rem; color: var(--text-3); font-size: var(--fs-2xs); }
.media-overlay { position: absolute; left: 6px; right: 6px; bottom: 6px; display: flex; flex-wrap: wrap; gap: 0.3rem; }
.ov-badge { padding: 0.14rem 0.42rem; border-radius: 5px; font-size: var(--fs-2xs); font-weight: var(--fw-semibold); font-family: var(--font-mono); background: var(--surface-deep); color: var(--text-strong); border: 1px solid var(--border); }
.ov-sales { color: var(--brand); border-color: var(--brand-border); }

.podium-meta { display: flex; flex-direction: column; gap: 0.12rem; min-width: 0; }
.cr-name { font-size: var(--fs-xs); font-weight: var(--fw-semibold); color: var(--text-strong); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.cr-id { font-size: var(--fs-2xs); color: var(--text-3); font-family: var(--font-mono); }

.podium-stats { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 0.35rem; padding-top: 0.5rem; border-top: 1px solid var(--border-soft); }
.podium-stats.is-blurred .blur-target { filter: blur(6px); user-select: none; }
.p-stat { display: flex; flex-direction: column; gap: 0.08rem; min-width: 0; }
.p-lbl { font-size: var(--fs-2xs); color: var(--text-3); text-transform: uppercase; }
.p-val { font-size: var(--fs-2xs); font-weight: var(--fw-semibold); color: var(--text-2); font-family: var(--font-mono); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.val-strong { color: var(--text-strong); }
.val-brand { color: var(--brand); }
</style>
