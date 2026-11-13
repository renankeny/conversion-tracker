<template>
  <div class="meta-form">
    <MetaAdsOAuthCard
      :account="account"
      @disconnected="onDisconnected"
      @reused="$emit('refresh')"
      @connect-new="onConnectNew"
      @reuse-new="onReuseNew"
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
      @ad-account-selected="loadRemotePixels"
    />

    <section class="form-footer-check">
      <BaseCheckbox v-model="form.enabled">
        <strong>Conta ativa</strong> — quando desmarcada, o enriquecimento de anúncios e envios via OAuth ficam pausados.
      </BaseCheckbox>
    </section>
  </div>
</template>

<script setup lang="ts">
import MetaAdsOAuthCard from '~/components/meta-ads/MetaAdsOAuthCard.vue'
import MetaAdsAssetsSection from '~/components/meta-ads/MetaAdsAssetsSection.vue'
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
  'connect-new': [payload: { input: MetaAccountInput; switchAccount: boolean }]
  'reuse-new': [input: MetaAccountInput]
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
const oauthPictureUrl = ref('')
const remoteAdAccounts = ref<MetaRemoteAdAccount[]>([])
const remotePages = ref<MetaRemotePage[]>([])
const remotePixels = ref<MetaRemotePixel[]>([])
const loadingAssets = ref(false)

const oauthConnected = computed(
  () => !!(props.account?.oauth_connected || props.account?.connected),
)

const loadRemotePixels = async (adAccountId?: string) => {
  const target = adAccountId ?? form.ad_account_id
  if (!props.account || !oauthConnected.value || !target) {
    remotePixels.value = []
    return
  }
  try {
    const res = await metaApi.listPixels(props.account.id, target)
    remotePixels.value = res.pixels || []
  } catch {
    remotePixels.value = []
  }
}

const loadRemoteAssets = async () => {
  if (!props.account || !oauthConnected.value) return
  loadingAssets.value = true
  try {
    const [accRes, pageRes] = await Promise.all([
      metaApi.listAdAccounts(props.account.id),
      metaApi.listPages(props.account.id).catch(() => ({ pages: [] })),
    ])
    remoteAdAccounts.value = accRes.ad_accounts || []
    remotePages.value = pageRes.pages || []
    if (accRes.oauth_picture_url) {
      oauthPictureUrl.value = accRes.oauth_picture_url
    }

    if (!form.ad_account_id && remoteAdAccounts.value.length === 1) {
      const only = remoteAdAccounts.value[0]!
      form.ad_account_id = only.id.startsWith('act_') ? only.id : `act_${only.account_id || only.id}`
      form.ad_account_name = only.name || ''
      form.ad_account_picture_url = only.picture_url || oauthPictureUrl.value || ''
      if (!form.name.trim() || form.name === 'Meta Ads') {
        form.name = only.name || 'Meta Ads'
      }
    } else if (form.ad_account_id && !form.ad_account_picture_url) {
      const matched = remoteAdAccounts.value.find((a) => (a.id.startsWith('act_') ? a.id : `act_${a.account_id}`) === form.ad_account_id)
      if (matched?.picture_url) form.ad_account_picture_url = matched.picture_url
    }
    if (!form.default_page_id && remotePages.value.length === 1) {
      form.default_page_id = remotePages.value[0]!.id
      form.default_page_name = remotePages.value[0]!.name
      form.default_page_picture_url = remotePages.value[0]!.picture_url || ''
    }
    if (form.ad_account_id) {
      await loadRemotePixels(form.ad_account_id)
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
      if (acc.oauth_connected || acc.connected) {
        await loadRemoteAssets()
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

const buildPayload = (): MetaAccountInput => ({
  name: form.name.trim() || 'Meta Ads',
  ad_account_id: form.ad_account_id.trim(),
  ad_account_name: form.ad_account_name.trim(),
  ad_account_picture_url: form.ad_account_picture_url.trim(),
  default_page_id: form.default_page_id.trim(),
  default_page_name: form.default_page_name.trim(),
  default_page_picture_url: form.default_page_picture_url.trim(),
  enabled: form.enabled,
})

const onDisconnected = () => {
  remoteAdAccounts.value = []
  remotePages.value = []
  remotePixels.value = []
  emit('refresh')
}

const onSubmit = () => {
  errors.name = ''
  if (!form.name.trim()) {
    errors.name = 'Nome é obrigatório'
    return
  }
  emit('submit', buildPayload())
}

const onConnectNew = (switchAccount = false) => {
  if (!form.name.trim()) form.name = 'Meta Ads'
  emit('connect-new', { input: buildPayload(), switchAccount })
}

const onReuseNew = (sourceAccountId: number) => {
  if (!form.name.trim()) form.name = 'Meta Ads'
  emit('reuse-new', { ...buildPayload(), reuse_oauth_from_id: sourceAccountId })
}

defineExpose({ submit: onSubmit })
</script>

<style scoped>
.meta-form { display: flex; flex-direction: column; gap: 1.5rem; }
.form-footer-check { display: flex; flex-direction: column; gap: 0.75rem; }
</style>
