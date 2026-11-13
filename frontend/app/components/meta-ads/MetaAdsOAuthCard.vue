<template>
  <section class="form-section">
    <header class="section-header">
      <h4>1. Conexão com Facebook / Meta Ads (OAuth)</h4>
      <p class="section-hint">
        Autorize sua conta Meta ou reutilize um login já conectado neste painel para listar suas Contas de Anúncios, Páginas e Pixels.
      </p>
    </header>

    <div v-if="oauthConnected" class="oauth-box">
      <div class="oauth-info">
        <div class="oauth-avatar">
          <img
            v-if="connectedAvatar && !brokenAvatar"
            :src="connectedAvatar"
            alt=""
            referrerpolicy="no-referrer"
            class="oauth-avatar-img"
            @error="brokenAvatar = true"
          />
          <span v-else class="oauth-initials">{{ initials(connectedLabel) }}</span>
        </div>
        <span class="oauth-pill">Meta Conectada</span>
        <span class="oauth-identity">{{ connectedLabel }}</span>
      </div>
      <div class="oauth-actions">
        <BaseButton type="button" variant="ghost" size="sm" :loading="oauthLoading" @click="onConnect(true)">
          Entrar em outra conta
        </BaseButton>
        <BaseButton type="button" variant="ghost" size="sm" :loading="oauthLoading" @click="showDisconnectConfirm = true">
          Desconectar
        </BaseButton>
      </div>
    </div>

    <div v-else-if="reusableProfiles.length" class="reuse-container">
      <BaseAdAccountSelect
        v-if="reusableProfiles.length > 1"
        v-model="selectedSourceId"
        label="Conta Facebook já conectada neste painel"
        :options="reuseOptions"
      />
      <div v-else class="oauth-box">
        <div class="oauth-info">
          <div class="oauth-avatar">
            <img
              v-if="singleProfileAvatar && !brokenReuseAvatar"
              :src="singleProfileAvatar"
              alt=""
              referrerpolicy="no-referrer"
              class="oauth-avatar-img"
              @error="brokenReuseAvatar = true"
            />
            <span v-else class="oauth-initials">{{ initials(singleProfileLabel) }}</span>
          </div>
          <div class="oauth-stack">
            <span class="oauth-identity">{{ singleProfileLabel }}</span>
            <span class="oauth-sub">Perfil Facebook já autorizado neste painel</span>
          </div>
        </div>
        <div class="oauth-actions">
          <BaseButton type="button" variant="primary" size="sm" :loading="oauthLoading" @click="onUseExisting">
            Usar mesma conta
          </BaseButton>
          <BaseButton type="button" variant="ghost" size="sm" :loading="oauthLoading" @click="onConnect(true)">
            Entrar em outra conta
          </BaseButton>
        </div>
      </div>

      <div v-if="reusableProfiles.length > 1" class="oauth-actions-bar">
        <BaseButton type="button" variant="primary" size="sm" :loading="oauthLoading" @click="onUseExisting">
          Usar conta selecionada
        </BaseButton>
        <BaseButton type="button" variant="ghost" size="sm" :loading="oauthLoading" @click="onConnect(true)">
          Entrar em outra conta
        </BaseButton>
      </div>
    </div>

    <div v-else class="oauth-box">
      <p class="oauth-text">
        Conecte sua conta Meta para listar Contas de Anúncios, Páginas e Pixels automaticamente.
      </p>
      <BaseButton type="button" variant="primary" size="sm" :loading="oauthLoading" @click="onConnect(false)">
        Conectar com Facebook
      </BaseButton>
    </div>

    <ConfirmModal
      v-model="showDisconnectConfirm"
      title="Desconectar Meta Ads"
      message="Revogar a conexão OAuth desta conta Meta Ads? O enriquecimento automático de anúncios e gastos será pausado."
      confirm-label="Desconectar"
      variant="danger"
      :loading="oauthLoading"
      @confirm="executeDisconnect"
    />
  </section>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import type { AdAccountSelectOption } from '~/components/BaseAdAccountSelect.vue'
import type { MetaAccount } from '#shared/types'
import { useMetaAdsStore } from '~/stores/useMetaAdsStore'
import { errorMessage } from '~/utils/errorMessage'

const props = defineProps<{
  account?: MetaAccount | null
}>()

const emit = defineEmits<{
  disconnected: []
  reused: []
  'connect-new': [switchAccount: boolean]
  'reuse-new': [sourceAccountId: number]
}>()

const metaApi = useMetaAccounts()
const metaAdsStore = useMetaAdsStore()
const { accounts: allAccounts } = storeToRefs(metaAdsStore)
const toast = useToast()

const oauthLoading = ref(false)
const showDisconnectConfirm = ref(false)
const brokenAvatar = ref(false)
const brokenReuseAvatar = ref(false)
const selectedSourceId = ref<number>(0)

const oauthConnected = computed(() => !!(props.account?.oauth_connected || props.account?.connected))
const connectedLabel = computed(() => props.account?.oauth_name || props.account?.oauth_email || 'Perfil Meta autorizado')
const connectedAvatar = computed(() => props.account?.oauth_picture_url || props.account?.picture_url || '')

const reusableProfiles = computed<MetaAccount[]>(() => {
  const seen = new Set<string>()
  const out: MetaAccount[] = []
  for (const a of allAccounts.value) {
    if (!a.connected || a.id === props.account?.id) continue
    const key = a.oauth_user_id || a.oauth_email || a.oauth_name || String(a.id)
    if (seen.has(key)) continue
    seen.add(key)
    out.push(a)
  }
  return out
})

watch(reusableProfiles, (list) => {
  if (list.length && !list.some((a) => a.id === selectedSourceId.value)) {
    selectedSourceId.value = list[0]!.id
  }
}, { immediate: true })

const singleProfileLabel = computed(() => {
  const p = reusableProfiles.value[0]
  return p?.oauth_name || p?.oauth_email || p?.name || 'Perfil Meta'
})

const singleProfileAvatar = computed(() => {
  const p = reusableProfiles.value[0]
  return p?.oauth_picture_url || p?.picture_url || p?.ad_account_picture_url || ''
})

const reuseOptions = computed<AdAccountSelectOption[]>(() =>
  reusableProfiles.value.map((a) => ({
    value: a.id,
    label: a.oauth_name || a.oauth_email || a.name,
    sublabel: a.oauth_email || a.ad_account_name || '',
    avatarUrl: a.oauth_picture_url || a.picture_url || a.ad_account_picture_url || '',
    channel: 'meta',
    badge: 'Já autorizada',
  })),
)

const initials = (name: string) => {
  const parts = (name || '').trim().split(/\s+/).filter(Boolean)
  if (!parts.length) return 'FB'
  return parts.length === 1 ? parts[0]!.slice(0, 2).toUpperCase() : (parts[0]![0]! + parts[1]![0]!).toUpperCase()
}

const onUseExisting = async () => {
  const srcId = selectedSourceId.value || reusableProfiles.value[0]?.id || 0
  if (!srcId) return
  if (!props.account) {
    emit('reuse-new', srcId)
    return
  }
  oauthLoading.value = true
  try {
    await metaApi.reuseOAuth(props.account.id, srcId)
    toast.success('Conta vinculada', 'Conexão Meta reutilizada com sucesso.')
    emit('reused')
  } catch (error: unknown) {
    toast.error('Erro', errorMessage(error, 'Falha ao reutilizar conta Meta'))
  } finally {
    oauthLoading.value = false
  }
}

const onConnect = async (switchAccount: boolean) => {
  if (!props.account) {
    emit('connect-new', switchAccount)
    return
  }
  oauthLoading.value = true
  try {
    await metaApi.startOAuth(props.account.id, switchAccount)
  } catch (error: unknown) {
    toast.error('Erro', errorMessage(error, 'Falha ao iniciar OAuth da Meta'))
    oauthLoading.value = false
  }
}

const executeDisconnect = async () => {
  if (!props.account) return
  oauthLoading.value = true
  try {
    await metaApi.disconnect(props.account.id)
    showDisconnectConfirm.value = false
    toast.success('Desconectado', 'Conexão com a Meta revogada.')
    emit('disconnected')
  } catch (error: unknown) {
    toast.error('Erro', errorMessage(error, 'Falha ao desconectar'))
  } finally {
    oauthLoading.value = false
  }
}

onMounted(() => { metaApi.list() })
</script>

<style scoped>
.form-section { display: flex; flex-direction: column; gap: 0.75rem; padding-bottom: 1.25rem; border-bottom: 1px solid var(--border-faint); }
.section-header h4 { font-family: var(--font-heading); font-size: var(--fs-lg); font-weight: var(--fw-semibold); color: var(--text-strong); margin: 0; }
.section-hint { font-size: var(--fs-base); color: var(--text-2); margin: 0.25rem 0 0; line-height: 1.5; }
.reuse-container { display: flex; flex-direction: column; gap: 0.65rem; }
.oauth-box { display: flex; align-items: center; justify-content: space-between; gap: 0.75rem; padding: 0.875rem; background: var(--surface-soft); border: 1px solid var(--border); border-radius: 8px; flex-wrap: wrap; }
.oauth-info, .oauth-actions, .oauth-actions-bar { display: flex; align-items: center; gap: 0.55rem; flex-wrap: wrap; }
.oauth-avatar { width: 32px; height: 32px; border-radius: 8px; background: var(--brand-soft); border: 1px solid var(--brand-border); display: flex; align-items: center; justify-content: center; overflow: hidden; flex-shrink: 0; }
.oauth-avatar-img { width: 100%; height: 100%; object-fit: cover; display: block; }
.oauth-initials { font-size: var(--fs-xs); font-weight: var(--fw-bold); color: var(--brand); }
.oauth-stack { display: flex; flex-direction: column; line-height: 1.25; }
.oauth-sub { font-size: var(--fs-xs); color: var(--text-3); }
.oauth-pill { font-size: var(--fs-xs); font-weight: var(--fw-semibold); padding: 0.2rem 0.55rem; border-radius: 999px; background: var(--success-soft); color: var(--success); }
.oauth-identity { font-size: var(--fs-md); color: var(--text-1); font-weight: var(--fw-medium); }
.oauth-text { font-size: var(--fs-md); color: var(--text-2); margin: 0; }
</style>
