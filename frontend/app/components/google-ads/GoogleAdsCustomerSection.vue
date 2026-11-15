<template>
  <section class="form-section">
    <header class="section-header">
      <div class="header-row">
        <div>
          <h4>2. Conta de Anúncios Google Ads</h4>
          <p class="section-hint">
            Escolha a conta de anúncios que receberá as conversões offline. Subcontas gerenciadas por MCC têm o Login ID preenchido automaticamente.
          </p>
        </div>
        <BaseButton
          v-if="oauthConnected"
          type="button"
          variant="ghost"
          size="sm"
          :loading="loadingCustomers"
          @click="$emit('refresh-customers', true)"
        >
          Atualizar lista
        </BaseButton>
      </div>
    </header>

    <BaseInput
      v-model="form.name"
      label="Nome interno da conexão"
      placeholder="Ex: Google Ads Principal"
      :error="nameError"
      required
    />

    <div class="grid-2">
      <BaseAdAccountSelect
        :model-value="selectedCustomerKey"
        label="Conta Google Ads (Customer ID)"
        :options="customerOptions"
        :placeholder="dropdownPlaceholder"
        :disabled="!oauthConnected || loadingCustomers"
        :loading="loadingCustomers"
        @update:model-value="onSelectRemoteCustomer"
      />

      <BaseInput
        v-model="form.login_customer_id"
        label="MCC (Login Customer ID, opcional)"
        placeholder="Ex: 123-456-7890"
        hint="Preenchido automaticamente para subcontas MCC."
        :disabled="!oauthConnected"
      />
    </div>

    <div v-if="isSelectedManager" class="mcc-warn">
      Esta conta é uma MCC (administradora). Se suas Ações de Conversão ficam na subconta anunciante, selecione a subconta na lista acima.
    </div>

    <div v-if="form.customer_id" class="selected-account-meta">
      <span class="meta-pill">
        Customer ID: <strong>{{ formatCustomerId(form.customer_id) }}</strong>
      </span>
      <span v-if="form.customer_name" class="meta-pill">
        Conta: <strong>{{ form.customer_name }}</strong>
      </span>
      <span v-if="form.login_customer_id" class="meta-pill mcc">
        MCC: <strong>{{ formatCustomerId(form.login_customer_id) }}</strong>
      </span>
    </div>
  </section>
</template>

<script setup lang="ts">
import type { AdAccountSelectOption, AdAccountValue } from '~/components/BaseAdAccountSelect.vue'
import type { GoogleAdsRemoteCustomer } from '#shared/types'

interface CustomerFormState {
  name: string
  customer_id: string
  customer_name: string
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
  'refresh-customers': [forceRefresh: boolean]
  'customer-selected': [customer: GoogleAdsRemoteCustomer]
}>()

const stripDashes = (s: string) => (s || '').replace(/\D/g, '')

const formatCustomerId = (raw: string) => {
  const d = stripDashes(raw)
  return d.length === 10 ? `${d.slice(0, 3)}-${d.slice(3, 6)}-${d.slice(6)}` : raw
}

const selectedCustomerKey = computed(() => stripDashes(props.form.customer_id))

const isSelectedManager = computed(() => {
  const cur = selectedCustomerKey.value
  if (!cur) return false
  return Boolean(props.remoteCustomers.find((c) => stripDashes(c.id) === cur)?.is_manager)
})

const dropdownPlaceholder = computed(() => {
  if (!props.oauthConnected) return 'Conecte o Google OAuth no passo 1 primeiro...'
  if (props.loadingCustomers) return 'Carregando contas do Google Ads...'
  if (props.remoteCustomers.length === 0) return 'Nenhuma conta encontrada (clique em Atualizar lista)'
  return 'Escolha uma conta Google Ads...'
})

const customerOptions = computed<AdAccountSelectOption[]>(() => {
  const list: AdAccountSelectOption[] = props.remoteCustomers.map((c) => {
    const fmtId = c.formatted_id || formatCustomerId(c.id)
    const parts = [fmtId]
    if (c.currency_code) parts.push(c.currency_code)
    if (c.manager_customer_id) parts.push(`MCC ${formatCustomerId(c.manager_customer_id)}`)
    return {
      value: stripDashes(c.id),
      label: c.name,
      sublabel: parts.join(' · '),
      avatarUrl: props.oauthPictureUrl || '',
      channel: 'google_ads',
      badge: c.is_manager ? 'MCC' : undefined,
    }
  })
  const cur = selectedCustomerKey.value
  if (cur && !list.some((o) => o.value === cur)) {
    list.unshift({
      value: cur,
      label: props.form.customer_name || props.form.name || 'Conta selecionada',
      sublabel: formatCustomerId(cur),
      avatarUrl: props.oauthPictureUrl || '',
      channel: 'google_ads',
    })
  }
  return list
})

const onSelectRemoteCustomer = (val: AdAccountValue) => {
  const cleanId = stripDashes(String(val ?? ''))
  const found = props.remoteCustomers.find((c) => stripDashes(c.id) === cleanId)
  if (!found) return
  props.form.customer_id = found.formatted_id || formatCustomerId(found.id)
  props.form.customer_name = found.name || ''
  props.form.login_customer_id = found.manager_customer_id ? formatCustomerId(found.manager_customer_id) : ''
  if (!props.form.name.trim() || props.form.name === 'Google Ads') {
    props.form.name = found.name
  }
  emit('customer-selected', found)
}
</script>

<style scoped>
.form-section { display: flex; flex-direction: column; gap: 0.875rem; padding-bottom: 1.25rem; border-bottom: 1px solid var(--border-faint); }
.header-row { display: flex; align-items: flex-start; justify-content: space-between; gap: 0.75rem; flex-wrap: wrap; }
.section-header h4 { font-family: var(--font-heading); font-size: var(--fs-lg); font-weight: var(--fw-semibold); color: var(--text-strong); margin: 0; }
.section-hint { font-size: var(--fs-base); color: var(--text-2); margin: 0.25rem 0 0; line-height: 1.5; max-width: 560px; }
.grid-2 { display: grid; grid-template-columns: 1fr; gap: 0.875rem; align-items: start; }
@media (min-width: 640px) { .grid-2 { grid-template-columns: 1.2fr 1fr; } }
.mcc-warn { font-size: var(--fs-sm); color: var(--warning); background: var(--warning-soft); border: 1px solid var(--warning-border); padding: 0.55rem 0.75rem; border-radius: 8px; line-height: 1.4; }
.selected-account-meta { display: flex; flex-wrap: wrap; gap: 0.5rem; }
.meta-pill { font-size: var(--fs-xs); color: var(--text-2); background: var(--surface-soft); border: 1px solid var(--border); padding: 0.25rem 0.625rem; border-radius: 999px; }
.meta-pill strong { color: var(--text-1); font-family: var(--font-mono); font-weight: var(--fw-semibold); }
.meta-pill.mcc { border-color: var(--brand-border); background: var(--brand-soft); }
</style>
