<template>
  <div class="ga-form">
    <GoogleAdsOAuthCard
      :account="account"
      @disconnected="onOAuthDisconnected"
      @reused="$emit('refresh')"
      @connect-new="onConnectNew"
      @reuse-new="onReuseNew"
    />

    <GoogleAdsCustomerSection
      :form="form"
      :name-error="errors.name"
      :oauth-connected="oauthConnected"
      :oauth-picture-url="oauthPictureUrl"
      :remote-customers="remoteCustomers"
      :loading-customers="loadingCustomers"
      @refresh-customers="loadRemoteCustomers"
      @customer-selected="loadRemoteActions"
    />

    <GoogleAdsConversionMappings
      :actions="form.conversion_actions"
      :remote-actions="remoteActions"
      :loading-actions="loadingActions"
      :can-load-remote-actions="canLoadRemoteActions"
      @refresh-actions="loadRemoteActions"
    />

    <GoogleAdsTestSection
      v-if="account"
      :account="account"
      :actions="form.conversion_actions"
    />

    <section class="form-section">
      <BaseCheckbox v-model="form.enabled">
        <strong>Conta ativa</strong> — quando desligado, nenhuma conversão
        dessa conta é enviada.
      </BaseCheckbox>
    </section>
  </div>
</template>

<script setup lang="ts">
import GoogleAdsCustomerSection from '~/components/google-ads/GoogleAdsCustomerSection.vue'
import GoogleAdsOAuthCard from '~/components/google-ads/GoogleAdsOAuthCard.vue'
import GoogleAdsConversionMappings from '~/components/google-ads/GoogleAdsConversionMappings.vue'
import GoogleAdsTestSection from '~/components/google-ads/GoogleAdsTestSection.vue'
import type {
  GoogleAdsAccount,
  GoogleAdsAccountInput,
  GoogleAdsConversionAction,
  GoogleAdsRemoteConversionAction,
  GoogleAdsRemoteCustomer,
} from '#shared/types'

interface Props {
  account?: GoogleAdsAccount | null
}
const props = defineProps<Props>()

const emit = defineEmits<{
  submit: [input: GoogleAdsAccountInput]
  'connect-new': [input: GoogleAdsAccountInput]
  'reuse-new': [input: GoogleAdsAccountInput]
  refresh: []
}>()

const googleAdsApi = useGoogleAdsAccounts()
const toast = useToast()

const form = reactive<{
  name: string
  customer_id: string
  login_customer_id: string
  conversion_actions: GoogleAdsConversionAction[]
  enabled: boolean
}>({
  name: '',
  customer_id: '',
  login_customer_id: '',
  conversion_actions: [],
  enabled: true,
})

const errors = reactive({ name: '' })
const oauthPictureUrl = ref('')
const remoteCustomers = ref<GoogleAdsRemoteCustomer[]>([])
const loadingCustomers = ref(false)
const remoteActions = ref<GoogleAdsRemoteConversionAction[]>([])
const loadingActions = ref(false)

const oauthConnected = computed(
  () => !!(props.account?.oauth_connected || props.account?.connected || props.account?.oauth_email),
)

const canLoadRemoteActions = computed(
  () => oauthConnected.value && !!form.customer_id.replace(/\D/g, ''),
)

const loadRemoteCustomers = async () => {
  if (!props.account || !oauthConnected.value) return
  loadingCustomers.value = true
  try {
    const res = await googleAdsApi.listCustomers(props.account.id)
    remoteCustomers.value = res.customers || []
    if (res.oauth_picture_url) oauthPictureUrl.value = res.oauth_picture_url
    if (res.warning && !remoteCustomers.value.length) {
      toast.warning('Aviso do Google Ads', res.warning)
    }
    if (!form.customer_id && remoteCustomers.value.length === 1) {
      const only = remoteCustomers.value[0]!
      form.customer_id = only.formatted_id || only.id
      form.login_customer_id = only.manager_customer_id || ''
      if (!form.name.trim() || form.name === 'Google Ads') {
        form.name = only.name
      }
    }
  } catch (e: any) {
    toast.error('Erro ao buscar contas Google Ads', e.message || 'Tente novamente')
  } finally {
    loadingCustomers.value = false
  }
}

const loadRemoteActions = async () => {
  if (!props.account || !canLoadRemoteActions.value) return
  loadingActions.value = true
  try {
    const res = await googleAdsApi.listConversionActions(
      props.account.id,
      form.customer_id,
      form.login_customer_id,
    )
    remoteActions.value = res.conversion_actions || []
  } catch (e: any) {
    toast.error('Erro ao buscar Conversion Actions', e.message || 'Tente novamente')
  } finally {
    loadingActions.value = false
  }
}

const defaultConversionRow = (): GoogleAdsConversionAction => ({
  event_type: 'Purchase',
  conversion_action_id: '',
  name: '',
  enabled: true,
})

const fillFromAccount = (a: GoogleAdsAccount) => {
  form.name = a.name
  form.customer_id = a.customer_id || ''
  form.login_customer_id = a.login_customer_id || ''
  form.enabled = a.enabled
  oauthPictureUrl.value = a.picture_url || a.oauth_picture_url || ''
  const mapped = (a.conversion_actions || []).map((ca) => ({
    event_type: ca.event_type,
    conversion_action_id: ca.conversion_action_id,
    name: ca.name || '',
    enabled: ca.enabled !== false,
  }))
  form.conversion_actions = mapped.length ? mapped : [defaultConversionRow()]
}

watch(
  () => props.account,
  async (acc) => {
    remoteCustomers.value = []
    remoteActions.value = []
    if (acc) {
      fillFromAccount(acc)
      if (acc.oauth_connected || acc.connected || acc.oauth_email) {
        await loadRemoteCustomers()
        if (form.customer_id) {
          await loadRemoteActions()
        }
      }
    } else {
      form.name = ''
      form.customer_id = ''
      form.login_customer_id = ''
      form.conversion_actions = [defaultConversionRow()]
      form.enabled = true
    }
  },
  { immediate: true },
)

const validate = (): boolean => {
  errors.name = ''
  if (!form.name.trim()) {
    errors.name = 'Nome é obrigatório'
    return false
  }
  return true
}

const buildPayload = (): GoogleAdsAccountInput => ({
  name: form.name.trim() || 'Google Ads',
  customer_id: form.customer_id.trim(),
  login_customer_id: form.login_customer_id.trim(),
  conversion_actions: form.conversion_actions
    .map((ca) => ({
      event_type: ca.event_type,
      conversion_action_id: ca.conversion_action_id.trim(),
      name: ca.name.trim(),
      enabled: ca.enabled,
    }))
    .filter((ca) => ca.event_type && ca.conversion_action_id),
  enabled: form.enabled,
})

const onOAuthDisconnected = () => {
  remoteCustomers.value = []
  remoteActions.value = []
  emit('refresh')
}

const onSubmit = () => {
  if (!validate()) return
  emit('submit', buildPayload())
}

const onConnectNew = () => {
  if (!form.name.trim()) form.name = 'Google Ads'
  emit('connect-new', buildPayload())
}

const onReuseNew = (sourceAccountId: number) => {
  if (!form.name.trim()) form.name = 'Google Ads'
  emit('reuse-new', { ...buildPayload(), reuse_oauth_from_id: sourceAccountId })
}

defineExpose({ submit: onSubmit })
</script>

<style scoped>
.ga-form { display: flex; flex-direction: column; gap: 1.5rem; }
.form-section { display: flex; flex-direction: column; gap: 0.75rem; }
</style>
