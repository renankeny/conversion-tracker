<template>
  <section class="form-section">
    <header class="section-header">
      <div class="header-row">
        <h4>2. Conta de Anúncios Google Ads</h4>
        <BaseButton
          v-if="oauthConnected"
          type="button"
          variant="ghost"
          size="sm"
          :loading="loadingCustomers"
          @click="$emit('refresh-customers')"
        >
          Atualizar contas
        </BaseButton>
      </div>
      <p class="section-hint">
        Escolha a conta de anúncios que receberá as conversões. Subcontas gerenciadas por MCC são configuradas automaticamente.
      </p>
    </header>

    <div class="grid-2">
      <BaseAdAccountSelect
        :model-value="selectedCustomerKey"
        label="Conta Google Ads"
        :options="customerOptions"
        :placeholder="dropdownPlaceholder"
        :disabled="!oauthConnected || loadingCustomers"
        :loading="loadingCustomers"
        @update:model-value="onSelectRemoteCustomer"
      />

      <BaseInput
        v-model="form.name"
        label="Nome interno da conta"
        placeholder="Ex: Google Ads — Principal"
        :error="nameError"
        hint="Preenchido automaticamente ao escolher a conta."
        required
      />
    </div>

    <div v-if="form.customer_id" class="selected-account-meta">
      <span class="meta-pill">
        Customer ID: <strong>{{ form.customer_id }}</strong>
      </span>
      <span v-if="form.login_customer_id" class="meta-pill mcc">
        MCC (Login Customer ID): <strong>{{ formatCustomerId(form.login_customer_id) }}</strong>
      </span>
    </div>
  </section>
</template>

<script setup lang="ts">
import type { AdAccountSelectOption } from '~/components/BaseAdAccountSelect.vue'
import type { GoogleAdsRemoteCustomer } from '#shared/types'

interface CustomerFormState {
  name: string
  customer_id: string
  login_customer_id: string
}

const props = defineProps<{
  form: CustomerFormState
  nameError: string
  oauthConnected?: boolean
  oauthPictureUrl?: string
  remoteCustomers: GoogleAdsRemoteCustomer[]
  loadingCustomers?: boolean
}>()

const emit = defineEmits<{
  'refresh-customers': []
  'customer-selected': [customer: GoogleAdsRemoteCustomer]
}>()

const stripDashes = (s: string) => (s || '').replace(/\D/g, '')

const formatCustomerId = (raw: string) => {
  const d = stripDashes(raw)
  if (d.length !== 10) return raw
  return `${d.slice(0, 3)}-${d.slice(3, 6)}-${d.slice(6)}`
}

const selectedCustomerKey = computed(() => stripDashes(props.form.customer_id))

const dropdownPlaceholder = computed(() => {
  if (!props.oauthConnected) return 'Conecte o Google OAuth no passo 1 primeiro...'
  if (props.loadingCustomers) return 'Carregando contas do Google Ads...'
  if (props.remoteCustomers.length === 0) return 'Nenhuma conta encontrada — clique em Atualizar contas'
  return 'Escolha uma conta Google Ads...'
})

const customerOptions = computed<AdAccountSelectOption[]>(() => {
  const list: AdAccountSelectOption[] = props.remoteCustomers.map((c) => {
    const fmtId = c.formatted_id || formatCustomerId(c.id)
    const mccSuffix = c.manager_customer_id ? ` · MCC ${formatCustomerId(c.manager_customer_id)}` : ''
    return {
      value: stripDashes(c.id),
      label: c.name,
      sublabel: `${fmtId}${mccSuffix}`,
      avatarUrl: props.oauthPictureUrl || '',
      channel: 'google_ads',
      badge: c.is_manager ? 'MCC' : undefined,
    }
  })
  const cur = selectedCustomerKey.value
  if (cur && !list.some((o) => o.value === cur)) {
    list.unshift({
      value: cur,
      label: props.form.name || 'Conta selecionada',
      sublabel: formatCustomerId(cur),
      avatarUrl: props.oauthPictureUrl || '',
      channel: 'google_ads',
    })
  }
  return list
})

const onSelectRemoteCustomer = (val: string | number) => {
  const cleanId = stripDashes(String(val))
  const found = props.remoteCustomers.find((c) => stripDashes(c.id) === cleanId)
  if (!found) return
  props.form.customer_id = found.formatted_id || formatCustomerId(found.id)
  props.form.login_customer_id = found.manager_customer_id || ''
  if (!props.form.name.trim() || props.form.name === 'Google Ads') {
    props.form.name = found.name
  }
  emit('customer-selected', found)
}
</script>

<style scoped>
.form-section { display: flex; flex-direction: column; gap: 0.75rem; padding-bottom: 1.25rem; border-bottom: 1px solid var(--border-faint); }
.header-row { display: flex; align-items: center; justify-content: space-between; gap: 0.75rem; }
.section-header h4 { font-family: var(--font-heading); font-size: var(--fs-lg); font-weight: var(--fw-semibold); color: var(--text-strong); margin: 0; }
.section-hint { font-size: var(--fs-base); color: var(--text-2); margin: 0.2rem 0 0; line-height: 1.5; }
.grid-2 { display: grid; grid-template-columns: 1fr; gap: 0.75rem; align-items: start; }
@media (min-width: 640px) { .grid-2 { grid-template-columns: 1.2fr 1fr; } }
.selected-account-meta { display: flex; flex-wrap: wrap; gap: 0.5rem; }
.meta-pill { font-size: var(--fs-xs); color: var(--text-2); background: var(--surface-soft); border: 1px solid var(--border); padding: 0.25rem 0.625rem; border-radius: 999px; }
.meta-pill strong { color: var(--text-1); font-family: var(--font-mono); font-weight: var(--fw-semibold); }
.meta-pill.mcc { border-color: var(--brand-border); background: var(--brand-soft); }
</style>
