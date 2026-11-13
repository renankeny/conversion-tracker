<template>
  <section class="card">
    <h3 class="card-title">Retenção de dados</h3>
    <p class="card-hint">
      Os registros são apagados automaticamente todo dia às 03:00 (horário São Paulo).
      Você pode definir um override por conta abaixo. Use <strong>0</strong> pra desativar
      a retenção (mantém para sempre) ou deixe vazio pra usar o default global.
    </p>

    <div v-if="!maintenanceData" class="muted">Carregando...</div>

    <div v-else class="retention-grid">
      <div class="retention-item">
        <span class="retention-label">Eventos completos (sent_events)</span>
        <span class="retention-value">
          {{ maintenanceData.effective_sent_events_retention_days ?? maintenanceData.sent_events_retention_days ?? '∞' }}
          <small>dias</small>
        </span>
        <small v-if="maintenanceData.account_sent_events_retention_days != null" class="retention-source">
          override da conta
        </small>
        <small v-else class="retention-source">default global</small>
      </div>
      <div class="retention-item">
        <span class="retention-label">Agregados (event_counters)</span>
        <span class="retention-value">
          {{ maintenanceData.effective_event_counters_retention_days ?? maintenanceData.event_counters_retention_days ?? '∞' }}
          <small>dias</small>
        </span>
        <small v-if="maintenanceData.account_event_counters_retention_days != null" class="retention-source">
          override da conta
        </small>
        <small v-else class="retention-source">default global</small>
      </div>
    </div>

    <div class="retention-form">
      <h4 class="card-subtitle">Ajustar retenção desta empresa</h4>
      <div class="form-row">
        <BaseInput
          v-model.number="retentionForm.sent_events"
          type="number"
          label="sent_events (dias)"
          placeholder="vazio = usa default global"
          hint="0 = nunca apaga; >0 = mantém os últimos N dias"
          min="0"
        />
        <BaseInput
          v-model.number="retentionForm.event_counters"
          type="number"
          label="event_counters (dias)"
          placeholder="vazio = usa default global"
          hint="0 = nunca apaga; >0 = mantém os últimos N dias"
          min="0"
        />
      </div>
      <div class="retention-actions">
        <BaseButton variant="primary" :loading="savingRetention" @click="onSaveRetention">
          Salvar retenção
        </BaseButton>
        <BaseButton variant="ghost" :loading="savingRetention" @click="onResetRetention">
          Voltar pro default global
        </BaseButton>
      </div>
    </div>

    <div class="retention-actions">
      <BaseButton variant="outline" :loading="pruning" @click="confirmPrune = true">
        Executar limpeza agora
      </BaseButton>
      <BaseButton variant="danger" :loading="resetting" @click="confirmReset = true">
        Resetar tudo desta empresa
      </BaseButton>
    </div>

    <div v-if="lastPruneResults.length" class="prune-results">
      <h4>Resultado da última limpeza:</h4>
      <div v-for="r in lastPruneResults" :key="r.table" class="prune-row">
        <code>{{ r.table }}</code>
        <span>
          <strong>{{ formatNumber(r.deleted) }}</strong> rows apagadas
          <small v-if="r.retention_days">({{ r.retention_days }} dias)</small>
        </span>
      </div>
    </div>

    <div v-if="lastResetResult" class="prune-results">
      <h4>Reset executado:</h4>
      <div class="prune-row">
        <code>sent_events</code>
        <span><strong>{{ formatNumber(lastResetResult.deleted_sent_events) }}</strong> rows apagadas</span>
      </div>
      <div class="prune-row">
        <code>event_counters</code>
        <span><strong>{{ formatNumber(lastResetResult.deleted_event_counters) }}</strong> rows apagadas</span>
      </div>
    </div>

    <ConfirmModal v-model="confirmPrune" title="Executar limpeza agora?" message="Vai apagar todos os registros mais antigos que a retenção configurada. Operação irreversível." confirm-label="Sim, limpar agora" variant="danger" :loading="pruning" @confirm="onPrune" />
    <ConfirmModal v-model="confirmReset" title="Resetar TUDO desta empresa?" message="Vai apagar TODOS os sent_events e event_counters desta empresa — independente da idade. Histórico de envios à Meta CAPI, contadores de PageView, conversões — tudo. Pixels, tracked_users e configurações ficam intactos. Operação IRREVERSÍVEL." confirm-label="Sim, apagar tudo" variant="danger" :loading="resetting" @confirm="onReset" />
  </section>
</template>

<script setup lang="ts">
import type { MaintenanceStatusResponse, PruneResultRow, ResetAccountResponse } from '#shared/types'
import { errorMessage } from '~/utils/errorMessage'

const settings = useSettings()
const tenantStore = useTenantStore()
const toast = useToast()

const maintenanceData = ref<MaintenanceStatusResponse | null>(null)
const pruning = ref(false)
const confirmPrune = ref(false)
const lastPruneResults = ref<PruneResultRow[]>([])
const retentionForm = reactive<{ sent_events: number | string | undefined; event_counters: number | string | undefined }>({
  sent_events: undefined, event_counters: undefined,
})

const normalizeRetentionDays = (val: number | string | null | undefined): number | null => {
  if (val === '' || val === null || val === undefined) return null
  const n = typeof val === 'number' ? val : Number(String(val).trim())
  return Number.isFinite(n) && n >= 0 ? Math.trunc(n) : null
}
const savingRetention = ref(false)
const confirmReset = ref(false)
const resetting = ref(false)
const lastResetResult = ref<ResetAccountResponse | null>(null)

const formatNumber = (n: number): string => new Intl.NumberFormat('pt-BR').format(n)

const loadMaintenance = async () => {
  try {
    maintenanceData.value = await settings.maintenanceStatus()
    retentionForm.sent_events = maintenanceData.value.account_sent_events_retention_days ?? undefined
    retentionForm.event_counters = maintenanceData.value.account_event_counters_retention_days ?? undefined
  } catch (error: unknown) {
    toast.error('Erro', errorMessage(error, 'Falha ao consultar retenção'))
  }
}

const onPrune = async () => {
  pruning.value = true
  try {
    const res = await settings.pruneNow()
    lastPruneResults.value = res.results || []
    toast.success('Limpeza executada', 'Veja resultado abaixo')
    confirmPrune.value = false
  } catch (error: unknown) {
    toast.error('Erro', errorMessage(error, 'Falha ao executar limpeza'))
  } finally {
    pruning.value = false
  }
}

const onSaveRetention = async () => {
  savingRetention.value = true
  try {
    await settings.updateRetention({
      sent_events_retention_days: normalizeRetentionDays(retentionForm.sent_events),
      event_counters_retention_days: normalizeRetentionDays(retentionForm.event_counters),
    })
    toast.success('Retenção atualizada', 'Próxima limpeza usará a nova janela')
    await loadMaintenance()
  } catch (error: unknown) {
    toast.error('Erro', errorMessage(error, 'Falha ao salvar retenção'))
  } finally {
    savingRetention.value = false
  }
}

const onResetRetention = async () => {
  savingRetention.value = true
  try {
    await settings.updateRetention({
      sent_events_retention_days: null,
      event_counters_retention_days: null,
    })
    retentionForm.sent_events = undefined
    retentionForm.event_counters = undefined
    toast.success('Override removido', 'Conta volta a usar o default global')
    await loadMaintenance()
  } catch (error: unknown) {
    toast.error('Erro', errorMessage(error, 'Falha ao remover override'))
  } finally {
    savingRetention.value = false
  }
}

const onReset = async () => {
  resetting.value = true
  try {
    const res = await settings.resetAccount()
    lastResetResult.value = res
    toast.success(
      'Empresa resetada',
      `${formatNumber(res.deleted_sent_events)} eventos + ${formatNumber(res.deleted_event_counters)} contadores apagados`
    )
    confirmReset.value = false
  } catch (error: unknown) {
    toast.error('Erro', errorMessage(error, 'Falha ao resetar conta'))
  } finally {
    resetting.value = false
  }
}

watch(() => tenantStore.switchVersion, () => {
  lastPruneResults.value = []
  lastResetResult.value = null
  loadMaintenance()
})

onMounted(loadMaintenance)
</script>

<style scoped>
.card { background: var(--surface-2); border: 1px solid var(--border); border-radius: 12px; padding: 1.5rem; box-shadow: var(--shadow-card); }
.card-title { font-family: var(--font-sans); font-size: var(--fs-xl); font-weight: var(--fw-semibold); color: var(--text-strong); margin-bottom: 0.5rem; }
.card-hint { font-size: var(--fs-md); color: var(--text-2); margin-bottom: 1rem; line-height: 1.5; }
.muted { color: var(--text-3); font-size: var(--fs-md); font-style: italic; }
.retention-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 0.75rem 1rem; }
.retention-item { display: flex; flex-direction: column; gap: 0.25rem; }
.retention-label { font-size: var(--fs-xs); text-transform: uppercase; letter-spacing: 0.05em; color: var(--text-3); font-weight: var(--fw-semibold); }
.retention-value { font-family: var(--font-sans); font-size: var(--fs-2xl); font-weight: var(--fw-bold); color: var(--text-strong); }
.retention-value small { font-size: var(--fs-sm); font-weight: var(--fw-normal); color: var(--text-3); }
.retention-source { font-size: var(--fs-xs); color: var(--text-3); font-style: italic; margin-top: 0.125rem; }
.retention-form { margin-top: 1.25rem; padding-top: 1.25rem; border-top: 1px solid var(--border-soft); }
.card-subtitle { font-size: var(--fs-md); font-weight: var(--fw-semibold); color: var(--text-strong); margin-bottom: 0.875rem; }
.form-row { display: grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: 1rem; margin-bottom: 1rem; }
.retention-actions { display: flex; gap: 0.5rem; flex-wrap: wrap; margin-top: 0.5rem; }
.prune-results { margin-top: 1rem; padding-top: 1rem; border-top: 1px solid var(--border-soft); }
.prune-results h4 { font-size: var(--fs-base); font-weight: var(--fw-semibold); color: var(--text-strong); margin-bottom: 0.5rem; }
.prune-row { display: flex; justify-content: space-between; align-items: center; padding: 0.5rem 0; font-size: var(--fs-md); border-bottom: 1px solid var(--border-soft); }
.prune-row:last-child { border-bottom: none; }
.prune-row code { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; background: var(--surface-soft); padding: 0.125rem 0.5rem; border-radius: 4px; }
</style>

