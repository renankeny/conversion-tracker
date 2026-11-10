<template>
  <form class="tab-content" @submit.prevent="$emit('submit')">
    <BaseDropdown
      v-if="accounts.length > 0"
      v-model="form.account_id"
      label="Empresa"
      :options="accounts.map(a => ({ value: a.id, label: a.name }))"
      hint="Empresa à qual este pixel pertence."
    />

    <BaseInput
      v-model="form.name"
      label="Nome"
      placeholder="Ex: principal"
      hint="Identificador interno (slug). Usado em /tracker.js?pixel=<nome>"
      :error="nameError"
      required
    />

    <BaseInput
      v-model="form.label"
      label="Rótulo"
      placeholder="Ex: Pixel principal de vendas"
      hint="Nome amigável exibido no painel"
    />

    <div class="form-section">
      <h4 class="section-title">Domínios autorizados</h4>
      <p class="section-hint">Sites onde o tracker roda. Separe por vírgula.</p>
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
        hint="Quando desativado, o /tracker.js retorna `pixel disabled`."
      />
    </div>

    <PixelAggregationSection
      v-model:enabled="form.aggregation_enabled"
      v-model:window-minutes="form.aggregation_window_minutes"
    />
  </form>
</template>

<script setup lang="ts">
import PixelAggregationSection from '~/components/pixels/form/PixelAggregationSection.vue'

interface BasicFormState {
  name: string
  label: string
  enabled: boolean
  account_id: number | null
  aggregation_enabled: boolean
  aggregation_window_minutes: number
}

const props = defineProps<{
  form: BasicFormState
  domainsInput: string
  nameError: string
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
    if (!props.form.account_id && activeAcc) {
      const activeId = parseInt(activeAcc, 10)
      if (accounts.value.some((a) => a.id === activeId)) {
        props.form.account_id = activeId
      }
    }
    if (!props.form.account_id && accounts.value.length > 0) {
      props.form.account_id = accounts.value[0]?.id ?? null
    }
  } catch {
    // sem permissão ou indisponível — segue sem o seletor (backend usa default)
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
