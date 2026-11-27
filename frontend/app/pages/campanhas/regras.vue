<template>
  <div class="rules-page">
    <header class="page-header">
      <div>
        <h1 class="page-title">Regras Automáticas</h1>
        <p class="page-subtitle">
          Piloto automático baseado em vendas reais líquidas, reembolsos e WhatsApp IA para Meta Ads v26.0 e Google Ads v26.
        </p>
      </div>
      <div class="header-actions">
        <BaseButton variant="outline" size="sm" @click="templatesOpen = true">
          <Sparkles :size="15" /><span>Modelos prontos</span>
        </BaseButton>
        <BaseButton variant="primary" size="sm" @click="openNewRule">
          <Plus :size="15" /><span>Nova regra</span>
        </BaseButton>
      </div>
    </header>

    <section class="kpi-strip">
      <div class="kpi-item"><span class="kpi-label">Regras ativas</span><strong class="kpi-val">{{ stats.active }} / {{ stats.total }}</strong></div>
      <div class="kpi-item"><span class="kpi-label">Em simulação</span><strong class="kpi-val">{{ stats.dry_run }}</strong></div>
      <div class="kpi-item"><span class="kpi-label">Ações reais hoje</span><strong class="kpi-val is-brand">{{ stats.executed_today }}</strong></div>
      <div class="kpi-item"><span class="kpi-label">Simulações hoje</span><strong class="kpi-val">{{ stats.simulated_today }}</strong></div>
    </section>

    <div v-if="lastRunSummary" class="run-banner">
      <div class="run-banner-text">
        <strong>{{ lastRunSummary.dry_run ? 'Simulação' : 'Execução' }} — {{ lastRunSummary.rule_name }}:</strong>
        <span>
          {{ lastRunSummary.evaluated_count }} avaliados · {{ lastRunSummary.matched_count }} atingiram ·
          {{ lastRunSummary.executed_count }} registrados · {{ lastRunSummary.skipped_count }} ignorados
          <template v-if="lastRunSummary.failed_count > 0"> · {{ lastRunSummary.failed_count }} falharam</template>
        </span>
      </div>
      <button type="button" class="banner-close" @click="lastRunSummary = null"><X :size="14" /></button>
    </div>

    <div class="tabs-bar">
      <button type="button" class="tab-btn" :class="{ active: activeTab === 'rules' }" @click="activeTab = 'rules'">
        Regras configuradas ({{ rules.length }})
      </button>
      <button type="button" class="tab-btn" :class="{ active: activeTab === 'logs' }" @click="openLogsTab">
        Histórico de execuções ({{ logs.length }})
      </button>
    </div>

    <section v-if="activeTab === 'rules'" class="rules-list">
      <div v-if="loadingRules && rules.length === 0" class="empty-state">Carregando regras...</div>

      <div v-else-if="rules.length === 0" class="empty-state">
        <Bot :size="28" class="empty-icon" />
        <h3 class="empty-title">Nenhuma regra automática configurada</h3>
        <p class="empty-desc">
          Comece com um dos 15 modelos prontos (Stop Loss, Escala por ROAS Líquido, WhatsApp IA, Reativação por Venda Atrasada ou Dayparting).
        </p>
        <div class="empty-btns">
          <BaseButton variant="primary" size="sm" @click="templatesOpen = true">Explorar modelos prontos</BaseButton>
          <BaseButton variant="outline" size="sm" @click="openNewRule">Criar do zero</BaseButton>
        </div>
      </div>

      <CampaignRuleCard
        v-for="rule in rules"
        :key="rule.id"
        :rule="rule"
        :running="runningRuleId === rule.id"
        @toggle="toggleRule"
        @run="runRuleNow"
        @edit="openEditRule"
        @delete="pendingDeleteRule = $event"
      />
    </section>

    <CampaignRuleLogsTable
      v-else
      :logs="logs"
      :rules="rules"
      :loading="loadingLogs"
      :undoing-id="undoingLogId"
      :status-filter="logStatusFilter"
      :rule-filter="logRuleFilter"
      @update:status-filter="(v) => { logStatusFilter = v; fetchLogs() }"
      @update:rule-filter="(v) => { logRuleFilter = v; fetchLogs() }"
      @refresh="fetchLogs"
      @undo="pendingUndoLog = $event"
    />

    <CampaignRuleTemplatesModal v-model="templatesOpen" @select="onPickTemplate" />
    <CampaignRuleFormModal
      v-model="formOpen"
      :editing-rule="editingRule"
      :preset-payload="presetPayload"
      :pixel-names="pixelNames"
      :saving="saving"
      @save="onSaveRule"
    />
    <ConfirmModal
      :model-value="!!pendingDeleteRule"
      title="Excluir regra automática"
      :message="`Deseja remover a regra '${pendingDeleteRule?.name || ''}'? O histórico de execuções passadas será preservado.`"
      confirm-label="Excluir regra"
      variant="danger"
      @update:model-value="(v) => { if (!v) pendingDeleteRule = null }"
      @confirm="onConfirmDelete"
    />
    <ConfirmModal
      :model-value="!!pendingUndoLog"
      title="Desfazer alteração na plataforma"
      :message="`Restaurar '${pendingUndoLog?.object_name || ''}' ao estado anterior diretamente na API de anúncios?`"
      confirm-label="Desfazer agora"
      variant="primary"
      :loading="undoingLogId === pendingUndoLog?.id"
      @update:model-value="(v) => { if (!v) pendingUndoLog = null }"
      @confirm="onConfirmUndo"
    />
  </div>
</template>

<script setup lang="ts">
import { Bot, Plus, Sparkles, X } from 'lucide-vue-next'
import type { CampaignRule, CampaignRuleLog, CampaignRulePayload } from '#shared/types'
import type { CampaignRuleTemplate } from '~/utils/campaignRuleTemplates'
import CampaignRuleCard from '~/components/campaigns/rules/CampaignRuleCard.vue'
import CampaignRuleFormModal from '~/components/campaigns/rules/CampaignRuleFormModal.vue'
import CampaignRuleLogsTable from '~/components/campaigns/rules/CampaignRuleLogsTable.vue'
import CampaignRuleTemplatesModal from '~/components/campaigns/rules/CampaignRuleTemplatesModal.vue'

definePageMeta({ layout: 'app', middleware: 'auth' })
const { formatTitle } = useBranding()
useHead({ title: computed(() => formatTitle('Regras Automáticas')) })

const pixelStore = usePixelStore()
const pixelNames = computed(() => pixelStore.pixels.map((p) => p.name).filter(Boolean))

const {
  rules, stats, logs, loadingRules, loadingLogs, saving, runningRuleId, undoingLogId,
  logStatusFilter, logRuleFilter, lastRunSummary,
  fetchRules, fetchLogs, saveRule, toggleRule, deleteRule, runRuleNow, undoLog,
} = useCampaignRules()

const activeTab = ref<'rules' | 'logs'>('rules')
const templatesOpen = ref(false)
const formOpen = ref(false)
const editingRule = ref<CampaignRule | null>(null)
const presetPayload = ref<CampaignRulePayload | null>(null)
const pendingDeleteRule = ref<CampaignRule | null>(null)
const pendingUndoLog = ref<CampaignRuleLog | null>(null)

onMounted(async () => {
  await Promise.all([pixelStore.fetch(), fetchRules(), fetchLogs()])
})

const openLogsTab = () => { activeTab.value = 'logs'; fetchLogs() }
const openNewRule = () => { editingRule.value = null; presetPayload.value = null; formOpen.value = true }
const openEditRule = (rule: CampaignRule) => { editingRule.value = rule; presetPayload.value = null; formOpen.value = true }
const onPickTemplate = (tpl: CampaignRuleTemplate) => { editingRule.value = null; presetPayload.value = tpl.payload; formOpen.value = true }
const onSaveRule = async (payload: CampaignRulePayload) => { if (await saveRule(payload, editingRule.value?.id)) formOpen.value = false }
const onConfirmDelete = async () => { if (pendingDeleteRule.value && await deleteRule(pendingDeleteRule.value.id)) pendingDeleteRule.value = null }
const onConfirmUndo = async () => { if (pendingUndoLog.value && await undoLog(pendingUndoLog.value)) pendingUndoLog.value = null }
</script>

<style scoped>
.rules-page { display: flex; flex-direction: column; gap: 1rem; padding-bottom: 2rem; }
.page-header { display: flex; align-items: flex-start; justify-content: space-between; gap: 1rem; flex-wrap: wrap; }
.header-actions { display: flex; align-items: center; gap: 0.5rem; flex-wrap: wrap; }
.page-title { margin: 0; font-size: var(--fs-2xl); font-weight: var(--fw-bold); color: var(--text-strong); }
.page-subtitle { margin: 0.25rem 0 0; font-size: var(--fs-base); color: var(--text-3); }
.kpi-strip { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 0.75rem; }
.kpi-item { display: flex; flex-direction: column; gap: 0.2rem; padding: 0.85rem 1rem; border-radius: 12px; background: var(--surface-2); border: 1px solid var(--border-soft); }
.kpi-label { font-size: var(--fs-xs); text-transform: uppercase; letter-spacing: 0.05em; color: var(--text-3); font-weight: var(--fw-semibold); }
.kpi-val { font-size: var(--fs-xl); font-weight: var(--fw-bold); color: var(--text-1); }
.kpi-val.is-brand { color: var(--brand); }
.run-banner { display: flex; align-items: center; justify-content: space-between; gap: 0.75rem; padding: 0.7rem 0.95rem; border-radius: 10px; background: var(--brand-soft); border: 1px solid var(--brand-border); color: var(--text-1); font-size: var(--fs-base); }
.run-banner-text { display: flex; flex-wrap: wrap; gap: 0.35rem; }
.banner-close { background: transparent; border: none; color: var(--text-2); cursor: pointer; display: inline-flex; }
.tabs-bar { display: flex; gap: 0.35rem; border-bottom: 1px solid var(--border-soft); padding-bottom: 0.45rem; }
.tab-btn { padding: 0.45rem 0.85rem; border-radius: 8px; background: transparent; border: 1px solid transparent; color: var(--text-2); font-size: var(--fs-md); font-weight: var(--fw-medium); cursor: pointer; }
.tab-btn:hover { color: var(--text-1); background: var(--surface-soft); }
.tab-btn.active { background: var(--brand-soft); border-color: var(--brand-border); color: var(--brand); font-weight: var(--fw-semibold); }
.rules-list { display: flex; flex-direction: column; gap: 0.75rem; }
.empty-state { display: flex; flex-direction: column; align-items: center; text-align: center; gap: 0.55rem; padding: 2.5rem 1.25rem; border-radius: 12px; background: var(--surface-2); border: 1px solid var(--border-soft); }
.empty-icon { color: var(--brand); }
.empty-title { margin: 0; font-size: var(--fs-lg); font-weight: var(--fw-semibold); color: var(--text-1); }
.empty-desc { margin: 0; max-width: 480px; font-size: var(--fs-base); color: var(--text-3); line-height: 1.5; }
.empty-btns { display: flex; gap: 0.5rem; margin-top: 0.4rem; flex-wrap: wrap; justify-content: center; }
@media (max-width: 820px) { .kpi-strip { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
</style>
