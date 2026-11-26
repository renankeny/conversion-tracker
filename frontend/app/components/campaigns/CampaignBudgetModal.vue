<template>
  <BaseModal
    :model-value="Boolean(row)"
    title="Editar Orçamento Diário"
    :description="row ? `${row.name} (${row.platform === 'meta' ? 'Meta Ads v26.0' : 'Google Ads v26'})` : ''"
    size="sm"
    @update:model-value="onUpdateOpen"
  >
    <div v-if="row" class="budget-form">
      <div v-if="row.platform === 'google' && row.budget_shared" class="shared-warning">
        <AlertTriangle :size="15" />
        <span>
          Este orçamento do Google Ads é compartilhado entre múltiplas campanhas. A alteração impactará todas as campanhas vinculadas.
        </span>
      </div>

      <div class="current-box">
        <span class="current-label">Orçamento diário atual</span>
        <strong class="current-val">{{ formatCampaignMoney(row.daily_budget, row.currency || currency) }}</strong>
      </div>

      <BaseInput
        v-model="amountStr"
        type="number"
        label="Novo orçamento diário"
        placeholder="Ex: 150.00"
        hint="Informe o valor em moeda real (ex: 150.00). Será enviado diretamente para a API oficial."
      />
    </div>

    <template #footer>
      <BaseButton variant="outline" :disabled="saving" @click="emit('close')">Cancelar</BaseButton>
      <BaseButton variant="primary" :loading="saving" :disabled="!isValidAmount" @click="onSubmit">
        Salvar Orçamento
      </BaseButton>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import { AlertTriangle } from 'lucide-vue-next'
import type { CampaignRow } from '#shared/types'
import { formatCampaignMoney } from '~/utils/campaignFormatters'

const props = defineProps<{
  row: CampaignRow | null
  currency: string
  saving: boolean
}>()

const emit = defineEmits<{
  close: []
  save: [row: CampaignRow, dailyBudget: number]
}>()

const amountStr = ref('')

watch(
  () => props.row,
  (r) => {
    amountStr.value = r && r.daily_budget > 0 ? r.daily_budget.toFixed(2) : ''
  },
  { immediate: true },
)

const parsedAmount = computed(() => Number(String(amountStr.value).replace(',', '.')))
const isValidAmount = computed(() => Number.isFinite(parsedAmount.value) && parsedAmount.value > 0)

const onUpdateOpen = (open: boolean) => {
  if (!open) emit('close')
}

const onSubmit = () => {
  if (!props.row || !isValidAmount.value) return
  emit('save', props.row, parsedAmount.value)
}
</script>

<style scoped>
.budget-form {
  display: flex;
  flex-direction: column;
  gap: 0.9rem;
}
.shared-warning {
  display: flex;
  align-items: flex-start;
  gap: 0.5rem;
  padding: 0.7rem 0.85rem;
  border-radius: 9px;
  background: var(--warning-soft);
  border: 1px solid var(--warning-border);
  color: var(--warning);
  font-size: var(--fs-sm);
  line-height: 1.4;
}
.current-box {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.65rem 0.85rem;
  border-radius: 9px;
  background: var(--surface-1);
  border: 1px solid var(--border);
}
.current-label {
  font-size: var(--fs-base);
  color: var(--text-3);
}
.current-val {
  font-size: var(--fs-lg);
  font-weight: var(--fw-bold);
  color: var(--text-strong);
}
</style>
