<template>
  <div class="ga-form">
    <GoogleAdsOAuthCard
      ref="oauthRef"
      :account="account"
      :default-name="form.name"
      @connected="onOAuthConnected"
      @disconnected="onOAuthDisconnected"
    />

    <GoogleAdsCustomerSection
      :form="form"
      :name-error="errors.name"
      :oauth-connected="oauthConnected"
      :oauth-picture-url="oauthPictureUrl"
      :remote-customers="remoteCustomers"
      :loading-customers="loadingCustomers"
      @refresh-customers="loadRemoteCustomers"
      @customer-selected="() => loadRemoteActions(false)"
    />

    <GoogleAdsConversionMappings
      :actions="form.conversion_actions"
      :remote-actions="remoteActions"
      :loading-actions="loadingActions"
      :can-load-remote-actions="canLoadRemoteActions"
      @refresh-actions="loadRemoteActions"
    />

    <GoogleAdsTestSection
      :account="account"
      :oauth-connected="oauthConnected"
      :customer-id="form.customer_id"
      :login-customer-id="form.login_customer_id"
      :actions="form.conversion_actions"
    />

    <section class="form-section">
      <BaseCheckbox v-model="form.enabled">
        <strong>Conta ativa:</strong> quando desmarcada, o envio de conversões offline desta conta fica pausado.
      </BaseCheckbox>
    </section>
  </div>
</template>

<script setup lang="ts">
import GoogleAdsCustomerSection from '~/components/google-ads/GoogleAdsCustomerSection.vue'
import GoogleAdsOAuthCard from '~/components/google-ads/GoogleAdsOAuthCard.vue'
import GoogleAdsConversionMappings from '~/components/google-ads/GoogleAdsConversionMappings.vue'
import GoogleAdsTestSection from '~/components/google-ads/GoogleAdsTestSection.vue'
import { errorMessage } from '~/utils/errorMessage'
import type {
  GoogleAdsAccount,
  GoogleAdsAccountInput,
  GoogleAdsConversionAction,
  GoogleAdsRemoteConversionAction,
  GoogleAdsRemoteCustomer,
} from '#shared/types'

const props = defineProps<{
  account?: GoogleAdsAccount | null
}>()

const emit = defineEmits<{
  submit: [input: GoogleAdsAccountInput]
  connected: [account: GoogleAdsAccount]
  refresh: []
}>()

const googleAdsApi = useGoogleAdsAccounts()
const toast = useToast()

const form = reactive({
  name: '',
  customer_id: '',
  customer_name: '',
  login_customer_id: '',
  conversion_actions: [] as GoogleAdsConversionAction[],
  enabled: true,
})

const errors = reactive({ name: '' })
const oauthRef = ref<{ triggerDefaultAction: () => void } | null>(null)
const oauthPictureUrl = ref('')
const remoteCustomers = ref<GoogleAdsRemoteCustomer[]>([])
const loadingCustomers = ref(false)
const remoteActions = ref<GoogleAdsRemoteConversionAction[]>([])
const loadingActions = ref(false)

const isAccConnected = (acc?: GoogleAdsAccount | null) =>
  Boolean(acc?.oauth_connected || acc?.connected || acc?.oauth_email)

const oauthConnected = computed(() => isAccConnected(props.account))
const canLoadRemoteActions = computed(
  () => oauthConnected.value && Boolean(form.customer_id.replace(/\D/g, '')),
)

const defaultConversionRow = (): GoogleAdsConversionAction => ({
  event_type: 'Purchase', conversion_action_id: '', name: '', enabled: true,
})

const loadRemoteActions = async (forceRefresh = false, targetAccount?: GoogleAdsAccount) => {
  const acc = targetAccount ?? props.account
  if (!acc || !form.customer_id.replace(/\D/g, '')) return
  loadingActions.value = true
  try {
    const res = await googleAdsApi.listConversionActions(
      acc.id, form.customer_id, form.login_customer_id, forceRefresh,
    )
    remoteActions.value = res.conversion_actions || []
  } catch (e: unknown) {
    toast.error('Erro ao buscar Ações de Conversão', errorMessage(e, 'Tente novamente'))
  } finally {
    loadingActions.value = false
  }
}

const loadRemoteCustomers = async (forceRefresh = false, targetAccount?: GoogleAdsAccount) => {
  const acc = targetAccount ?? props.account
  if (!acc || !isAccConnected(acc)) return
  loadingCustomers.value = true
  try {
    const res = await googleAdsApi.listCustomers(acc.id, forceRefresh)
    remoteCustomers.value = res.customers || []
    if (res.oauth_picture_url) oauthPictureUrl.value = res.oauth_picture_url
    if (res.warning && !remoteCustomers.value.length) toast.warning('Aviso do Google Ads', res.warning)

    const nonManager = remoteCustomers.value.filter((c) => !c.is_manager)
    const autoPick = nonManager.length === 1 ? nonManager[0] : (remoteCustomers.value.length === 1 ? remoteCustomers.value[0] : undefined)
    if (!form.customer_id && autoPick) {
      form.customer_id = autoPick.formatted_id || autoPick.id
      form.customer_name = autoPick.name || ''
      form.login_customer_id = autoPick.manager_customer_id || ''
      if (!form.name.trim() || form.name === 'Google Ads') form.name = autoPick.name
    } else if (form.customer_id && !form.customer_name) {
      const clean = form.customer_id.replace(/\D/g, '')
      const found = remoteCustomers.value.find((c) => c.id.replace(/\D/g, '') === clean)
      if (found?.name) form.customer_name = found.name
    }
    if (form.customer_id) await loadRemoteActions(forceRefresh, acc)
  } catch (e: unknown) {
    toast.error('Erro ao buscar contas Google Ads', errorMessage(e, 'Tente novamente'))
  } finally {
    loadingCustomers.value = false
  }
}

const fillFromAccount = (a: GoogleAdsAccount) => {
  Object.assign(form, {
    name: a.name, customer_id: a.customer_id || '', customer_name: a.customer_name || '',
    login_customer_id: a.login_customer_id || '', enabled: a.enabled,
  })
  oauthPictureUrl.value = a.picture_url || a.oauth_picture_url || ''
  const mapped = (a.conversion_actions || []).map((ca) => ({
    event_type: ca.event_type, conversion_action_id: ca.conversion_action_id,
    name: ca.name || '', enabled: ca.enabled !== false,
  }))
  form.conversion_actions = mapped.length ? mapped : [defaultConversionRow()]
}

watch(() => props.account, async (acc) => {
  remoteCustomers.value = []
  remoteActions.value = []
  if (acc) {
    fillFromAccount(acc)
    if (isAccConnected(acc)) await loadRemoteCustomers(false, acc)
  } else {
    oauthPictureUrl.value = ''
    Object.assign(form, {
      name: '', customer_id: '', customer_name: '', login_customer_id: '',
      conversion_actions: [defaultConversionRow()], enabled: true,
    })
  }
}, { immediate: true })

const onOAuthConnected = async (updated: GoogleAdsAccount) => {
  if (form.name.trim() && form.name !== 'Google Ads') {
    updated = { ...updated, name: form.name.trim() }
  }
  emit('connected', updated)
  await loadRemoteCustomers(true, updated)
}

const onOAuthDisconnected = () => {
  remoteCustomers.value = []
  remoteActions.value = []
  form.customer_id = ''
  form.customer_name = ''
  form.login_customer_id = ''
  form.conversion_actions = [defaultConversionRow()]
  emit('refresh')
}

const onSubmit = () => {
  if (!props.account) {
    oauthRef.value?.triggerDefaultAction()
    return
  }
  errors.name = ''
  if (!form.name.trim()) {
    errors.name = 'Nome é obrigatório'
    return
  }
  const seen = new Set<string>()
  const dedupedActions = form.conversion_actions
    .map((ca) => ({
      event_type: ca.event_type,
      conversion_action_id: ca.conversion_action_id.trim(),
      name: ca.name.trim(),
      enabled: ca.enabled,
    }))
    .filter((ca) => {
      if (!ca.event_type || !ca.conversion_action_id) return false
      const key = `${ca.event_type}:${ca.conversion_action_id}`
      if (seen.has(key)) return false
      seen.add(key)
      return true
    })

  emit('submit', {
    name: form.name.trim(),
    customer_id: form.customer_id.trim(),
    customer_name: form.customer_name.trim(),
    login_customer_id: form.login_customer_id.trim(),
    conversion_actions: dedupedActions,
    enabled: form.enabled,
  })
}

defineExpose({ submit: onSubmit })
</script>

<style scoped>
.ga-form { display: flex; flex-direction: column; gap: 1.5rem; }
.form-section { display: flex; flex-direction: column; gap: 0.75rem; }
</style>
