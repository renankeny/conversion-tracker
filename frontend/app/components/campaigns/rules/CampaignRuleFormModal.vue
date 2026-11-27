<template>
  <BaseModal
    :model-value="modelValue"
    :title="editingRule ? 'Editar regra automática' : 'Nova regra automática'"
    description="Defina condições com vendas reais líquidas, escolha a ação e proteja sua operação com limites e modo simulação."
    size="lg"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <form class="rule-form" @submit.prevent="onSubmit">
      <div class="grid-2">
        <BaseInput v-model="form.name" label="Nome da regra" placeholder="Ex: Stop Loss > R$ 150 sem venda" required />
        <BaseDropdown
          :model-value="form.rule_kind"
          label="Tipo de regra"
          :options="RULE_KIND_OPTIONS"
          size="sm"
          @update:model-value="(v) => onChangeKind(String(v || 'metric') as CampaignRuleKind)"
        />
      </div>

      <BaseInput
        v-model="form.description"
        label="Descrição (opcional)"
        placeholder="Explique o objetivo desta regra para a equipe"
      />

      <div class="grid-3">
        <BaseDropdown
          :model-value="form.platform"
          label="Plataforma"
          :options="RULE_PLATFORM_OPTIONS"
          size="sm"
          @update:model-value="(v) => (form.platform = String(v || 'all') as 'all' | 'meta' | 'google')"
        />
        <BaseDropdown
          :model-value="form.pixel_name"
          label="Pixel / Funil"
          :options="pixelOptions"
          size="sm"
          @update:model-value="(v) => (form.pixel_name = String(v || ''))"
        />
        <BaseDropdown
          :model-value="form.level"
          label="Nível avaliado"
          :options="RULE_LEVEL_OPTIONS"
          size="sm"
          @update:model-value="(v) => (form.level = String(v || 'adset') as CampaignRuleLevel)"
        />
      </div>

      <div class="grid-3">
        <BaseDropdown
          :model-value="form.period_preset"
          label="Janela de métricas"
          :options="RULE_PERIOD_OPTIONS"
          size="sm"
          @update:model-value="(v) => (form.period_preset = String(v || 'today') as CampaignRulePeriod)"
        />
        <BaseDropdown
          :model-value="form.target_status"
          label="Aplicar em itens com status"
          :options="RULE_TARGET_STATUS_OPTIONS"
          size="sm"
          @update:model-value="(v) => (form.target_status = String(v || 'ACTIVE') as CampaignRuleTargetStatus)"
        />
        <BaseDropdown
          :model-value="form.attribution_mode"
          label="Modelo de atribuição"
          :options="RULE_ATTRIBUTION_OPTIONS"
          size="sm"
          @update:model-value="(v) => (form.attribution_mode = String(v || 'last_click') as 'last_click' | 'first_click')"
        />
      </div>

      <CampaignRuleConditionsEditor v-model="form.conditions" :is-schedule="form.rule_kind === 'schedule'" />

      <div class="action-box">
        <div class="grid-3">
          <BaseDropdown
            :model-value="form.action_type"
            label="Ação a executar"
            :options="RULE_ACTION_OPTIONS"
            size="sm"
            @update:model-value="(v) => onChangeAction(String(v || 'pause') as CampaignRuleActionType)"
          />
          <BaseInput
            v-if="showBudgetFields"
            :model-value="String(form.action_value)"
            type="number"
            :label="actionValueLabel"
            placeholder="20"
            @update:model-value="(v) => (form.action_value = Number(v || 0))"
          />
          <BaseInput
            v-if="showBudgetFields && form.action_type !== 'set_budget'"
            :model-value="String(isIncreaseAction ? form.max_budget : form.min_budget)"
            type="number"
            :label="isIncreaseAction ? 'Teto máximo diário (R$)' : 'Piso mínimo diário (R$)'"
            placeholder="0"
            @update:model-value="(v) => onUpdateBound(Number(v || 0))"
          />
        </div>

        <BaseCheckbox
          v-if="form.action_type !== 'notify_only'"
          v-model="form.dry_run"
          label="Modo simulação segura (Apenas avisar)"
          hint="Avalia e registra o que teria feito + envia alertas, sem pausar ou alterar verba na Meta/Google."
        />
      </div>

      <CampaignRuleScheduleSection v-model="form" />
    </form>

    <template #footer>
      <BaseButton variant="ghost" type="button" @click="$emit('update:modelValue', false)">Cancelar</BaseButton>
      <BaseButton variant="primary" :loading="saving" type="button" @click="onSubmit">
        {{ editingRule ? 'Salvar alterações' : 'Criar regra' }}
      </BaseButton>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import type {
  CampaignRule, CampaignRuleActionType, CampaignRuleKind, CampaignRuleLevel,
  CampaignRulePayload, CampaignRulePeriod, CampaignRuleTargetStatus,
} from '#shared/types'
import type { DropdownOption } from '~/components/BaseDropdown.vue'
import {
  cloneRulePayload, defaultRulePayload, isBudgetAction,
  RULE_ACTION_OPTIONS, RULE_ATTRIBUTION_OPTIONS, RULE_KIND_OPTIONS,
  RULE_LEVEL_OPTIONS, RULE_PERIOD_OPTIONS, RULE_PLATFORM_OPTIONS, RULE_TARGET_STATUS_OPTIONS,
} from '~/utils/campaignRuleOptions'
import CampaignRuleConditionsEditor from './CampaignRuleConditionsEditor.vue'
import CampaignRuleScheduleSection from './CampaignRuleScheduleSection.vue'

const props = defineProps<{
  modelValue: boolean
  editingRule?: CampaignRule | null
  presetPayload?: CampaignRulePayload | null
  pixelNames: string[]
  saving?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  save: [payload: CampaignRulePayload]
}>()

const form = ref<CampaignRulePayload>(defaultRulePayload())

const pixelOptions = computed<DropdownOption[]>(() => [
  { value: '', label: 'Todos os pixels da empresa' },
  ...props.pixelNames.map((name) => ({ value: name, label: name })),
])

const showBudgetFields = computed(() => isBudgetAction(form.value.action_type))
const isIncreaseAction = computed(() => ['increase_budget_pct', 'increase_budget_fixed'].includes(form.value.action_type))
const actionValueLabel = computed(() => {
  if (form.value.action_type.endsWith('_pct')) return 'Percentual (%)'
  if (form.value.action_type === 'set_budget') return 'Novo orçamento diário (R$)'
  return 'Valor fixo (R$)'
})

watch(
  () => [props.modelValue, props.editingRule, props.presetPayload] as const,
  ([open]) => {
    if (open) form.value = cloneRulePayload(props.editingRule || props.presetPayload)
  },
  { immediate: true },
)

const onChangeKind = (kind: CampaignRuleKind) => {
  form.value.rule_kind = kind
  if (kind === 'reactivate_late_sale') {
    form.value.target_status = 'PAUSED'
    form.value.action_type = 'activate'
  } else if (kind === 'schedule' && !form.value.schedule_time) {
    form.value.schedule_time = '08:00'
  }
}

const onChangeAction = (action: CampaignRuleActionType) => {
  form.value.action_type = action
  if (isBudgetAction(action) && form.value.level === 'ad') form.value.level = 'adset'
  if (action === 'activate') form.value.target_status = 'PAUSED'
  if (action === 'pause') form.value.target_status = 'ACTIVE'
}

const onUpdateBound = (v: number) => {
  if (isIncreaseAction.value) form.value.max_budget = v
  else form.value.min_budget = v
}

const onSubmit = () => emit('save', { ...form.value })
</script>

<style scoped>
.rule-form { display: flex; flex-direction: column; gap: 0.9rem; }
.grid-2 { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 0.75rem; }
.grid-3 { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 0.75rem; }
.action-box {
  display: flex; flex-direction: column; gap: 0.75rem; padding: 0.85rem 0.95rem;
  border-radius: 10px; background: var(--surface-soft-2); border: 1px solid var(--border-soft);
}
@media (max-width: 720px) { .grid-2, .grid-3 { grid-template-columns: 1fr; } }
</style>
