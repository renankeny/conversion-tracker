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
      @customer-selected="onCustomerSelected"
    />

    <GoogleAdsConversionMappings
      :actions="form.conversion_actions"
      :remote-actions="remoteActions"
      :loading-actions="loadingActions"
      :creating-action="creatingAction"
      :can-load-remote-actions="canLoadRemoteActions"
      @refresh-actions="loadRemoteActions"
      @create-action="onCreateRemoteAction"
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
import { useGoogleAdsAccountForm } from '~/composables/useGoogleAdsAccountForm'
import type { GoogleAdsAccount, GoogleAdsAccountInput } from '#shared/types'

const props = defineProps<{ account?: GoogleAdsAccount | null }>()
const emit = defineEmits<{
  submit: [input: GoogleAdsAccountInput]
  connected: [account: GoogleAdsAccount]
  refresh: []
}>()

const oauthRef = ref<{ triggerDefaultAction: () => void } | null>(null)
const accountRef = computed(() => props.account)

const {
  form, errors, oauthPictureUrl, remoteCustomers, loadingCustomers,
  remoteActions, loadingActions, creatingAction, oauthConnected, canLoadRemoteActions,
  loadRemoteCustomers, loadRemoteActions, onCustomerSelected, onCreateRemoteAction,
  onOAuthConnected, onOAuthDisconnected, buildValidatedInput,
} = useGoogleAdsAccountForm(
  accountRef,
  (input) => emit('submit', input),
  (acc) => emit('connected', acc),
  () => emit('refresh'),
)

const onSubmit = () => {
  if (!props.account) {
    oauthRef.value?.triggerDefaultAction()
    return
  }
  buildValidatedInput()
}

defineExpose({ submit: onSubmit })
</script>

<style scoped>
.ga-form { display: flex; flex-direction: column; gap: 1.5rem; }
.form-section { display: flex; flex-direction: column; gap: 0.75rem; }
</style>
