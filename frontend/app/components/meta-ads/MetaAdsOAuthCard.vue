<template>
  <div class="oauth-card-wrap">
    <OAuthConnectionBox
      ref="boxRef"
      title="1. Conexão com Facebook / Meta Ads (OAuth)"
      description="Autorize em uma janela rápida ou conecte com um login Facebook já autorizado neste painel para listar Contas de Anúncios, Páginas e Pixels."
      channel="meta"
      :connected="isConnected"
      :token-expired="isTokenExpired"
      :expiry-warning="expiryWarning"
      :identity-label="connectedLabel"
      :identity-sub="connectedSub"
      :avatar-url="connectedAvatar"
      disconnect-title="Desconectar Meta Ads"
      disconnect-message="Revogar a conexão OAuth desta conta Meta Ads? A Conta de Anúncios e a Página vinculadas serão limpas e os envios via OAuth ficarão pausados."
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
import type { MetaAccount } from '#shared/types'
import { useMetaAdsStore } from '~/stores/useMetaAdsStore'
import { useTenantStore } from '~/stores/useTenantStore'
import { errorMessage } from '~/utils/errorMessage'

const props = defineProps<{
  account?: MetaAccount | null
  defaultName?: string
}>()

const emit = defineEmits<{
  connected: [account: MetaAccount]
  disconnected: []
}>()

const metaApi = useMetaAccounts()
const metaAdsStore = useMetaAdsStore()
const tenantStore = useTenantStore()
const { accounts: allAccounts } = storeToRefs(metaAdsStore)
const { runOAuthPopup } = useOAuthPopup()
const toast = useToast()

const oauthLoading = ref(false)
const boxRef = ref<{ triggerDefaultAction: () => void } | null>(null)

const isTokenExpired = computed(() => {
  if (props.account?.token_expired) return true
  if (!props.account?.token_expires_at || !props.account?.oauth_connected) return false
  return new Date(props.account.token_expires_at).getTime() <= Date.now()
})

const isConnected = computed(() => Boolean(props.account?.connected && !isTokenExpired.value))

const expiryWarning = computed(() => {
  if (!props.account?.token_expires_at || isTokenExpired.value) return ''
  const days = Math.ceil((new Date(props.account.token_expires_at).getTime() - Date.now()) / 86_400_000)
  if (days > 0 && days <= 7) return days === 1 ? 'Expira em 1 dia' : `Expira em ${days} dias`
  return ''
})

const connectedLabel = computed(
  () => props.account?.oauth_name || props.account?.oauth_email || 'Perfil Meta autorizado',
)
const connectedSub = computed(() => {
  if (props.account?.oauth_email) return props.account.oauth_email
  if (isConnected.value && !props.account?.token_expires_at) return 'Conexão permanente ativa'
  return ''
})
const connectedAvatar = computed(
  () => props.account?.oauth_picture_url || props.account?.picture_url || '',
)

const reuseOptions = computed<AdAccountSelectOption[]>(() => {
  const activeTenant = Number(tenantStore.activeAccountId || 0)
  const candidates = [...allAccounts.value]
    .filter((a) => a.connected && !a.token_expired && a.id !== props.account?.id)
    .sort((a, b) => {
      const aSame = activeTenant > 0 && a.account_id === activeTenant ? 1 : 0
      const bSame = activeTenant > 0 && b.account_id === activeTenant ? 1 : 0
      if (aSame !== bSame) return bSame - aSame
      return Number(Boolean(b.ad_account_id)) - Number(Boolean(a.ad_account_id))
    })

  const seen = new Set<string>()
  const opts: AdAccountSelectOption[] = []
  for (const a of candidates) {
    const key = a.oauth_user_id || a.oauth_email || a.oauth_name || String(a.id)
    if (seen.has(key)) continue
    seen.add(key)
    opts.push({
      value: a.id,
      label: a.oauth_name || a.oauth_email || a.name,
      sublabel: a.oauth_email || a.ad_account_name || '',
      avatarUrl: a.oauth_picture_url || a.picture_url || a.ad_account_picture_url || '',
      channel: 'meta',
      badge: 'Já autorizada',
    })
  }
  return opts
})

const onConnect = async (switchAccount: boolean) => {
  const hadConnection = Boolean(props.account?.connected)
  oauthLoading.value = true
  try {
    await runOAuthPopup({
      channel: 'meta',
      providerLabel: 'Facebook / Meta Ads',
      prepare: async (popupMode) => {
        if (props.account?.id) {
          const url = await metaApi.getOAuthStartUrl(props.account.id, { switchAccount, popup: popupMode })
          return { url, accountId: props.account.id, createdTemp: false }
        }
        const created = await metaApi.create({ name: props.defaultName?.trim() || 'Meta Ads', enabled: true })
        const url = await metaApi.getOAuthStartUrl(created.id, { switchAccount, popup: popupMode })
        return { url, accountId: created.id, createdTemp: true }
      },
      onSuccess: async (payload) => {
        await metaApi.list(true)
        const fresh = await metaApi.get(payload.accountId)
        toast.success('Meta Ads conectada', payload.message ? `Perfil autorizado: ${payload.message}` : 'Conexão concluída.')
        emit('connected', fresh)
      },
      onError: async (msg, accountId, createdTemp) => {
        if (createdTemp && accountId > 0) {
          try {
            const check = await metaApi.get(accountId)
            if (!check.oauth_connected) await metaApi.remove(accountId)
          } catch {}
        }
        toast.error('Falha ao conectar Meta Ads', msg)
      },
      onManualClose: async (accountId, createdTemp) => {
        if (accountId <= 0) return
        try {
          const check = await metaApi.get(accountId)
          if (check.connected && (!hadConnection || createdTemp)) {
            await metaApi.list(true)
            toast.success('Meta Ads conectada', 'Autenticação concluída com sucesso.')
            emit('connected', check)
          } else if (createdTemp && !check.oauth_connected) {
            await metaApi.remove(accountId)
          }
        } catch {}
      },
    })
  } catch (error: unknown) {
    toast.error('Erro ao iniciar login', errorMessage(error, 'Falha ao abrir OAuth da Meta'))
  } finally {
    oauthLoading.value = false
  }
}

const onReuse = async (sourceAccountId: number) => {
  oauthLoading.value = true
  try {
    const updated = props.account?.id
      ? await metaApi.reuseOAuth(props.account.id, sourceAccountId)
      : await metaApi.create({
          name: props.defaultName?.trim() || 'Meta Ads',
          reuse_oauth_from_id: sourceAccountId,
          enabled: true,
        })
    toast.success('Conta conectada', 'Agora escolha a Conta de Anúncios e a Página padrão.')
    emit('connected', updated)
  } catch (error: unknown) {
    await metaApi.list(true).catch(() => [])
    toast.error('Erro ao usar conta existente', errorMessage(error, 'Falha ao vincular conta Meta'))
  } finally {
    oauthLoading.value = false
  }
}

const onDisconnect = async () => {
  if (!props.account?.id) return
  oauthLoading.value = true
  try {
    await metaApi.disconnect(props.account.id)
    toast.success('Desconectado', 'Conexão com a Meta revogada.')
    emit('disconnected')
  } catch (error: unknown) {
    toast.error('Erro', errorMessage(error, 'Falha ao desconectar'))
  } finally {
    oauthLoading.value = false
  }
}

defineExpose({ triggerDefaultAction: () => boxRef.value?.triggerDefaultAction() })
onMounted(() => { metaApi.list() })
</script>

<style scoped>
.oauth-card-wrap { display: contents; }
</style>
