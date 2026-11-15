<template>
  <div class="tab-content">
    <div class="section-header">
      <h4 class="section-title">
        <BrandIcon channel="google_ads" size="sm" :colored="true" />
        Google Ads
      </h4>
      <span class="badge-optional">opcional</span>
    </div>
    <p class="section-hint">
      Escolha a Conta Google (OAuth) e a Conta de Anúncios Google Ads específica deste Pixel para acompanhar os resultados dessa conta na Dashboard e, quando houver Ações de Conversão configuradas, enviar conversões offline.
    </p>

    <div class="oauth-link-box">
      <BaseAdAccountSelect
        :model-value="selectedOAuthKey"
        label="1. Conta Google (OAuth)"
        :options="oauthAccountOptions"
        :loading="loadingAccounts"
        @update:model-value="onSelectOAuthAccount"
      />

      <div v-if="selectedOAuthKey !== '0'" class="customer-step">
        <div class="step-label-row">
          <span class="step-caption">2. Conta de Anúncios Google Ads</span>
          <BaseButton
            v-if="oauthConnected"
            type="button"
            variant="ghost"
            size="sm"
            :loading="loadingCustomers"
            @click="loadCustomers(true, false)"
          >
            Atualizar lista
          </BaseButton>
        </div>

        <BaseAdAccountSelect
          :model-value="selectedCustomerCleanId"
          :options="customerOptions"
          :placeholder="customerPlaceholder"
          :disabled="!oauthConnected || loadingCustomers"
          :loading="loadingCustomers"
          @update:model-value="onSelectCustomer"
        />

        <div v-if="isSelectedManager" class="mcc-warn">
          Esta conta é uma MCC (administradora). Para ver os resultados e métricas de uma conta anunciante específica na Dashboard, selecione a subconta na lista acima.
        </div>

        <div v-if="form.google_ads_customer_id" class="selected-meta">
          <span class="meta-pill">
            Customer ID: <strong>{{ formatGoogleCid(form.google_ads_customer_id) }}</strong>
          </span>
          <span v-if="form.google_ads_login_customer_id" class="meta-pill mcc">
            MCC: <strong>{{ formatGoogleCid(form.google_ads_login_customer_id) }}</strong>
          </span>
          <span v-if="hasOfflineConversions" class="meta-pill ready">
            Resultados na Dashboard + Conversões offline
          </span>
          <span v-else class="meta-pill insights-only">
            Resultados na Dashboard ativos (sem Ações de Conversão offline)
          </span>
        </div>
      </div>

      <p class="help-text">
        Conecte contas Google e configure Ações de Conversão offline em
        <NuxtLink to="/integracoes/google-ads" class="link">Integrações &gt; Google Ads</NuxtLink>.
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import BrandIcon from '~/components/events/BrandIcon.vue'
import { formatGoogleCid } from '~/utils/googleAdsOptions'
import {
  usePixelGoogleAdsTab,
  type GoogleAdsTabFormState,
} from '~/composables/usePixelGoogleAdsTab'

const props = defineProps<{
  form: GoogleAdsTabFormState
}>()

const {
  loadingAccounts,
  loadingCustomers,
  selectedOAuthKey,
  selectedCustomerCleanId,
  oauthAccountOptions,
  customerOptions,
  activeOAuthAccount,
  isSelectedManager,
  hasOfflineConversions,
  onSelectOAuthAccount,
  onSelectCustomer,
  loadCustomers,
} = usePixelGoogleAdsTab(props)

const oauthConnected = computed(() =>
  Boolean(activeOAuthAccount.value?.oauth_connected || activeOAuthAccount.value?.connected),
)

const customerPlaceholder = computed(() => {
  if (!oauthConnected.value) return 'Reconecte o OAuth desta Conta Google em Integrações > Google Ads'
  if (loadingCustomers.value) return 'Carregando contas do Google Ads...'
  if (!customerOptions.value.length) return 'Nenhuma conta encontrada (clique em Atualizar lista)'
  return 'Escolha qual conta Google Ads vincular a este Pixel...'
})
</script>

<style scoped>
.tab-content { display: flex; flex-direction: column; gap: 1rem; }
.section-header { display: flex; align-items: center; justify-content: space-between; gap: 0.5rem; }
.section-title { font-size: var(--fs-lg); font-weight: var(--fw-semibold); color: var(--text-strong); display: inline-flex; align-items: center; gap: 0.5rem; margin: 0; }
.badge-optional { font-size: var(--fs-2xs); font-weight: var(--fw-semibold); text-transform: uppercase; letter-spacing: 0.05em; color: var(--text-3); background: var(--surface-soft); border: 1px solid var(--border-soft); padding: 0.125rem 0.4rem; border-radius: 4px; }
.section-hint { font-size: var(--fs-base); color: var(--text-3); margin: 0; line-height: 1.45; }
.oauth-link-box { display: flex; flex-direction: column; gap: 0.875rem; padding: 0.875rem; background: var(--surface-soft); border: 1px solid var(--border-soft); border-radius: 10px; }
.customer-step { display: flex; flex-direction: column; gap: 0.5rem; padding-top: 0.75rem; border-top: 1px solid var(--border-faint); }
.step-label-row { display: flex; align-items: center; justify-content: space-between; gap: 0.5rem; flex-wrap: wrap; }
.step-caption { font-size: var(--fs-sm); font-weight: var(--fw-medium); color: var(--text-1); }
.mcc-warn { font-size: var(--fs-sm); color: var(--warning); background: var(--warning-soft); border: 1px solid var(--warning-border); padding: 0.5rem 0.75rem; border-radius: 8px; line-height: 1.4; }
.selected-meta { display: flex; flex-wrap: wrap; gap: 0.4rem; margin-top: 0.125rem; }
.meta-pill { font-size: var(--fs-xs); color: var(--text-2); background: var(--surface-2); border: 1px solid var(--border); padding: 0.2rem 0.55rem; border-radius: 999px; }
.meta-pill strong { color: var(--text-1); font-family: var(--font-mono); font-weight: var(--fw-semibold); }
.meta-pill.mcc { border-color: var(--brand-border); background: var(--brand-soft); }
.meta-pill.ready { color: var(--success); border-color: var(--success); background: var(--success-soft); font-weight: var(--fw-medium); }
.meta-pill.insights-only { color: var(--brand); border-color: var(--brand-border); background: var(--brand-soft); font-weight: var(--fw-medium); }
.help-text { font-size: var(--fs-sm); color: var(--text-3); margin: 0; line-height: 1.45; }
.link { color: var(--brand); text-decoration: underline; }
</style>
