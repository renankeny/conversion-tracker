import type { CampaignRow, PixelFinancialConfig } from '#shared/types'

export function extractCampaignErrMsg(err: unknown, fallback: string): string {
  const e = err as { data?: { error?: string }; message?: string }
  return e?.data?.error || e?.message || fallback
}

export function useCampaignMutations(
  pixelName: Ref<string>,
  currency: ComputedRef<string>,
  onRefreshOverview: (force: boolean) => Promise<void>,
) {
  const api = useAPI()
  const toast = useToast()
  const pixelStore = usePixelStore()

  const mutatingIds = reactive<Record<string, boolean>>({})
  const savingBudget = ref(false)
  const savingFinancials = ref(false)

  const toggleRowStatus = async (row: CampaignRow) => {
    if (!row.can_edit_status || row.platform === 'unattributed') return
    const key = `${row.platform}:${row.level}:${row.id}`
    const nextStatus = row.status === 'ACTIVE' ? 'PAUSED' : 'ACTIVE'
    mutatingIds[key] = true
    try {
      await api.post('/admin/campaigns/status', {
        platform: row.platform,
        level: row.level,
        entity_subtype: row.entity_subtype,
        object_id: row.id,
        parent_id: row.adset_id || '',
        status: nextStatus,
        pixel_name: pixelName.value,
        account_ref_id: row.account_ref_id || 0,
        ad_account_id: row.ad_account_id || '',
        login_customer_id: row.login_customer_id || '',
      })
      row.status = nextStatus
      toast.success(
        nextStatus === 'ACTIVE' ? 'Ativado com sucesso' : 'Pausado com sucesso',
        `${row.name} agora está ${nextStatus === 'ACTIVE' ? 'ativo' : 'pausado'}.`,
      )
    } catch (err) {
      toast.error('Falha ao alterar status', extractCampaignErrMsg(err, 'Erro na API da plataforma.'))
    } finally {
      mutatingIds[key] = false
    }
  }

  const saveRowBudget = async (row: CampaignRow, dailyBudget: number): Promise<boolean> => {
    savingBudget.value = true
    try {
      await api.post('/admin/campaigns/budget', {
        platform: row.platform,
        level: row.level,
        object_id: row.id,
        budget_id: row.budget_id || '',
        daily_budget: dailyBudget,
        currency: row.currency || currency.value,
        pixel_name: pixelName.value,
        account_ref_id: row.account_ref_id || 0,
        ad_account_id: row.ad_account_id || '',
        login_customer_id: row.login_customer_id || '',
      })
      row.daily_budget = dailyBudget
      toast.success('Orçamento atualizado', `Novo orçamento diário aplicado em ${row.name}.`)
      return true
    } catch (err) {
      toast.error('Falha ao editar orçamento', extractCampaignErrMsg(err, 'Erro ao atualizar orçamento.'))
      return false
    } finally {
      savingBudget.value = false
    }
  }

  const savePixelFinancials = async (pixelId: number, cfg: PixelFinancialConfig): Promise<boolean> => {
    savingFinancials.value = true
    try {
      await api.put(`/admin/pixels/${pixelId}/financials`, cfg)
      await pixelStore.fetch(true)
      await onRefreshOverview(true)
      toast.success('Custos salvos', 'Taxas, impostos e CMV aplicados ao Lucro Líquido Real.')
      return true
    } catch (err) {
      toast.error('Erro ao salvar custos', extractCampaignErrMsg(err, 'Não foi possível salvar os custos do pixel.'))
      return false
    } finally {
      savingFinancials.value = false
    }
  }

  return { mutatingIds, savingBudget, savingFinancials, toggleRowStatus, saveRowBudget, savePixelFinancials }
}
