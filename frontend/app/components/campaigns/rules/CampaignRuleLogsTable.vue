<template>
  <div class="logs-wrap">
    <div class="logs-toolbar">
      <div class="toolbar-filters">
        <BaseDropdown
          :model-value="statusFilter"
          :options="statusOptions"
          size="sm"
          @update:model-value="$emit('update:statusFilter', String($event || 'all'))"
        />
        <BaseDropdown
          :model-value="ruleFilter"
          :options="ruleOptions"
          size="sm"
          @update:model-value="$emit('update:ruleFilter', Number($event || 0))"
        />
      </div>
      <BaseButton variant="outline" size="sm" :loading="loading" @click="$emit('refresh')">
        <RefreshCw :size="14" />
        <span>Atualizar</span>
      </BaseButton>
    </div>

    <div v-if="loading && logs.length === 0" class="empty-box">Carregando histórico...</div>

    <div v-else-if="logs.length === 0" class="empty-box">
      Nenhuma execução registrada ainda. Clique em <strong>Simular</strong> em qualquer regra para testar com segurança.
    </div>

    <div v-else class="table-shell">
      <table class="logs-table">
        <thead>
          <tr>
            <th>Data</th>
            <th>Regra & Objeto</th>
            <th>Alteração</th>
            <th>Métricas no momento</th>
            <th>Status</th>
            <th class="text-right">Ação</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in logs" :key="item.id">
            <td class="col-date">{{ formatDate(item.created_at) }}</td>
            <td>
              <div class="obj-name">{{ item.object_name }}</div>
              <div class="obj-sub">
                {{ item.rule_name }} · {{ item.platform.toUpperCase() }} ({{ item.level }})
              </div>
              <div v-if="item.reason" class="obj-reason">{{ item.reason }}</div>
              <div v-if="item.error_message" class="obj-err">{{ item.error_message }}</div>
            </td>
            <td class="col-diff">{{ formatChange(item) }}</td>
            <td class="col-snap">
              <span>Gasto {{ fmtBRL(item.metrics_snapshot.spend) }}</span>
              <span>Vendas {{ item.metrics_snapshot.net_sales }}</span>
              <span>ROAS {{ item.metrics_snapshot.net_roas.toFixed(2) }}x</span>
              <span v-if="item.metrics_snapshot.ai_qualified > 0">IA {{ item.metrics_snapshot.ai_qualified }}</span>
            </td>
            <td>
              <BaseBadge :variant="statusBadge(item.status)">{{ statusLabel(item.status) }}</BaseBadge>
            </td>
            <td class="text-right">
              <BaseButton
                v-if="canUndo(item)"
                variant="outline"
                size="sm"
                :loading="undoingId === item.id"
                @click="$emit('undo', item)"
              >
                <RotateCcw :size="13" />
                <span>Desfazer</span>
              </BaseButton>
              <span v-else class="muted-dash">—</span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { RefreshCw, RotateCcw } from 'lucide-vue-next'
import type { CampaignRule, CampaignRuleLog, CampaignRuleLogStatus } from '#shared/types'
import type { DropdownOption } from '~/components/BaseDropdown.vue'

const props = defineProps<{
  logs: CampaignRuleLog[]
  rules: CampaignRule[]
  loading?: boolean
  undoingId?: number | null
  statusFilter: string
  ruleFilter: number | string
}>()

defineEmits<{
  'update:statusFilter': [value: string]
  'update:ruleFilter': [value: number]
  refresh: []
  undo: [log: CampaignRuleLog]
}>()

const statusOptions: DropdownOption[] = [
  { value: 'all', label: 'Todos os status' },
  { value: 'executed', label: 'Executadas' },
  { value: 'simulated', label: 'Simuladas' },
  { value: 'failed', label: 'Com falha' },
  { value: 'undone', label: 'Desfeitas' },
]

const ruleOptions = computed<DropdownOption[]>(() => [
  { value: 0, label: 'Todas as regras' },
  ...props.rules.map((r) => ({ value: r.id, label: r.name })),
])

const canUndo = (l: CampaignRuleLog) =>
  !l.dry_run && l.status === 'executed' && l.action_type !== 'notify_only'

const fmtBRL = (v: number) =>
  `R$ ${Number(v || 0).toFixed(2).replace('.', ',')}`

const formatDate = (iso: string) =>
  new Intl.DateTimeFormat('pt-BR', { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' }).format(new Date(iso))

const formatChange = (l: CampaignRuleLog) => {
  if (l.action_type === 'notify_only') return 'Apenas alerta'
  if (l.new_status) return `${l.previous_status || '—'} → ${l.new_status}`
  if (l.new_budget > 0) return `${fmtBRL(l.previous_budget)} → ${fmtBRL(l.new_budget)}`
  return l.action_type
}

const statusLabel = (s: CampaignRuleLogStatus) =>
  ({ executed: 'Executada', simulated: 'Simulada', failed: 'Falhou', undone: 'Desfeita' }[s] || s)

const statusBadge = (s: CampaignRuleLogStatus): 'success' | 'warning' | 'danger' | 'default' => {
  if (s === 'executed') return 'success'
  if (s === 'simulated') return 'warning'
  if (s === 'failed') return 'danger'
  return 'default'
}
</script>

<style scoped>
.logs-wrap { display: flex; flex-direction: column; gap: 0.75rem; }
.logs-toolbar { display: flex; align-items: center; justify-content: space-between; gap: 0.75rem; flex-wrap: wrap; }
.toolbar-filters { display: flex; align-items: center; gap: 0.5rem; flex-wrap: wrap; min-width: 240px; }
.toolbar-filters > * { min-width: 185px; }
.empty-box {
  padding: 2rem 1rem; text-align: center; border-radius: 12px;
  background: var(--surface-2); border: 1px solid var(--border-soft); font-size: var(--fs-base); color: var(--text-3);
}
.table-shell {
  overflow-x: auto; border-radius: 12px; background: var(--surface-2); border: 1px solid var(--border-soft);
}
.logs-table { width: 100%; border-collapse: collapse; font-size: var(--fs-base); }
.logs-table th {
  text-align: left; padding: 0.7rem 0.85rem; font-size: var(--fs-xs); font-weight: var(--fw-semibold);
  text-transform: uppercase; letter-spacing: 0.05em; color: var(--text-3); border-bottom: 1px solid var(--border-soft);
}
.logs-table td { padding: 0.75rem 0.85rem; border-bottom: 1px solid var(--border-faint); vertical-align: top; color: var(--text-2); }
.logs-table tr:last-child td { border-bottom: none; }
.col-date { white-space: nowrap; color: var(--text-3); font-size: var(--fs-sm); }
.obj-name { font-weight: var(--fw-semibold); color: var(--text-1); }
.obj-sub { font-size: var(--fs-sm); color: var(--text-3); margin-top: 0.1rem; }
.obj-reason { font-size: var(--fs-sm); color: var(--text-2); margin-top: 0.25rem; }
.obj-err { font-size: var(--fs-sm); color: var(--danger); margin-top: 0.2rem; }
.col-diff { white-space: nowrap; font-weight: var(--fw-semibold); color: var(--text-1); }
.col-snap { display: flex; flex-wrap: wrap; gap: 0.3rem; max-width: 230px; }
.col-snap span {
  font-size: var(--fs-xs); padding: 0.15rem 0.45rem; border-radius: 6px;
  background: var(--surface-deep); border: 1px solid var(--border-soft); color: var(--text-2);
}
.muted-dash { color: var(--text-3); }
</style>
