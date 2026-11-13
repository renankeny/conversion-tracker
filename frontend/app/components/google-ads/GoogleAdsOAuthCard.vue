<template>
  <section class="form-section">
    <header class="section-header">
      <h4>1. Conexão com o Google Ads (OAuth)</h4>
      <p class="section-hint">
        Autorize uma conta Google ou reutilize um login já conectado neste painel para carregar Contas de Anúncios (inclusive MCC) e Ações de Conversão.
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
        <span class="oauth-pill">Google Conectado</span>
        <span class="oauth-email">{{ connectedLabel }}</span>
      </div>
      <div class="oauth-actions">
        <BaseButton type="button" variant="ghost" size="sm" :loading="oauthLoading" @click="onConnect">
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
        label="Conta Google já conectada neste painel"
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
            <span class="oauth-email">{{ singleProfileLabel }}</span>
            <span class="oauth-sub">Conta Google já autorizada neste painel</span>
          </div>
        </div>
        <div class="oauth-actions">
          <BaseButton type="button" variant="primary" size="sm" :loading="oauthLoading" @click="onUseExisting">
            Usar mesma conta
          </BaseButton>
          <BaseButton type="button" variant="ghost" size="sm" :loading="oauthLoading" @click="onConnect">
            Entrar em outra conta
          </BaseButton>
        </div>
      </div>

      <div v-if="reusableProfiles.length > 1" class="oauth-actions-bar">
        <BaseButton type="button" variant="primary" size="sm" :loading="oauthLoading" @click="onUseExisting">
          Usar conta selecionada
        </BaseButton>
        <BaseButton type="button" variant="ghost" size="sm" :loading="oauthLoading" @click="onConnect">
          Entrar em outra conta
        </BaseButton>
      </div>
    </div>

    <div v-else class="oauth-box">
      <p class="oauth-text">
        Conecte sua conta Google em 1 clique para importar suas contas e conversões.
      </p>
      <BaseButton type="button" variant="primary" size="sm" :loading="oauthLoading" @click="onConnect">
        Conectar com Google
      </BaseButton>
    </div>

    <ConfirmModal
      v-model="showDisconnectConfirm"
      title="Desconectar Google Ads"
      message="Desconectar essa conta Google Ads? As conversões pararão de ser enviadas."
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
import type { GoogleAdsAccount } from '#shared/types'
import { useGoogleAdsStore } from '~/stores/useGoogleAdsStore'
import { errorMessage } from '~/utils/errorMessage'

const props = defineProps<{
  account?: GoogleAdsAccount | null
}>()

const emit = defineEmits<{
  disconnected: []
  reused: []
  'connect-new': []
  'reuse-new': [sourceAccountId: number]
}>()

const googleAdsApi = useGoogleAdsAccounts()
const googleAdsStore = useGoogleAdsStore()
const { accounts: allAccounts } = storeToRefs(googleAdsStore)
const toast = useToast()

const oauthLoading = ref(false)
const showDisconnectConfirm = ref(false)
const brokenAvatar = ref(false)
const brokenReuseAvatar = ref(false)
const selectedSourceId = ref<number>(0)

const oauthConnected = computed(
  () => !!(props.account?.oauth_connected || props.account?.connected || props.account?.oauth_email),
)
const connectedLabel = computed(() => props.account?.oauth_email || 'Conta Google autorizada')
const connectedAvatar = computed(() => props.account?.picture_url || props.account?.oauth_picture_url || '')

const reusableProfiles = computed<GoogleAdsAccount[]>(() => {
  const seen = new Set<string>()
  const out: GoogleAdsAccount[] = []
  for (const a of allAccounts.value) {
    if ((!a.connected && !a.oauth_connected) || a.id === props.account?.id) continue
    const key = (a.oauth_email || String(a.id)).toLowerCase()
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
  return p?.oauth_email || p?.name || 'Conta Google'
})

const singleProfileAvatar = computed(() => {
  const p = reusableProfiles.value[0]
  return p?.picture_url || p?.oauth_picture_url || ''
})

const reuseOptions = computed<AdAccountSelectOption[]>(() =>
  reusableProfiles.value.map((a) => ({
    value: a.id,
    label: a.oauth_email || a.name,
    sublabel: a.name,
    avatarUrl: a.picture_url || a.oauth_picture_url || '',
    channel: 'google_ads',
    badge: 'Já autorizada',
  })),
)

const initials = (name: string) => {
  const clean = (name || '').split('@')[0] || 'GA'
  return clean.slice(0, 2).toUpperCase()
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
    await googleAdsApi.reuseOAuth(props.account.id, srcId)
    toast.success('Conta vinculada', 'Conexão Google reutilizada com sucesso.')
    emit('reused')
  } catch (error: unknown) {
    toast.error('Erro', errorMessage(error, 'Falha ao reutilizar conta Google'))
  } finally {
    oauthLoading.value = false
  }
}

const onConnect = async () => {
  if (!props.account) {
    emit('connect-new')
    return
  }
  oauthLoading.value = true
  try {
    await googleAdsApi.startOAuth(props.account.id)
  } catch (error: unknown) {
    toast.error('Erro', errorMessage(error, 'Falha ao iniciar OAuth'))
    oauthLoading.value = false
  }
}

const executeDisconnect = async () => {
  if (!props.account) return
  oauthLoading.value = true
  try {
    await googleAdsApi.disconnect(props.account.id)
    showDisconnectConfirm.value = false
    toast.success('Desconectado', 'Conta Google revogada.')
    emit('disconnected')
  } catch (error: unknown) {
    toast.error('Erro', errorMessage(error, 'Falha ao desconectar'))
  } finally {
    oauthLoading.value = false
  }
}

onMounted(() => { googleAdsApi.list() })
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
.oauth-email { font-size: var(--fs-md); color: var(--text-1); font-weight: var(--fw-medium); }
.oauth-text { font-size: var(--fs-md); color: var(--text-2); margin: 0; }
</style>
