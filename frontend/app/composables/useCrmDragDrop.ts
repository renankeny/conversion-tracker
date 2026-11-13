import { ref, watch, type Ref } from 'vue'
import type { CrmLead, CrmStage } from '#shared/types'
import { useAPI } from '~/composables/useAPI'
import { resolveStageEffectiveEvent } from '~/utils/crmEffectiveEvent'
import { errorMessage } from '~/utils/errorMessage'

interface UseCrmDragDropDeps {
  stages: Ref<CrmStage[]>
  leadsByStage: Ref<Record<string, CrmLead[]>>
  fetchBoard: (silent?: boolean) => Promise<void>
  canManageStages: Ref<boolean>
}

interface PendingLeadMove {
  lead: CrmLead
  from: string
  to: string
  fromIndex: number
  toIndex: number
}

export const useCrmDragDrop = (deps: UseCrmDragDropDeps) => {
  const api = useAPI()
  const toast = useToast()

  const showPurchaseModal = ref(false)
  const purchaseValue = ref<number | null>(null)
  const isConfirmingPurchase = ref(false)
  let pendingMoveEvent: PendingLeadMove | null = null

  const saveStagesOrder = async (silent = false) => {
    if (!deps.canManageStages.value) return
    const payload = {
      stages: deps.stages.value.map((s, idx) => ({ id: s.id, order_index: idx })),
    }
    try {
      await api.put('/admin/crm/stages/reorder', payload)
      if (!silent) toast.success('Sucesso', 'Ordem do funil atualizada.')
    } catch (err: unknown) {
      toast.error('Erro', errorMessage(err, 'Falha ao salvar ordem das etapas.'))
      await deps.fetchBoard(true)
    }
  }

  const moveStageUp = async (index: number) => {
    if (index <= 0 || !deps.canManageStages.value) return
    const current = deps.stages.value[index]
    const prev = deps.stages.value[index - 1]
    if (!current || !prev) return
    deps.stages.value[index] = prev
    deps.stages.value[index - 1] = current
    await saveStagesOrder()
  }

  const moveStageDown = async (index: number) => {
    if (index >= deps.stages.value.length - 1 || !deps.canManageStages.value) return
    const current = deps.stages.value[index]
    const next = deps.stages.value[index + 1]
    if (!current || !next) return
    deps.stages.value[index] = next
    deps.stages.value[index + 1] = current
    await saveStagesOrder()
  }

  const onStageDragEnd = async (evt: { oldIndex?: number; newIndex?: number }) => {
    if (evt.oldIndex === evt.newIndex) return
    if (!deps.canManageStages.value) {
      await deps.fetchBoard(true)
      return
    }
    await saveStagesOrder(true)
  }

  const executeMove = async (lead: CrmLead, stageId: string, value: number): Promise<boolean> => {
    const parsedStage = parseInt(stageId, 10)
    lead.stage_id = parsedStage
    try {
      await api.put(`/admin/crm/leads/${lead.id}/move`, { stage_id: parsedStage, value })
      return true
    } catch (err: unknown) {
      toast.error('Erro ao mover lead', errorMessage(err, 'Não foi possível mover o lead para esta etapa.'))
      await deps.fetchBoard(true)
      return false
    }
  }

  interface SortableDragEvent {
    from?: { dataset?: { stageId?: string } }
    to?: { dataset?: { stageId?: string } }
    oldIndex?: number
    newIndex?: number
  }

  const onDragEnd = (evt: SortableDragEvent) => {
    const fromStageId = evt.from?.dataset?.stageId
    const toStageId = evt.to?.dataset?.stageId
    if (!fromStageId || !toStageId || fromStageId === toStageId) return

    const newIndex = evt.newIndex ?? 0
    const movedLead = deps.leadsByStage.value[toStageId]?.[newIndex]
    if (!movedLead) return
    const targetStage = deps.stages.value.find((s) => String(s.id) === String(toStageId))
    const effectiveEvent = resolveStageEffectiveEvent(targetStage, movedLead.tracked_user)

    if (effectiveEvent === 'Purchase' && (!movedLead.value || movedLead.value <= 0)) {
      pendingMoveEvent = {
        lead: movedLead,
        from: String(fromStageId),
        to: String(toStageId),
        fromIndex: evt.oldIndex ?? 0,
        toIndex: newIndex,
      }
      purchaseValue.value = null
      showPurchaseModal.value = true
      return
    }
    void executeMove(movedLead, String(toStageId), movedLead.value || 0)
  }

  const revertPendingMove = () => {
    if (!pendingMoveEvent) return
    const { lead, from, to, fromIndex, toIndex } = pendingMoveEvent
    pendingMoveEvent = null

    const toList = deps.leadsByStage.value[to]
    if (toList) {
      const idx = toList.findIndex((l) => l.id === lead.id)
      if (idx >= 0) toList.splice(idx, 1)
      else if (toIndex >= 0 && toIndex < toList.length) toList.splice(toIndex, 1)
    }

    lead.stage_id = Number(from)
    if (!deps.leadsByStage.value[from]) {
      deps.leadsByStage.value[from] = []
    }
    const fromList = deps.leadsByStage.value[from]!
    if (!fromList.some((l) => l.id === lead.id)) {
      const insertAt = Math.min(Math.max(0, fromIndex), fromList.length)
      fromList.splice(insertAt, 0, lead)
    }
  }

  const cancelPurchase = () => {
    revertPendingMove()
    showPurchaseModal.value = false
  }

  const confirmPurchase = async () => {
    if (isConfirmingPurchase.value) return
    if (!purchaseValue.value || purchaseValue.value <= 0) {
      toast.error('Valor obrigatório', 'Informe um valor maior que R$ 0,00 para confirmar a venda.')
      return
    }
    if (!pendingMoveEvent) {
      showPurchaseModal.value = false
      return
    }
    isConfirmingPurchase.value = true
    const { lead, to } = pendingMoveEvent
    const prevValue = lead.value
    const chosenValue = purchaseValue.value
    lead.value = chosenValue
    try {
      const ok = await executeMove(lead, to, chosenValue)
      if (ok) {
        pendingMoveEvent = null
        showPurchaseModal.value = false
      } else {
        lead.value = prevValue
      }
    } finally {
      isConfirmingPurchase.value = false
    }
  }

  watch(showPurchaseModal, (open) => {
    if (!open && pendingMoveEvent && !isConfirmingPurchase.value) {
      revertPendingMove()
    }
  })

  return {
    showPurchaseModal,
    purchaseValue,
    isConfirmingPurchase,
    moveStageUp,
    moveStageDown,
    onStageDragEnd,
    onDragEnd,
    cancelPurchase,
    confirmPurchase,
  }
}
