<template>
  <div class="ga-page">
    <div class="page-header">
      <div>
        <h1 class="page-title">Google Ads</h1>
        <p class="page-subtitle">
          Conecte contas Google Ads via OAuth e gerencie as Conversion Actions
          que recebem conversões offline. Cada conta pode ter várias actions
          (uma de Vendas para webhooks, outra de Lead, etc.) e pode ser
          vinculada a um ou mais pixels.
        </p>
      </div>
      <BaseButton variant="primary" @click="openCreateModal">Nova conta Google Ads</BaseButton>
    </div>

    <div v-if="loading && !accounts.length" class="loading-state">
      <div class="spinner" />
      <p>Carregando contas...</p>
    </div>

    <div v-else-if="!accounts.length" class="empty-state">
      <h2 class="empty-title">Nenhuma conta Google Ads conectada</h2>
      <p class="empty-description">
        Cadastre uma conta para enviar conversões offline (Purchase, Lead, etc.)
        usando os click identifiers (gclid/wbraid/gbraid) capturados pelo tracker.
      </p>
      <BaseButton variant="primary" @click="openCreateModal">Conectar conta Google Ads</BaseButton>
    </div>

    <div v-else class="ga-grid">
      <GoogleAdsAccountCard
        v-for="account in accounts"
        :key="account.id"
        :account="account"
        @edit="openEditModal"
        @delete="openDeleteModal"
      />
    </div>

    <BaseModal
      v-model="formModalOpen"
      :title="editing ? 'Editar conta Google Ads' : 'Nova conta Google Ads'"
      :description="editing
        ? 'Selecione sua conta Google Ads e vincule as Conversion Actions aos eventos.'
        : 'Conecte sua conta Google em 1 clique para carregar suas contas e Conversion Actions automaticamente.'"
      size="lg"
    >
      <GoogleAdsAccountForm
        ref="formRef"
        :account="editing"
        @submit="onFormSubmit"
        @connect-new="onConnectNew"
        @reuse-new="onReuseNew"
        @refresh="onAccountRefreshed"
      />
      <template #footer>
        <BaseButton variant="ghost" @click="formModalOpen = false">Cancelar</BaseButton>
        <BaseButton variant="primary" :loading="saving" @click="formRef?.submit()">
          {{ editing ? 'Salvar alterações' : 'Criar conta' }}
        </BaseButton>
      </template>
    </BaseModal>

    <ConfirmModal
      v-model="deleteModalOpen"
      title="Excluir conta Google Ads?"
      description="Esta operação é irreversível"
      :message="deleting?.name
        ? `A conta &quot;${deleting.name}&quot; será removida. Pixels vinculados perdem o link automaticamente — eventos param de ir pro Google Ads dessa conta.`
        : ''"
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
import type { GoogleAdsAccount, GoogleAdsAccountInput } from '#shared/types'
import { useGoogleAdsStore } from '~/stores/useGoogleAdsStore'
import { useTenantStore } from '~/stores/useTenantStore'
import { errorMessage } from '~/utils/errorMessage'

definePageMeta({ layout: 'app', middleware: 'auth' })
const { formatTitle } = useBranding()
useHead({ title: computed(() => formatTitle('Google Ads')) })

const api = useGoogleAdsAccounts()
const googleAdsStore = useGoogleAdsStore()
const tenantStore = useTenantStore()
const { accounts, loading } = storeToRefs(googleAdsStore)
const toast = useToast()

const formModalOpen = ref(false)
const editing = ref<GoogleAdsAccount | null>(null)
const saving = ref(false)
const formRef = ref<{ submit: () => void } | null>(null)
const deleteModalOpen = ref(false)
const deleting = ref<GoogleAdsAccount | null>(null)
const deletingLoading = ref(false)

const fetchAccounts = async (force = false) => {
  try {
    await api.list(force)
  } catch (error: unknown) {
    toast.error('Erro ao carregar', errorMessage(error, 'Não foi possível listar contas'))
  }
}

const openCreateModal = () => { editing.value = null; formModalOpen.value = true }
const openEditModal = (a: GoogleAdsAccount) => { editing.value = a; formModalOpen.value = true }
const openDeleteModal = (a: GoogleAdsAccount) => { deleting.value = a; deleteModalOpen.value = true }

const onAccountRefreshed = async () => {
  if (!editing.value) return
  try {
    const fresh = await api.get(editing.value.id)
    googleAdsStore.upsertAccount(fresh)
    editing.value = fresh
  } catch {
    const local = accounts.value.find((a) => a.id === editing.value?.id)
    if (local) editing.value = local
  }
}

const onFormSubmit = async (input: GoogleAdsAccountInput) => {
  saving.value = true
  try {
    if (editing.value) {
      const updated = await api.update(editing.value.id, input)
      toast.success('Conta atualizada', `"${updated.name}" salva.`)
      editing.value = updated
      formModalOpen.value = false
    } else {
      const created = await api.create(input)
      toast.success('Conta criada', `"${created.name}" pronta para autorizar OAuth.`)
      editing.value = created
    }
  } catch (error: unknown) {
    toast.error('Erro ao salvar', errorMessage(error, 'Falha ao salvar conta'))
  } finally {
    saving.value = false
  }
}

const onConnectNew = async (input: GoogleAdsAccountInput) => {
  saving.value = true
  try {
    const created = await api.create(input)
    editing.value = created
    await api.startOAuth(created.id)
  } catch (error: unknown) {
    toast.error('Erro ao conectar OAuth', errorMessage(error, 'Falha ao iniciar OAuth do Google Ads'))
  } finally {
    saving.value = false
  }
}

const onReuseNew = async (input: GoogleAdsAccountInput) => {
  saving.value = true
  try {
    const created = await api.create(input)
    editing.value = created
    toast.success('Conta vinculada', 'Agora escolha a Conta Google Ads e as Conversion Actions.')
  } catch (error: unknown) {
    toast.error('Erro ao reutilizar conexão', errorMessage(error, 'Falha ao reutilizar login Google'))
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
    toast.success('Conta excluída', `"${removedName}" foi removida`)
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
  if (!status) return
  const message = (route.query.message as string | undefined) || ''
  const accountId = Number(route.query.account_id || 0)
  router.replace({ query: {} })
  if (status === 'success') {
    toast.success('Google Ads conectado', message ? `Conta autorizada: ${message}` : 'A conexão foi salva com sucesso.')
    await fetchAccounts(true)
    if (accountId > 0) {
      const found = accounts.value.find((a) => a.id === accountId)
      if (found) openEditModal(found)
    }
  } else if (status === 'error') {
    toast.error('Falha ao conectar Google Ads', message || 'Tente novamente.')
  }
}

watch(() => tenantStore.switchVersion, () => {
  formModalOpen.value = false
  editing.value = null
  fetchAccounts(true)
})

onMounted(async () => {
  await fetchAccounts()
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
