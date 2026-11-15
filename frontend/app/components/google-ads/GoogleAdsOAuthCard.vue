<template>
  <div class="oauth-card-wrap">
    <OAuthConnectionBox
      ref="boxRef"
      title="1. Conexão com o Google Ads (OAuth)"
      description="Autorize em uma janela rápida ou conecte com um login Google já autorizado neste painel para carregar Contas de Anúncios e Ações de Conversão."
      channel="google_ads"
      :connected="oauthConnected"
      :identity-label="connectedLabel"
      :avatar-url="connectedAvatar"
      disconnect-title="Desconectar Google Ads"
      disconnect-message="Revogar a conexão OAuth desta conta Google Ads? O Customer ID e as Ações de Conversão vinculadas serão limpos e os envios offline pararão."
      :reuse-options="reuseOptions"
      :loading="oauthLoading"
      @connect="onConnect"
      @reuse="onReuse"
      @disconnect="onDisconnect"
    />
  </div>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import OAuthConnectionBox from '~/components/integrations/OAuthConnectionBox.vue'
import type { AdAccountSelectOption } from '~/components/BaseAdAccountSelect.vue'
import type { GoogleAdsAccount } from '#shared/types'
import { useGoogleAdsStore } from '~/stores/useGoogleAdsStore'
import { useTenantStore } from '~/stores/useTenantStore'
import { errorMessage } from '~/utils/errorMessage'

const props = defineProps<{
  account?: GoogleAdsAccount | null
  defaultName?: string
}>()

const emit = defineEmits<{
  connected: [account: GoogleAdsAccount]
  disconnected: []
}>()

const googleAdsApi = useGoogleAdsAccounts()
const googleAdsStore = useGoogleAdsStore()
const tenantStore = useTenantStore()
const { accounts: allAccounts } = storeToRefs(googleAdsStore)
const { runOAuthPopup } = useOAuthPopup()
const toast = useToast()

const oauthLoading = ref(false)
const boxRef = ref<{ triggerDefaultAction: () => void } | null>(null)

const oauthConnected = computed(
  () => Boolean(props.account?.oauth_connected || props.account?.connected || props.account?.oauth_email),
)
const connectedLabel = computed(() => props.account?.oauth_email || 'Conta Google autorizada')
const connectedAvatar = computed(
  () => props.account?.picture_url || props.account?.oauth_picture_url || '',
)

const reuseOptions = computed<AdAccountSelectOption[]>(() => {
  const activeTenant = Number(tenantStore.activeAccountId || 0)
  const candidates = [...allAccounts.value]
    .filter((a) => (a.connected || a.oauth_connected) && a.id !== props.account?.id)
    .sort((a, b) => {
      const aSame = activeTenant > 0 && a.account_id === activeTenant ? 1 : 0
      const bSame = activeTenant > 0 && b.account_id === activeTenant ? 1 : 0
      if (aSame !== bSame) return bSame - aSame
      return Number(Boolean(b.connected)) - Number(Boolean(a.connected))
    })

  const seen = new Set<string>()
  const opts: AdAccountSelectOption[] = []
  for (const a of candidates) {
    const key = (a.oauth_email || String(a.id)).toLowerCase()
    if (seen.has(key)) continue
    seen.add(key)
    opts.push({
      value: a.id,
      label: a.oauth_email || a.name,
      sublabel: a.customer_name || a.name,
      avatarUrl: a.picture_url || a.oauth_picture_url || '',
      channel: 'google_ads',
      badge: 'Já autorizada',
    })
  }
  return opts
})

const onConnect = async (switchAccount: boolean) => {
  const hadConnection = oauthConnected.value
  oauthLoading.value = true
  try {
    await runOAuthPopup({
      channel: 'google_ads',
      providerLabel: 'Google Ads',
      prepare: async (popupMode) => {
        if (props.account?.id) {
          const url = await googleAdsApi.getOAuthStartUrl(props.account.id, { switchAccount, popup: popupMode })
          return { url, accountId: props.account.id, createdTemp: false }
        }
        const created = await googleAdsApi.create({ name: props.defaultName?.trim() || 'Google Ads', enabled: true })
        const url = await googleAdsApi.getOAuthStartUrl(created.id, { switchAccount, popup: popupMode })
        return { url, accountId: created.id, createdTemp: true }
      },
      onSuccess: async (payload) => {
        await googleAdsApi.list(true)
        const fresh = await googleAdsApi.get(payload.accountId)
        toast.success('Google Ads conectado', payload.message ? `Conta autorizada: ${payload.message}` : 'Conexão concluída.')
        emit('connected', fresh)
      },
      onError: async (msg, accountId, createdTemp) => {
        if (createdTemp && accountId > 0) {
          try {
            const check = await googleAdsApi.get(accountId)
            if (!check.oauth_connected && !check.connected) await googleAdsApi.remove(accountId)
          } catch {}
        }
        toast.error('Falha ao conectar Google Ads', msg)
      },
      onManualClose: async (accountId, createdTemp) => {
        if (accountId <= 0) return
        try {
          const check = await googleAdsApi.get(accountId)
          const nowConnected = Boolean(check.oauth_connected || check.connected)
          if (nowConnected && (!hadConnection || createdTemp)) {
            await googleAdsApi.list(true)
            toast.success('Google Ads conectado', 'Autenticação concluída com sucesso.')
            emit('connected', check)
          } else if (createdTemp && !nowConnected) {
            await googleAdsApi.remove(accountId)
          }
        } catch {}
      },
    })
  } catch (error: unknown) {
    toast.error('Erro ao iniciar login', errorMessage(error, 'Falha ao abrir OAuth do Google Ads'))
  } finally {
    oauthLoading.value = false
  }
}

const onReuse = async (sourceAccountId: number) => {
  oauthLoading.value = true
  try {
    const updated = props.account?.id
      ? await googleAdsApi.reuseOAuth(props.account.id, sourceAccountId)
      : await googleAdsApi.create({
          name: props.defaultName?.trim() || 'Google Ads',
          reuse_oauth_from_id: sourceAccountId,
          enabled: true,
        })
    toast.success('Conta conectada', 'Agora escolha a Conta Google Ads e as Ações de Conversão.')
    emit('connected', updated)
  } catch (error: unknown) {
    await googleAdsApi.list(true).catch(() => [])
    toast.error('Erro ao usar conta existente', errorMessage(error, 'Falha ao vincular conta Google'))
  } finally {
    oauthLoading.value = false
  }
}

const onDisconnect = async () => {
  if (!props.account?.id) return
  oauthLoading.value = true
  try {
    await googleAdsApi.disconnect(props.account.id)
    toast.success('Desconectado', 'Conexão com o Google Ads revogada.')
    emit('disconnected')
  } catch (error: unknown) {
    toast.error('Erro', errorMessage(error, 'Falha ao desconectar'))
  } finally {
    oauthLoading.value = false
  }
}

defineExpose({ triggerDefaultAction: () => boxRef.value?.triggerDefaultAction() })
onMounted(() => { googleAdsApi.list() })
</script>

<style scoped>
.oauth-card-wrap { display: contents; }
</style>
