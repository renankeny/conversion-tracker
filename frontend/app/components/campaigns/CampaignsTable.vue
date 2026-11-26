<template>
  <section class="table-card">
    <div v-if="loading && !rows.length" class="skeleton-wrap">
      <div v-for="n in 6" :key="n" class="skeleton-row" />
    </div>

    <div v-else-if="!rows.length" class="empty-state">
      <Megaphone :size="28" class="empty-icon" />
      <p class="empty-title">Nenhuma linha encontrada neste recorte</p>
      <p class="empty-sub">
        Conecte uma conta Meta Ads ou Google Ads nas integrações, ou ajuste o período e filtros acima.
      </p>
    </div>

    <div v-else class="table-scroll">
      <table class="campaigns-table">
        <thead>
          <tr>
            <th v-if="showSelection" class="col-check" />
            <th class="col-status">Status</th>
            <th class="col-name">{{ nameColumnLabel }}</th>
            <th v-if="activeTab !== 'ads'" class="col-num">Orçamento/dia</th>
            <th class="col-num">Gasto</th>
            <th class="col-num">Cliques / CTR</th>
            <th class="col-num">Conv. / IA</th>
            <th class="col-num">Vendas</th>
            <th class="col-num">Estornos</th>
            <th class="col-num">Rec. Líquida</th>
            <th class="col-num">CPA Real</th>
            <th class="col-num">ROAS Líq.</th>
            <th class="col-num">Lucro Líq.</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="row in rows"
            :key="`${row.platform}:${row.level}:${row.id}`"
            :class="{
              'is-unattributed': row.platform === 'unattributed',
              'is-selected': selectedIds.includes(row.id),
            }"
          >
            <td v-if="showSelection" class="col-check">
              <BaseCheckbox
                v-if="row.platform !== 'unattributed'"
                size="sm"
                :checked="selectedIds.includes(row.id)"
                @change="emit('toggle-select', row.id)"
              />
            </td>

            <td class="col-status">
              <CampaignStatusSwitch
                v-if="row.platform !== 'unattributed'"
                :active="row.status === 'ACTIVE'"
                :disabled="!row.can_edit_status"
                :loading="Boolean(mutatingIds[`${row.platform}:${row.level}:${row.id}`])"
                @toggle="emit('toggle-status', row)"
              />
              <span v-else class="unattr-dot" title="Conciliação orgânica/sem UTM" />
            </td>

            <td class="col-name">
              <div class="name-cell">
                <img
                  v-if="activeTab === 'ads' && row.thumbnail_url"
                  :src="row.thumbnail_url"
                  :alt="row.name"
                  referrerpolicy="no-referrer"
                  class="creative-thumb"
                />
                <span v-else-if="row.platform !== 'unattributed'" class="plat-icon">
                  <MetaAdsIcon v-if="row.platform === 'meta'" :size="15" />
                  <GoogleAdsIcon v-else :size="15" />
                </span>

                <div class="name-texts">
                  <button
                    v-if="activeTab !== 'ads' && row.platform !== 'unattributed'"
                    type="button"
                    class="row-link"
                    @click="emit('drill-down', row)"
                  >
                    <span class="row-title">{{ row.name }}</span>
                    <ChevronRight :size="13" class="drill-icon" />
                  </button>
                  <span v-else class="row-title">{{ row.name }}</span>

                  <span class="row-sub">
                    <template v-if="activeTab === 'adsets' && row.campaign_name">
                      {{ row.campaign_name }}
                      <span v-if="row.entity_subtype === 'asset_group'" class="subtype-pill">PMax</span>
                    </template>
                    <template v-else-if="activeTab === 'ads' && row.adset_name">
                      {{ row.campaign_name }} · {{ row.adset_name }}
                    </template>
                    <template v-else-if="row.platform === 'unattributed'">
                      Vendas diretas, orgânicas ou sem ID/UTM vinculado
                    </template>
                    <template v-else>
                      {{ row.channel_type || row.objective || (row.platform === 'meta' ? 'Meta Ads v26.0' : 'Google Ads v26') }}
                    </template>
                  </span>
                </div>
              </div>
            </td>

            <td v-if="activeTab !== 'ads'" class="col-num">
              <div class="budget-cell">
                <span>{{ row.daily_budget > 0 ? formatCampaignMoney(row.daily_budget, row.currency || currency) : '—' }}</span>
                <button
                  v-if="row.can_edit_budget"
                  type="button"
                  class="edit-budget-btn"
                  title="Editar orçamento diário"
                  @click.stop="emit('edit-budget', row)"
                >
                  <Pencil :size="12" />
                </button>
              </div>
            </td>

            <td class="col-num mono">{{ formatCampaignMoney(row.spend, row.currency || currency) }}</td>

            <td class="col-num">
              <div class="stacked-num">
                <span>{{ formatCompactInt(row.clicks) }}</span>
                <span class="sub-metric">{{ formatCampaignPct(row.ctr) }}</span>
              </div>
            </td>

            <td class="col-num">
              <div class="stacked-num">
                <span>{{ formatCompactInt(row.conversations || row.leads) }}</span>
                <span v-if="row.ai_qualified > 0" class="sub-metric text-brand">{{ row.ai_qualified }} IA</span>
                <span v-else-if="row.cost_per_lead > 0" class="sub-metric">{{ formatCampaignMoney(row.cost_per_lead, currency) }}</span>
              </div>
            </td>

            <td class="col-num">
              <div class="stacked-num">
                <strong>{{ formatCompactInt(row.net_sales) }}</strong>
                <span v-if="row.gross_sales !== row.net_sales" class="sub-metric">Bruto: {{ row.gross_sales }}</span>
              </div>
            </td>

            <td class="col-num">
              <span v-if="row.refund_count + row.chargeback_count > 0" class="refund-cell-pill">
                {{ row.refund_count + row.chargeback_count }} ({{ formatCampaignPct(row.refund_rate) }})
              </span>
              <span v-else class="muted-dash">—</span>
            </td>

            <td class="col-num mono">{{ formatCampaignMoney(row.net_revenue, currency) }}</td>
            <td class="col-num mono">{{ row.real_cpa > 0 ? formatCampaignMoney(row.real_cpa, currency) : '—' }}</td>
            <td class="col-num mono" :class="{ 'text-ok': row.net_roas >= 1.5 }">
              {{ row.spend > 0 ? formatRoas(row.net_roas) : '—' }}
            </td>
            <td class="col-num mono" :class="row.net_profit >= 0 ? 'text-ok' : 'text-danger'">
              <strong>{{ formatCampaignMoney(row.net_profit, currency) }}</strong>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ChevronRight, Megaphone, Pencil } from 'lucide-vue-next'
import type { CampaignRow } from '#shared/types'
import type { HierarchyTabKey } from './CampaignsHierarchyTabs.vue'
import CampaignStatusSwitch from './CampaignStatusSwitch.vue'
import { MetaAdsIcon, GoogleAdsIcon } from '~/utils/adIcons'
import { formatCampaignMoney, formatCampaignPct, formatCompactInt, formatRoas } from '~/utils/campaignFormatters'

const props = defineProps<{
  rows: CampaignRow[]
  activeTab: HierarchyTabKey
  currency: string
  loading: boolean
  selectedIds: string[]
  mutatingIds: Record<string, boolean>
}>()

const emit = defineEmits<{
  'toggle-select': [id: string]
  'toggle-status': [row: CampaignRow]
  'edit-budget': [row: CampaignRow]
  'drill-down': [row: CampaignRow]
}>()

const showSelection = computed(() => props.activeTab !== 'ads')
const nameColumnLabel = computed(() => (
  props.activeTab === 'adsets' ? 'Conjunto / Grupo' : props.activeTab === 'ads' ? 'Anúncio / Criativo' : 'Campanha'
))
</script>

<style scoped>
.table-card { background: var(--surface-2); border: 1px solid var(--border); border-radius: 12px; overflow: hidden; }
.table-scroll { overflow-x: auto; }
.campaigns-table { width: 100%; border-collapse: collapse; min-width: 1060px; }
.campaigns-table th {
  padding: 0.7rem 0.75rem; text-align: left; font-size: var(--fs-xs); font-weight: var(--fw-semibold);
  color: var(--text-3); text-transform: uppercase; letter-spacing: 0.04em;
  border-bottom: 1px solid var(--border); background: var(--surface-1); white-space: nowrap;
}
.campaigns-table td {
  padding: 0.68rem 0.75rem; border-bottom: 1px solid var(--border-soft);
  font-size: var(--fs-base); color: var(--text-1); vertical-align: middle;
}
.campaigns-table tbody tr:hover { background: var(--surface-soft); }
.campaigns-table tbody tr.is-selected { background: var(--brand-soft-2); }
.campaigns-table tbody tr.is-unattributed { background: var(--surface-soft-2); }
.col-check { width: 38px; text-align: center; }
.col-status { width: 58px; }
.col-num { text-align: right !important; white-space: nowrap; }
.mono { font-family: var(--font-mono); }
.name-cell { display: flex; align-items: center; gap: 0.6rem; min-width: 240px; max-width: 340px; }
.creative-thumb { width: 34px; height: 34px; border-radius: 7px; object-fit: cover; flex-shrink: 0; border: 1px solid var(--border); }
.plat-icon { width: 28px; height: 28px; border-radius: 7px; background: var(--surface-1); border: 1px solid var(--border); display: inline-flex; align-items: center; justify-content: center; flex-shrink: 0; }
.name-texts { display: flex; flex-direction: column; min-width: 0; gap: 0.12rem; }
.row-link { display: inline-flex; align-items: center; gap: 0.25rem; background: none; border: none; padding: 0; color: var(--text-strong); cursor: pointer; text-align: left; min-width: 0; }
.row-link:hover .row-title { color: var(--brand); }
.row-title { font-size: var(--fs-md); font-weight: var(--fw-semibold); color: var(--text-strong); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.drill-icon { color: var(--text-3); flex-shrink: 0; }
.row-sub { font-size: var(--fs-xs); color: var(--text-3); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.subtype-pill { margin-left: 0.3rem; padding: 0.05rem 0.35rem; border-radius: 4px; background: var(--brand-soft); color: var(--brand); font-size: var(--fs-2xs); font-weight: var(--fw-semibold); }
.budget-cell { display: inline-flex; align-items: center; justify-content: flex-end; gap: 0.35rem; font-family: var(--font-mono); }
.edit-budget-btn { width: 22px; height: 22px; border-radius: 6px; border: 1px solid var(--border); background: var(--surface-1); color: var(--text-2); display: inline-flex; align-items: center; justify-content: center; cursor: pointer; }
.edit-budget-btn:hover { border-color: var(--brand-border); color: var(--brand); }
.stacked-num { display: flex; flex-direction: column; align-items: flex-end; line-height: 1.25; }
.sub-metric { font-size: var(--fs-xs); color: var(--text-3); }
.text-brand { color: var(--brand); font-weight: var(--fw-semibold); }
.text-ok { color: var(--status-ok); }
.text-danger { color: var(--danger); }
.refund-cell-pill { font-size: var(--fs-xs); font-weight: var(--fw-semibold); padding: 0.14rem 0.42rem; border-radius: 999px; background: var(--danger-soft); color: var(--danger); border: 1px solid var(--danger-border); }
.muted-dash { color: var(--text-3); opacity: 0.5; }
.unattr-dot { display: inline-block; width: 10px; height: 10px; border-radius: 50%; background: var(--text-3); opacity: 0.45; margin-left: 0.6rem; }
.skeleton-wrap { padding: 1rem; display: flex; flex-direction: column; gap: 0.6rem; }
.skeleton-row { height: 42px; border-radius: 8px; background: var(--surface-soft); }
.empty-state { padding: 3rem 1.5rem; text-align: center; display: flex; flex-direction: column; align-items: center; gap: 0.5rem; }
.empty-icon { color: var(--text-3); opacity: 0.6; }
.empty-title { margin: 0; font-size: var(--fs-lg); font-weight: var(--fw-semibold); color: var(--text-1); }
.empty-sub { margin: 0; font-size: var(--fs-base); color: var(--text-3); max-width: 420px; }
</style>
