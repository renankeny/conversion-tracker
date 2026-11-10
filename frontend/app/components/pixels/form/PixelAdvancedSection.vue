<template>
  <div>
    <!-- Aba Google Ads -->
    <div v-show="activeTab === 'gads'" class="tab-content">
      <div class="section-header">
        <h4 class="section-title"><BrandIcon channel="google_ads" size="sm" :colored="true" /> Google Ads (Offline Conversions)</h4>
        <span class="badge-optional">opcional</span>
      </div>
      <p class="section-hint">
        Permite vincular este Pixel a uma conta Google Ads integrada para despachar conversões via GCLID / GBRAID / WBRAID.
      </p>

      <div class="form-field">
        <BaseAdAccountSelect
          v-model="googleAdsAccountIdSelect"
          label="Conta do Google Ads vinculada"
          :options="googleAdsAccountOptions"
          :loading="googleAdsLoading"
        />
        <p class="help-text">
          As contas listadas são aquelas configuradas em <NuxtLink to="/integracoes/google-ads" class="link">Integrações &gt; Google Ads</NuxtLink>.
        </p>
      </div>
    </div>

    <!-- Aba GA4 -->
    <div v-show="activeTab === 'ga4'" class="tab-content">
      <div class="section-header">
        <h4 class="section-title">Google Analytics 4 (GA4)</h4>
        <span class="badge-optional">opcional</span>
      </div>
      <p class="section-hint">
        Integração Server-Side via Measurement Protocol. O Measurement ID começa com "G-" e o API Secret deve ser gerado no painel do GA4 em "Data Streams".
      </p>

      <div class="form-section">
        <BaseInput v-model="form.ga4_measurement_id" label="Measurement ID" placeholder="G-XXXXXXXXXX" />
        <BaseInput v-model="form.ga4_api_secret" label="API Secret" placeholder="Seu secret key aqui" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import BrandIcon from '~/components/events/BrandIcon.vue'
import type { AdAccountSelectOption } from '~/components/BaseAdAccountSelect.vue'
import type { GoogleAdsAccount } from '#shared/types'

interface AdvancedFormState {
  google_ads_account_id: number | null
  ga4_measurement_id: string
  ga4_api_secret: string
}

const props = defineProps<{
  activeTab: string
  form: AdvancedFormState
}>()

const googleAdsApi = useGoogleAdsAccounts()
const googleAdsAccounts = ref<GoogleAdsAccount[]>([])
const googleAdsLoading = ref(false)

const formatCid = (raw: string) => {
  const d = (raw || '').replace(/\D/g, '')
  return d.length === 10 ? `${d.slice(0, 3)}-${d.slice(3, 6)}-${d.slice(6)}` : raw
}

const googleAdsAccountOptions = computed<AdAccountSelectOption[]>(() => {
  const items: AdAccountSelectOption[] = [
    { value: '0', label: 'Nenhuma (sem vínculo)', channel: 'none' },
  ]
  for (const a of googleAdsAccounts.value) {
    items.push({
      value: String(a.id),
      label: a.name,
      sublabel: a.customer_id ? formatCid(a.customer_id) : (a.oauth_email || ''),
      avatarUrl: a.picture_url || a.oauth_picture_url || '',
      channel: 'google_ads',
      badge: a.connected ? undefined : 'Pendente',
    })
  }
  return items
})

const googleAdsAccountIdSelect = computed<string>({
  get() {
    const id = props.form.google_ads_account_id
    return id != null && id > 0 ? String(id) : '0'
  },
  set(v: string) {
    const n = parseInt(String(v), 10)
    props.form.google_ads_account_id = Number.isFinite(n) && n > 0 ? n : 0
  },
})

onMounted(async () => {
  googleAdsLoading.value = true
  try {
    googleAdsAccounts.value = await googleAdsApi.list()
  } catch {
    // sem permissão / erro de rede — seletor fica com opção padrão
  } finally {
    googleAdsLoading.value = false
  }
})
</script>

<style scoped>
.tab-content { display: flex; flex-direction: column; gap: 1rem; }
.form-section { display: flex; flex-direction: column; gap: 0.5rem; padding-top: 0.5rem; border-top: 1px solid var(--border-soft); }
.section-header { display: flex; align-items: center; justify-content: space-between; gap: 0.5rem; }
.section-title { font-size: var(--fs-lg); font-weight: var(--fw-semibold); color: var(--text-strong); display: inline-flex; align-items: center; gap: 0.5rem; }
.badge-optional { font-size: var(--fs-2xs); font-weight: var(--fw-semibold); text-transform: uppercase; letter-spacing: 0.05em; color: var(--text-3); background: var(--surface-soft); border: 1px solid var(--border-soft); padding: 0.125rem 0.4rem; border-radius: 4px; }
.section-hint { font-size: var(--fs-base); color: var(--text-3); margin: 0; }
.form-field { display: flex; flex-direction: column; gap: 0.5rem; }
.help-text { font-size: var(--fs-sm); color: var(--text-3); margin: 0; }
.link { color: var(--brand); text-decoration: underline; }
</style>
