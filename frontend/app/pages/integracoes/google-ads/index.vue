<template>
  <div class="ga-page">
    <div class="page-header">
      <div>
        <h1 class="page-title">Google Ads</h1>
        <p class="page-subtitle">
          Conecte contas Google Ads em uma janela rápida via OAuth seguro e gerencie as Ações de Conversão que recebem conversões offline (Purchase, Lead e eventos personalizados) vinculadas aos seus Pixels.
        </p>
      </div>
      <BaseButton v-if="canEdit" variant="primary" @click="openCreateModal">
        Nova conta Google Ads
      </BaseButton>
    </div>

    <div v-if="loading && !accounts.length" class="loading-state">
      <div class="spinner" />
      <p>Carregando contas Google Ads...</p>
    </div>

    <div v-else-if="!accounts.length" class="empty-state">
      <h2 class="empty-title">Nenhuma conta Google Ads conectada</h2>
      <p class="empty-description">
        Conecte sua conta Google em 1 clique para despachar conversões offline usando os identificadores de clique (GCLID, WBRAID e GBRAID) capturados pelo tracker.
      </p>
      <BaseButton v-if="canEdit" variant="primary" @click="openCreateModal">
        Conectar conta Google Ads
      </BaseButton>
    </div>

    <div v-else class="ga-grid">
      <GoogleAdsAccountCard
        v-for="account in accounts"
        :key="account.id"
        :account="account"
        :linked-pixels="linkedPixelsFor(account.id)"
        :can-edit="canEdit"
        @edit="openEditModal"
        @delete="openDeleteModal"
      />
    </div>

    <BaseModal
      v-model="formModalOpen"
      :title="editing ? 'Configurar conta Google Ads' : 'Nova conta Google Ads'"
      :description="editing
        ? 'Selecione a conta Google Ads, mapeie as Ações de Conversão e valide o envio.'
        : 'Autorize sua conta Google na janela segura para carregar suas contas e Ações de Conversão.'"
      size="lg"
    >
      <GoogleAdsAccountForm
        ref="formRef"
        :account="editing"
        @submit="onFormSubmit"
        @connected="onAccountConnected"
        @refresh="onAccountRefreshed"
      />
      <template #footer>
        <BaseButton variant="ghost" @click="formModalOpen = false">Cancelar</BaseButton>
        <BaseButton variant="primary" :loading="saving" @click="formRef?.submit()">
          {{ primaryModalLabel }}
        </BaseButton>
      </template>
    </BaseModal>

    <ConfirmModal
      v-model="deleteModalOpen"
      title="Excluir conta Google Ads?"
      description="Esta operação é irreversível"
      :message="deleteMessage"
      confirm-label="Excluir definitivamente"
      variant="danger"
      :loading="deletingLoading"
      require-typing
      :typing-phrase="deleting?.name || ''"
      @confirm="onConfirmDelete"
      @cancel="deleting = null"
    />
  </div>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import GoogleAdsAccountCard from '~/components/google-ads/GoogleAdsAccountCard.vue'
import GoogleAdsAccountForm from '~/components/google-ads/GoogleAdsAccountForm.vue'
import type { GoogleAdsAccount, GoogleAdsAccountInput, Pixel } from '#shared/types'
import { useAuthStore } from '~/stores/useAuthStore'
import { useGoogleAdsStore } from '~/stores/useGoogleAdsStore'
import { usePixelStore } from '~/stores/usePixelStore'
import { useTenantStore } from '~/stores/useTenantStore'
import { errorMessage } from '~/utils/errorMessage'

definePageMeta({ layout: 'app', middleware: 'auth' })
const { formatTitle } = useBranding()
useHead({ title: computed(() => formatTitle('Google Ads')) })

const api = useGoogleAdsAccounts()
const authStore = useAuthStore()
const googleAdsStore = useGoogleAdsStore()
const pixelStore = usePixelStore()
const tenantStore = useTenantStore()
const { accounts, loading } = storeToRefs(googleAdsStore)
const { pixels } = storeToRefs(pixelStore)
const { canManagePixels: canEdit } = storeToRefs(authStore)
const toast = useToast()

const formModalOpen = ref(false)
const editing = ref<GoogleAdsAccount | null>(null)
const saving = ref(false)
const formRef = ref<{ submit: () => void } | null>(null)
const deleteModalOpen = ref(false)
const deleting = ref<GoogleAdsAccount | null>(null)
const deletingLoading = ref(false)

const linkedPixelsFor = (accountId: number): Pixel[] =>
  pixels.value.filter((p) => p.google_ads_account_id === accountId)

const primaryModalLabel = computed(() => {
  if (editing.value) return 'Salvar alterações'
  const hasExisting = accounts.value.some((a) => a.connected || a.oauth_connected)
  return hasExisting ? 'Conectar com conta existente' : 'Conectar nova conta'
})

const deleteMessage = computed(() => {
  if (!deleting.value?.name) return ''
  const affected = linkedPixelsFor(deleting.value.id)
  if (!affected.length) {
    return `A conta "${deleting.value.name}" será removida definitivamente.`
  }
  const names = affected.map((p) => p.label || p.name).join(', ')
  return `A conta "${deleting.value.name}" será removida e desvinculada de ${affected.length} ${affected.length === 1 ? 'pixel' : 'pixels'}: ${names}. Os envios offline dessa conta serão encerrados.`
})

const fetchPageData = async (force = false) => {
  try {
    await Promise.all([api.list(force), pixelStore.fetch(force).catch(() => [])])
  } catch (error: unknown) {
    toast.error('Erro ao carregar', errorMessage(error, 'Não foi possível listar contas Google Ads'))
  }
}

const openCreateModal = () => { editing.value = null; formModalOpen.value = true }
const openEditModal = (a: GoogleAdsAccount) => { editing.value = a; formModalOpen.value = true }
const openDeleteModal = (a: GoogleAdsAccount) => { deleting.value = a; deleteModalOpen.value = true }

const onAccountConnected = (acc: GoogleAdsAccount) => {
  editing.value = acc
}

const onAccountRefreshed = async () => {
  if (!editing.value) return
  try {
    const fresh = await api.get(editing.value.id)
    editing.value = fresh
  } catch {
    const local = accounts.value.find((a) => a.id === editing.value?.id)
    if (local) editing.value = local
  }
}

const onFormSubmit = async (input: GoogleAdsAccountInput) => {
  if (!editing.value) return
  saving.value = true
  try {
    const updated = await api.update(editing.value.id, input)
    toast.success('Conta atualizada', `"${updated.name}" salva com sucesso.`)
    editing.value = updated
    formModalOpen.value = false
  } catch (error: unknown) {
    toast.error('Erro ao salvar', errorMessage(error, 'Falha ao salvar conta Google Ads'))
  } finally {
    saving.value = false
  }
}

const onConfirmDelete = async () => {
  if (!deleting.value) return
  deletingLoading.value = true
  try {
    const removedName = deleting.value.name
    await api.remove(deleting.value.id)
    await pixelStore.fetch(true).catch(() => [])
    toast.success('Conta excluída', `"${removedName}" foi removida.`)
    deleteModalOpen.value = false
    deleting.value = null
  } catch (error: unknown) {
    toast.error('Erro ao excluir', errorMessage(error, 'Falha ao excluir conta'))
  } finally {
    deletingLoading.value = false
  }
}

const handleOAuthCallback = async () => {
  const route = useRoute()
  const router = useRouter()
  const status = route.query.google_ads_oauth as string | undefined
  const accountId = Number(route.query.account_id || 0)
  const message = String(route.query.message || '')

  if (import.meta.client && status && window.name === 'trackfind_oauth_popup') {
    const payload = { type: 'trackfind:oauth-complete', channel: 'google_ads', status, accountId, message, ts: Date.now() }
    try { new BroadcastChannel('trackfind_oauth').postMessage(payload) } catch {}
    try { localStorage.setItem('trackfind_oauth_event', JSON.stringify(payload)) } catch {}
    window.close()
    return
  }

  if (!status && !accountId) return
  router.replace({ query: {} })
  if (status === 'success') {
    toast.success('Google Ads conectado', message ? `Conta autorizada: ${message}` : 'Conexão salva com sucesso.')
    await fetchPageData(true)
  } else if (status === 'error') {
    toast.error('Falha ao conectar Google Ads', message || 'Tente novamente.')
  }
  if (accountId > 0 && canEdit.value) {
    const found = accounts.value.find((a) => a.id === accountId)
    if (found) openEditModal(found)
  }
}

watch(() => tenantStore.switchVersion, () => {
  formModalOpen.value = false
  editing.value = null
  fetchPageData(true)
})

onMounted(async () => {
  await fetchPageData()
  await handleOAuthCallback()
})
</script>

<style scoped>
.ga-page { display: flex; flex-direction: column; gap: 2rem; }
.page-header { display: flex; justify-content: space-between; align-items: flex-start; gap: 1rem; flex-wrap: wrap; }
.page-header > div:first-child { flex: 1; min-width: 280px; }
.page-title { font-family: var(--font-heading); font-size: var(--fs-3xl); font-weight: var(--fw-bold); color: var(--text-strong); margin-bottom: 0.25rem; }
.page-subtitle { color: var(--text-2); font-size: var(--fs-lg); max-width: 720px; line-height: 1.5; }
.ga-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(min(100%, 360px), 1fr)); gap: 1.25rem; }
.loading-state, .empty-state { display: flex; flex-direction: column; align-items: center; justify-content: center; padding: 4rem 2rem; background: var(--surface-2); border: 1px dashed var(--border); border-radius: 12px; gap: 0.75rem; text-align: center; }
.empty-title { font-family: var(--font-heading); font-size: var(--fs-2xl); font-weight: var(--fw-semibold); color: var(--text-strong); }
.empty-description { color: var(--text-2); max-width: 460px; margin-bottom: 0.5rem; line-height: 1.5; }
.spinner { width: 2rem; height: 2rem; border: 3px solid var(--border); border-top-color: var(--brand); border-radius: 50%; animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
</style>
