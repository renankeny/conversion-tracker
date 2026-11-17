<template>
  <div class="meta-form">
    <MetaAdsOAuthCard
      ref="oauthRef"
      :account="account"
      :default-name="form.name"
      @connected="onOAuthConnected"
      @disconnected="onOAuthDisconnected"
    />

    <MetaAdsAssetsSection
      :form="form"
      :name-error="errors.name"
      :oauth-connected="oauthConnected"
      :oauth-picture-url="oauthPictureUrl"
      :remote-ad-accounts="remoteAdAccounts"
      :remote-pages="remotePages"
      :remote-pixels="remotePixels"
      :loading-assets="loadingAssets"
      @refresh-assets="loadRemoteAssets"
      @ad-account-selected="(id) => loadRemotePixels(id, false)"
    />

    <MetaAdsTestSection
      :account="account"
      :oauth-connected="oauthConnected"
      :ad-account-id="form.ad_account_id"
      :remote-pixels="remotePixels"
    />

    <section class="form-footer-check">
      <BaseCheckbox v-model="form.enabled">
        <strong>Conta ativa:</strong> quando desmarcada, o enriquecimento de anúncios e os envios via OAuth ficam pausados.
      </BaseCheckbox>
    </section>
  </div>
</template>

<script setup lang="ts">
import MetaAdsOAuthCard from '~/components/meta-ads/MetaAdsOAuthCard.vue'
import MetaAdsAssetsSection from '~/components/meta-ads/MetaAdsAssetsSection.vue'
import MetaAdsTestSection from '~/components/meta-ads/MetaAdsTestSection.vue'
import { errorMessage } from '~/utils/errorMessage'
import type {
  MetaAccount,
  MetaAccountInput,
  MetaRemoteAdAccount,
  MetaRemotePage,
  MetaRemotePixel,
} from '#shared/types'

const props = defineProps<{
  account?: MetaAccount | null
}>()

const emit = defineEmits<{
  submit: [input: MetaAccountInput]
  connected: [account: MetaAccount]
  refresh: []
}>()

const metaApi = useMetaAccounts()
const toast = useToast()

const form = reactive({
  name: '',
  ad_account_id: '',
  ad_account_name: '',
  ad_account_picture_url: '',
  default_page_id: '',
  default_page_name: '',
  default_page_picture_url: '',
  enabled: true,
})

const errors = reactive({ name: '' })
const oauthRef = ref<{ triggerDefaultAction: () => void } | null>(null)
const oauthPictureUrl = ref('')
const remoteAdAccounts = ref<MetaRemoteAdAccount[]>([])
const remotePages = ref<MetaRemotePage[]>([])
const remotePixels = ref<MetaRemotePixel[]>([])
const loadingAssets = ref(false)

const oauthConnected = computed(
  () => Boolean(props.account?.connected && !props.account?.token_expired),
)

const normalizeAct = (a: MetaRemoteAdAccount) => (
  a.id.startsWith('act_') ? a.id : `act_${a.account_id || a.id}`
)

const loadRemotePixels = async (adAccountId?: string, forceRefresh = false) => {
  const target = adAccountId ?? form.ad_account_id
  if (!props.account || !oauthConnected.value || !target) {
    remotePixels.value = []
    return
  }
  try {
    const res = await metaApi.listPixels(props.account.id, target, forceRefresh)
    remotePixels.value = res.pixels || []
  } catch {
    remotePixels.value = []
  }
}

const loadRemoteAssets = async (forceRefresh = false, targetAccount?: MetaAccount) => {
  const acc = targetAccount ?? props.account
  if (!acc || !acc.connected || acc.token_expired) return
  loadingAssets.value = true
  try {
    const [accRes, pageRes] = await Promise.all([
      metaApi.listAdAccounts(acc.id, forceRefresh),
      metaApi.listPages(acc.id, forceRefresh).catch(() => ({ pages: [] })),
    ])
    remoteAdAccounts.value = accRes.ad_accounts || []
    remotePages.value = pageRes.pages || []
    if (accRes.oauth_picture_url) oauthPictureUrl.value = accRes.oauth_picture_url

    let autoChanged = false
    if (!form.ad_account_id && remoteAdAccounts.value.length >= 1) {
      const pick = remoteAdAccounts.value.find((a) => a.account_status === 1) || remoteAdAccounts.value[0]!
      form.ad_account_id = normalizeAct(pick)
      form.ad_account_name = pick.name || ''
      form.ad_account_picture_url = pick.picture_url || oauthPictureUrl.value || ''
      if (!form.name.trim() || form.name === 'Meta Ads') form.name = pick.name || acc.oauth_name || 'Meta Ads'
      autoChanged = true
    } else if (form.ad_account_id && !form.ad_account_picture_url) {
      const matched = remoteAdAccounts.value.find((a) => normalizeAct(a) === form.ad_account_id)
      if (matched?.picture_url) form.ad_account_picture_url = matched.picture_url
    }

    if (!form.default_page_id && remotePages.value.length === 1) {
      form.default_page_id = remotePages.value[0]!.id
      form.default_page_name = remotePages.value[0]!.name
      form.default_page_picture_url = remotePages.value[0]!.picture_url || ''
      autoChanged = true
    }

    if (autoChanged && !acc.ad_account_id && form.ad_account_id) {
      try {
        const saved = await metaApi.update(acc.id, {
          name: form.name.trim() || 'Meta Ads', ad_account_id: form.ad_account_id,
          ad_account_name: form.ad_account_name, ad_account_picture_url: form.ad_account_picture_url,
          default_page_id: form.default_page_id, default_page_name: form.default_page_name,
          default_page_picture_url: form.default_page_picture_url, enabled: form.enabled,
        })
        emit('connected', saved)
      } catch {}
    }

    if (form.ad_account_id) {
      const pxRes = await metaApi.listPixels(acc.id, form.ad_account_id, forceRefresh).catch(() => ({ pixels: [] }))
      remotePixels.value = pxRes.pixels || []
    }
  } catch (e: unknown) {
    toast.error('Erro ao consultar Meta Graph API', errorMessage(e, 'Tente novamente'))
  } finally {
    loadingAssets.value = false
  }
}

const fillFromAccount = (a: MetaAccount) => {
  form.name = a.name
  form.ad_account_id = a.ad_account_id || ''
  form.ad_account_name = a.ad_account_name || ''
  form.ad_account_picture_url = a.ad_account_picture_url || a.picture_url || ''
  form.default_page_id = a.default_page_id || ''
  form.default_page_name = a.default_page_name || ''
  form.default_page_picture_url = a.default_page_picture_url || ''
  form.enabled = a.enabled
  oauthPictureUrl.value = a.oauth_picture_url || a.picture_url || ''
}

watch(
  () => props.account,
  async (acc) => {
    remoteAdAccounts.value = []
    remotePages.value = []
    remotePixels.value = []
    if (acc) {
      fillFromAccount(acc)
      if (acc.connected && !acc.token_expired) {
        await loadRemoteAssets(false, acc)
      }
    } else {
      oauthPictureUrl.value = ''
      Object.assign(form, {
        name: '', ad_account_id: '', ad_account_name: '', ad_account_picture_url: '',
        default_page_id: '', default_page_name: '', default_page_picture_url: '', enabled: true,
      })
    }
  },
  { immediate: true },
)

const onOAuthConnected = async (updated: MetaAccount) => {
  if (form.name.trim() && form.name !== 'Meta Ads') {
    updated = { ...updated, name: form.name.trim() }
  }
  emit('connected', updated)
  await loadRemoteAssets(true, updated)
}

const onOAuthDisconnected = () => {
  remoteAdAccounts.value = []
  remotePages.value = []
  remotePixels.value = []
  form.ad_account_id = ''
  form.ad_account_name = ''
  form.ad_account_picture_url = ''
  form.default_page_id = ''
  form.default_page_name = ''
  form.default_page_picture_url = ''
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
  emit('submit', {
    name: form.name.trim(),
    ad_account_id: form.ad_account_id.trim(),
    ad_account_name: form.ad_account_name.trim(),
    ad_account_picture_url: form.ad_account_picture_url.trim(),
    default_page_id: form.default_page_id.trim(),
    default_page_name: form.default_page_name.trim(),
    default_page_picture_url: form.default_page_picture_url.trim(),
    enabled: form.enabled,
  })
}

defineExpose({ submit: onSubmit })
</script>

<style scoped>
.meta-form { display: flex; flex-direction: column; gap: 1.5rem; }
.form-footer-check { display: flex; flex-direction: column; gap: 0.75rem; }
</style>
