<template>
  <form class="tab-content" @submit.prevent="$emit('submit')">
    <BaseDropdown
      v-if="!isEditing && accounts.length > 1"
      v-model="form.account_id"
      label="Empresa"
      :options="accounts.map((a) => ({ value: a.id, label: a.name }))"
      hint="Empresa que receberá este pixel e seus eventos."
    />

    <BaseInput
      v-model="form.name"
      label="Identificador (slug)"
      placeholder="Ex: principal"
      :disabled="isEditing"
      :hint="isEditing
        ? 'Identificador fixo após a criação para preservar webhooks, instâncias e histórico. Altere o Rótulo abaixo para mudar o nome exibido.'
        : 'Usado internamente em /tracker.js e integrações. Use letras, números, hífen (-) ou underline (_).'"
      :error="nameError"
      required
    />

    <BaseInput
      v-model="form.label"
      label="Rótulo de exibição"
      placeholder="Ex: Pixel principal de vendas"
      hint="Nome amigável exibido nos cards e seletores do painel. Pode ser alterado ou limpo a qualquer momento."
    />

    <div class="form-section">
      <h4 class="section-title">Domínios autorizados</h4>
      <p class="section-hint">Sites onde o script de rastreamento pode rodar. Separe múltiplos domínios por vírgula.</p>
      <BaseInput
        :model-value="domainsInput"
        placeholder="exemplo.com.br, www.exemplo.com.br"
        @update:model-value="$emit('update:domainsInput', $event)"
      />
    </div>

    <div class="form-section">
      <BaseCheckbox
        v-model="form.enabled"
        label="Pixel ativo"
        hint="Quando desativado, o script pausa a coleta e o recebimento de novos eventos."
      />
    </div>

    <PixelSalesGoalSection
      v-model:enabled="form.sales_goal_enabled"
      v-model:goal-type="form.sales_goal_type"
      v-model:target="form.sales_goal_target"
      v-model:show-on-public-link="form.public_show_sales_goal"
      :currency="form.tracker_config?.product_currency || 'BRL'"
      :error="salesGoalError"
    />

    <PixelAggregationSection
      v-model:enabled="form.aggregation_enabled"
      v-model:window-minutes="form.aggregation_window_minutes"
    />
  </form>
</template>

<script setup lang="ts">
import type { SalesGoalType, TrackerConfig } from '#shared/types'
import PixelAggregationSection from '~/components/pixels/form/PixelAggregationSection.vue'
import PixelSalesGoalSection from '~/components/pixels/form/PixelSalesGoalSection.vue'

interface BasicFormState {
  name: string
  label: string
  enabled: boolean
  account_id: number | null
  aggregation_enabled: boolean
  aggregation_window_minutes: number
  sales_goal_enabled: boolean
  sales_goal_type: SalesGoalType
  sales_goal_target: number
  public_show_sales_goal: boolean
  tracker_config?: TrackerConfig
}

const props = defineProps<{
  form: BasicFormState
  domainsInput: string
  nameError: string
  salesGoalError?: string
  isEditing: boolean
}>()

defineEmits<{
  'update:domainsInput': [value: string]
  submit: []
}>()

const accountsAPI = useAccounts()
const tenantStore = useTenantStore()
const accounts = ref<Array<{ id: number; name: string }>>([])

onMounted(async () => {
  try {
    const res = await accountsAPI.list()
    accounts.value = (res.data || []).map((a) => ({ id: a.id, name: a.name }))
    const activeAcc = tenantStore.activeAccountId
    if (!props.isEditing && !props.form.account_id && activeAcc) {
      const activeId = parseInt(activeAcc, 10)
      if (accounts.value.some((a) => a.id === activeId)) {
        props.form.account_id = activeId
      }
    }
    if (!props.isEditing && !props.form.account_id && accounts.value.length > 0) {
      props.form.account_id = accounts.value[0]?.id ?? null
    }
  } catch {
    // sem permissão de listagem global: backend utiliza a conta do usuário autenticado
  }
})
</script>

<style scoped>
.tab-content {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}
.form-section {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  padding-top: 0.5rem;
  border-top: 1px solid var(--border-soft);
}
.section-title {
  font-size: var(--fs-lg);
  font-weight: var(--fw-semibold);
  color: var(--text-strong);
}
.section-hint {
  font-size: var(--fs-base);
  color: var(--text-3);
}
</style>
