import type {
  CampaignRule,
  CampaignRuleLog,
  CampaignRulePayload,
  CampaignRuleRunSummary,
  CampaignRuleStats,
} from '#shared/types'

const emptyStats = (): CampaignRuleStats => ({
  total: 0,
  active: 0,
  dry_run: 0,
  executed_today: 0,
  simulated_today: 0,
})

function extractErr(err: unknown, fallback: string): string {
  const e = err as { data?: { error?: string }; message?: string }
  return e?.data?.error || e?.message || fallback
}

export function useCampaignRules() {
  const api = useAPI()
  const toast = useToast()

  const rules = ref<CampaignRule[]>([])
  const stats = ref<CampaignRuleStats>(emptyStats())
  const logs = ref<CampaignRuleLog[]>([])
  const loadingRules = ref(false)
  const loadingLogs = ref(false)
  const saving = ref(false)
  const runningRuleId = ref<number | null>(null)
  const undoingLogId = ref<number | null>(null)
  const logStatusFilter = ref('all')
  const logRuleFilter = ref<number | string>(0)
  const lastRunSummary = ref<CampaignRuleRunSummary | null>(null)

  const fetchRules = async () => {
    loadingRules.value = true
    try {
      const res = await api.get<{ data?: CampaignRule[]; stats?: CampaignRuleStats }>('/admin/campaigns/rules')
      rules.value = res.data || []
      stats.value = res.stats || emptyStats()
    } catch (err) {
      toast.error('Erro ao carregar regras', extractErr(err, 'Não foi possível buscar as regras de campanha.'))
    } finally {
      loadingRules.value = false
    }
  }

  const fetchLogs = async () => {
    loadingLogs.value = true
    try {
      const params = new URLSearchParams({ limit: '80' })
      if (logStatusFilter.value && logStatusFilter.value !== 'all') {
        params.set('status', logStatusFilter.value)
      }
      const rid = Number(logRuleFilter.value || 0)
      if (rid > 0) params.set('rule_id', String(rid))

      const res = await api.get<{ data?: CampaignRuleLog[] }>(`/admin/campaigns/rules/logs?${params.toString()}`)
      logs.value = res.data || []
    } catch (err) {
      toast.error('Erro ao carregar histórico', extractErr(err, 'Não foi possível buscar o histórico de execuções.'))
    } finally {
      loadingLogs.value = false
    }
  }

  const saveRule = async (payload: CampaignRulePayload, editingId?: number | null): Promise<boolean> => {
    saving.value = true
    try {
      if (editingId && editingId > 0) {
        await api.put(`/admin/campaigns/rules/${editingId}`, payload)
        toast.success('Regra atualizada', 'As alterações foram salvas com sucesso.')
      } else {
        await api.post('/admin/campaigns/rules', payload)
        toast.success('Regra criada', 'A regra automática já está pronta para operar.')
      }
      await fetchRules()
      return true
    } catch (err) {
      toast.error('Erro ao salvar regra', extractErr(err, 'Verifique os campos e tente novamente.'))
      return false
    } finally {
      saving.value = false
    }
  }

  const toggleRule = async (rule: CampaignRule) => {
    const next = !rule.enabled
    try {
      await api.request(`/admin/campaigns/rules/${rule.id}/toggle`, { method: 'PATCH', body: { enabled: next } })
      rule.enabled = next
      stats.value.active = rules.value.filter((r) => r.enabled).length
      toast.success(next ? 'Regra ativada' : 'Regra pausada', rule.name)
    } catch (err) {
      toast.error('Erro ao alterar status', extractErr(err, 'Não foi possível atualizar a regra.'))
    }
  }

  const deleteRule = async (ruleId: number): Promise<boolean> => {
    try {
      await api.delete(`/admin/campaigns/rules/${ruleId}`)
      toast.success('Regra removida', 'A regra foi excluída. O histórico de auditoria foi mantido.')
      await fetchRules()
      return true
    } catch (err) {
      toast.error('Erro ao excluir regra', extractErr(err, 'Não foi possível excluir a regra.'))
      return false
    }
  }

  const runRuleNow = async (rule: CampaignRule, dryRun: boolean) => {
    runningRuleId.value = rule.id
    try {
      const qs = dryRun ? '?dry_run=1' : ''
      const res = await api.post<{ data: CampaignRuleRunSummary }>(`/admin/campaigns/rules/${rule.id}/run${qs}`, {})
      lastRunSummary.value = res.data
      const label = res.data.dry_run ? 'Simulação concluída' : 'Execução concluída'
      toast.success(
        label,
        `${res.data.evaluated_count} avaliados · ${res.data.matched_count} atingiram · ${res.data.executed_count} registrados.`,
      )
      await Promise.all([fetchRules(), fetchLogs()])
    } catch (err) {
      toast.error('Erro ao executar regra', extractErr(err, 'Falha ao consultar campanhas ou aplicar ação.'))
    } finally {
      runningRuleId.value = null
    }
  }

  const undoLog = async (log: CampaignRuleLog): Promise<boolean> => {
    undoingLogId.value = log.id
    try {
      await api.post(`/admin/campaigns/rules/logs/${log.id}/undo`, {})
      toast.success('Ação desfeita', `${log.object_name} foi restaurado ao estado anterior.`)
      await fetchLogs()
      return true
    } catch (err) {
      toast.error('Erro ao desfazer ação', extractErr(err, 'Não foi possível reverter a alteração na plataforma.'))
      return false
    } finally {
      undoingLogId.value = null
    }
  }

  return {
    rules,
    stats,
    logs,
    loadingRules,
    loadingLogs,
    saving,
    runningRuleId,
    undoingLogId,
    logStatusFilter,
    logRuleFilter,
    lastRunSummary,
    fetchRules,
    fetchLogs,
    saveRule,
    toggleRule,
    deleteRule,
    runRuleNow,
    undoLog,
  }
}
