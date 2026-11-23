<template>
  <div v-if="overview && overview.total_links > 0" class="links-card">
    <div class="card-header">
      <div class="header-left">
        <div class="icon-badge">
          <Link2 :size="16" class="icon-brand" />
        </div>
        <div>
          <h3 class="card-title">Links Rastreáveis de WhatsApp</h3>
          <p class="card-subtitle">Desempenho de links de Bio, Stories e campanhas diretas no período</p>
        </div>
      </div>

      <span class="count-pill">
        {{ overview.total_links }} {{ overview.total_links === 1 ? 'link ativo' : 'links' }}
      </span>
    </div>

    <div class="kpi-strip">
      <div class="kpi-box">
        <span class="kpi-label">Cliques no Link</span>
        <strong class="kpi-val">{{ overview.total_clicks }}</strong>
      </div>
      <div class="kpi-box">
        <span class="kpi-label">Conversas Geradas</span>
        <div class="kpi-row">
          <strong class="kpi-val">{{ overview.total_chats }}</strong>
          <span v-if="overview.total_clicks > 0" class="kpi-badge">{{ totalChatRate }}%</span>
        </div>
      </div>
      <div class="kpi-box">
        <span class="kpi-label">Vendas Atribuídas</span>
        <div class="kpi-row">
          <strong class="kpi-val success">{{ overview.total_sales }}</strong>
          <span v-if="overview.total_chats > 0" class="kpi-badge success">{{ totalSaleRate }}%</span>
        </div>
      </div>
      <div class="kpi-box">
        <span class="kpi-label">Receita Gerada</span>
        <strong class="kpi-val" :class="{ 'is-blurred': blurred }">{{ formatMoney(overview.total_revenue) }}</strong>
      </div>
    </div>

    <div class="rows-list">
      <div v-for="item in overview.links" :key="item.id" class="item-row">
        <div class="item-main">
          <div class="item-title-line">
            <span class="item-name">{{ item.name }}</span>
            <code class="item-slug">/w/{{ item.slug }}</code>
            <button
              v-if="!publicMode && item.public_url"
              type="button"
              class="copy-mini"
              title="Copiar URL do link"
              @click="copyUrl(item.id, item.public_url)"
            >
              <Check v-if="copiedId === item.id" :size="12" />
              <Copy v-else :size="12" />
            </button>
          </div>

          <div class="item-channels">
            <template v-if="item.send_events_enabled">
              <BrandIcon v-if="item.send_meta" channel="meta" size="sm" colored />
              <BrandIcon v-if="item.send_tiktok" channel="tiktok" size="sm" colored />
              <BrandIcon v-if="item.send_google_ads" channel="google_ads" size="sm" colored />
              <BrandIcon v-if="item.send_ga4" channel="ga4" size="sm" colored />
              <span class="item-meta-txt">{{ item.event_name }} ({{ item.event_trigger_mode === 'on_click' ? 'no clique' : '1ª msg' }})</span>
            </template>
            <span v-else class="item-meta-txt">Rastreio interno</span>
          </div>
        </div>

        <div class="item-stats">
          <div class="stat-col">
            <span class="stat-lbl">Cliques</span>
            <span class="stat-num">{{ item.clicks_count }}</span>
          </div>
          <div class="stat-col">
            <span class="stat-lbl">Conversas</span>
            <span class="stat-num">{{ item.chats_count }}</span>
          </div>
          <div class="stat-col">
            <span class="stat-lbl">Vendas</span>
            <span class="stat-num success">{{ item.sales_count }}</span>
          </div>
          <div class="stat-col rev-col">
            <span class="stat-lbl">Receita</span>
            <span class="stat-num" :class="{ 'is-blurred': blurred }">{{ formatMoney(item.sales_revenue) }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Link2, Copy, Check } from 'lucide-vue-next'
import BrandIcon from '~/components/events/BrandIcon.vue'
import type { TrackableLinksOverview } from '#shared/types'

const props = withDefaults(defineProps<{
  overview?: TrackableLinksOverview | null
  currency?: string
  blurred?: boolean
  publicMode?: boolean
}>(), {
  overview: null,
  currency: 'BRL',
  blurred: false,
  publicMode: false,
})

const toast = useToast()
const copiedId = ref<number | null>(null)

const totalChatRate = computed(() => {
  const clk = props.overview?.total_clicks || 0
  if (!clk) return 0
  return Math.min(100, Math.round(((props.overview?.total_chats || 0) / clk) * 100))
})

const totalSaleRate = computed(() => {
  const chats = props.overview?.total_chats || 0
  if (!chats) return 0
  return Math.min(100, Math.round(((props.overview?.total_sales || 0) / chats) * 100))
})

const formatMoney = (val: number): string => {
  try {
    return new Intl.NumberFormat('pt-BR', { style: 'currency', currency: props.currency || 'BRL' }).format(val || 0)
  } catch {
    return `R$ ${(val || 0).toFixed(2)}`
  }
}

const copyUrl = async (id: number, url: string) => {
  try {
    await navigator.clipboard.writeText(url)
    copiedId.value = id
    toast.success('Link copiado!', url)
    setTimeout(() => { if (copiedId.value === id) copiedId.value = null }, 2000)
  } catch {
    toast.error('Erro', 'Não foi possível copiar o link.')
  }
}
</script>

<style scoped>
.links-card { background: var(--surface-card); border: 1px solid var(--border-soft); border-radius: 12px; padding: 1.125rem 1.25rem; display: flex; flex-direction: column; gap: 1rem; }
.card-header { display: flex; align-items: center; justify-content: space-between; gap: 0.75rem; flex-wrap: wrap; }
.header-left { display: flex; align-items: center; gap: 0.75rem; }
.icon-badge { width: 32px; height: 32px; border-radius: 8px; background: var(--brand-soft); border: 1px solid var(--brand-border); display: flex; align-items: center; justify-content: center; }
.icon-brand { color: var(--brand); }
.card-title { font-size: var(--fs-lg); font-weight: var(--fw-bold); color: var(--text-1); margin: 0; line-height: 1.2; }
.card-subtitle { font-size: var(--fs-sm); color: var(--text-2); margin: 0.2rem 0 0; }
.count-pill { font-size: var(--fs-xs); font-weight: var(--fw-semibold); padding: 0.25rem 0.625rem; border-radius: 9999px; background: var(--brand-soft); color: var(--brand); border: 1px solid var(--brand-border); }
.kpi-strip { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 0.75rem; }
.kpi-box { background: var(--surface-deep); border: 1px solid var(--border-soft); border-radius: 9px; padding: 0.65rem 0.85rem; display: flex; flex-direction: column; gap: 0.2rem; }
.kpi-label { font-size: var(--fs-xs); color: var(--text-3); text-transform: uppercase; letter-spacing: 0.04em; }
.kpi-row { display: flex; align-items: baseline; gap: 0.4rem; }
.kpi-val { font-size: var(--fs-xl); font-weight: var(--fw-bold); color: var(--text-strong); }
.kpi-val.success { color: var(--success); }
.kpi-badge { font-size: var(--fs-xs); font-weight: var(--fw-semibold); color: var(--brand); background: var(--brand-soft); padding: 0.0625rem 0.35rem; border-radius: 4px; }
.kpi-badge.success { color: var(--success); background: var(--success-soft); }
.rows-list { display: flex; flex-direction: column; gap: 0.5rem; }
.item-row { display: flex; align-items: center; justify-content: space-between; gap: 0.75rem; padding: 0.65rem 0.85rem; background: var(--surface-2); border: 1px solid var(--border-faint); border-radius: 8px; flex-wrap: wrap; }
.item-main { display: flex; flex-direction: column; gap: 0.25rem; min-width: 200px; flex: 1; }
.item-title-line { display: flex; align-items: center; gap: 0.45rem; flex-wrap: wrap; }
.item-name { font-size: var(--fs-md); font-weight: var(--fw-semibold); color: var(--text-strong); }
.item-slug { font-family: var(--font-mono); font-size: var(--fs-xs); color: var(--brand); background: var(--surface-soft); padding: 0.0625rem 0.375rem; border-radius: 4px; }
.copy-mini { display: inline-flex; align-items: center; justify-content: center; width: 22px; height: 22px; border-radius: 5px; border: 1px solid var(--border); background: transparent; color: var(--text-2); cursor: pointer; }
.copy-mini:hover { border-color: var(--brand-border); color: var(--brand); }
.item-channels { display: flex; align-items: center; gap: 0.35rem; }
.item-meta-txt { font-size: var(--fs-xs); color: var(--text-3); }
.item-stats { display: flex; align-items: center; gap: 1.25rem; flex-wrap: wrap; }
.stat-col { display: flex; flex-direction: column; align-items: flex-end; min-width: 54px; }
.rev-col { min-width: 88px; }
.stat-lbl { font-size: var(--fs-2xs); color: var(--text-3); text-transform: uppercase; letter-spacing: 0.04em; }
.stat-num { font-size: var(--fs-md); font-weight: var(--fw-bold); color: var(--text-1); }
.stat-num.success { color: var(--success); }
.is-blurred { filter: blur(6px); user-select: none; }
@media (max-width: 768px) { .kpi-strip { grid-template-columns: repeat(2, minmax(0, 1fr)); } .stat-col { align-items: flex-start; } }
</style>
