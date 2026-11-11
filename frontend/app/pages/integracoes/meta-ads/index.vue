<template>
  <div class="meta-page">
    <div class="page-header">
      <div>
        <h1 class="page-title">Meta Ads</h1>
        <p class="page-subtitle">
          Conecte sua conta Facebook / Business Manager em 1 clique via OAuth seguro para enriquecer automaticamente Campanha, Conjunto e Criativo nos leads Click-to-WhatsApp, exibir Spend ao vivo no Dashboard e vincular aos seus Pixels.
        </p>
      </div>
      <BaseButton variant="primary" @click="openCreateModal">Nova conta Meta Ads</BaseButton>
    </div>

    <div v-if="loading && !accounts.length" class="loading-state">
      <div class="spinner" />
      <p>Carregando contas Meta Ads...</p>
    </div>

    <div v-else-if="!accounts.length" class="empty-state">
      <h2 class="empty-title">Nenhuma conta Meta Ads conectada</h2>
      <p class="empty-description">
        Conecte sua conta Meta em 1 clique sem precisar gerar tokens manuais no Gerenciador de Negócios.
      </p>
      <BaseButton variant="primary" @click="openCreateModal">Conectar conta Meta Ads</BaseButton>
    </div>

    <div v-else class="meta-grid">
      <MetaAdsAccountCard
        v-for="acc in accounts"
        :key="acc.id"
        :account="acc"
        @edit="openEditModal"
        @delete="openDeleteModal"
      />
    </div>

    <BaseModal
      v-model="formModalOpen"
      :title="editing ? 'Configurar conta Meta Ads' : 'Nova conta Meta Ads'"
      :description="editing
        ? 'Escolha a Conta de Anúncios e a Página padrão vinculadas a esta conexão.'
        : 'Autorize sua conta Facebook/Meta em 1 clique para carregar suas Contas de Anúncios e Páginas.'"
      size="lg"
    >
      <MetaAdsAccountForm
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
      title="Excluir conta Meta Ads?"
      description="Esta operação é irreversível"
      :message="deleting?.name
        ? `A conta &quot;${deleting.name}&quot; será removida e desvinculada dos pixels associados.`
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
import MetaAdsAccountCard from '~/components/meta-ads/MetaAdsAccountCard.vue'
import MetaAdsAccountForm from '~/components/meta-ads/MetaAdsAccountForm.vue'
import type { MetaAccount, MetaAccountInput } from '#shared/types'
import { useMetaAdsStore } from '~/stores/useMetaAdsStore'
import { useTenantStore } from '~/stores/useTenantStore'

definePageMeta({ layout: 'app', middleware: 'auth' })
const { formatTitle } = useBranding()
useHead({ title: computed(() => formatTitle('Meta Ads')) })

const api = useMetaAccounts()
const metaAdsStore = useMetaAdsStore()
const tenantStore = useTenantStore()
const { accounts, loading } = storeToRefs(metaAdsStore)
const toast = useToast()

const formModalOpen = ref(false)
const editing = ref<MetaAccount | null>(null)
const saving = ref(false)
const formRef = ref<{ submit: () => void } | null>(null)
const deleteModalOpen = ref(false)
const deleting = ref<MetaAccount | null>(null)
const deletingLoading = ref(false)

const fetchAccounts = async (force = false) => {
  try {
    await api.list(force)
  } catch (error: any) {
    toast.error('Erro ao carregar', error.message || 'Não foi possível listar contas Meta Ads')
  }
}

const openCreateModal = () => { editing.value = null; formModalOpen.value = true }
const openEditModal = (a: MetaAccount) => { editing.value = a; formModalOpen.value = true }
const openDeleteModal = (a: MetaAccount) => { deleting.value = a; deleteModalOpen.value = true }

const onAccountRefreshed = async () => {
  if (!editing.value) return
  try {
    const fresh = await api.get(editing.value.id)
    metaAdsStore.upsertAccount(fresh)
    editing.value = fresh
  } catch {
    const local = accounts.value.find((a) => a.id === editing.value?.id)
    if (local) editing.value = local
  }
}

const onFormSubmit = async (input: MetaAccountInput) => {
  saving.value = true
  try {
    if (editing.value) {
      const updated = await api.update(editing.value.id, input)
      toast.success('Conta atualizada', `"${updated.name}" salva com sucesso.`)
      editing.value = updated
      formModalOpen.value = false
    } else {
      const created = await api.create(input)
      toast.success('Conta criada', `"${created.name}" pronta para conectar OAuth.`)
      editing.value = created
    }
  } catch (error: any) {
    toast.error('Erro ao salvar', error.message || 'Falha ao salvar conta Meta Ads')
  } finally {
    saving.value = false
  }
}

const onConnectNew = async ({ input, switchAccount }: { input: MetaAccountInput; switchAccount: boolean }) => {
  saving.value = true
  try {
    const created = await api.create(input)
    editing.value = created
    await api.startOAuth(created.id, switchAccount)
  } catch (error: any) {
    toast.error('Erro ao conectar OAuth', error.message || 'Falha ao iniciar OAuth da Meta')
  } finally {
    saving.value = false
  }
}

const onReuseNew = async (input: MetaAccountInput) => {
  saving.value = true
  try {
    const created = await api.create(input)
    editing.value = created
    toast.success('Conta vinculada', 'Agora escolha a Conta de Anúncios e a Página padrão.')
  } catch (error: any) {
    toast.error('Erro ao reutilizar conexão', error.message || 'Falha ao reutilizar login Meta')
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
    toast.success('Conta excluída', `"${removedName}" foi removida.`)
    deleteModalOpen.value = false
    deleting.value = null
  } catch (error: any) {
    toast.error('Erro ao excluir', error.message || 'Falha ao excluir conta')
  } finally {
    deletingLoading.value = false
  }
}

const handleOAuthCallback = async () => {
  const route = useRoute()
  const router = useRouter()
  const status = route.query.meta_ads_oauth as string | undefined
  if (!status) return
  const message = (route.query.message as string | undefined) || ''
  const accountId = Number(route.query.account_id || 0)
  router.replace({ query: {} })
  if (status === 'success') {
    toast.success('Meta Ads conectada', message ? `Perfil autorizado: ${message}` : 'Conexão salva com sucesso.')
    await fetchAccounts(true)
    if (accountId > 0) {
      const found = accounts.value.find((a) => a.id === accountId)
      if (found) openEditModal(found)
    }
  } else if (status === 'error') {
    toast.error('Falha ao conectar Meta Ads', message || 'Tente novamente.')
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
.meta-page { display: flex; flex-direction: column; gap: 2rem; }
.page-header { display: flex; justify-content: space-between; align-items: flex-start; gap: 1rem; flex-wrap: wrap; }
.page-header > div:first-child { flex: 1; min-width: 280px; }
.page-title { font-family: var(--font-heading); font-size: var(--fs-3xl); font-weight: var(--fw-bold); color: var(--text-strong); margin-bottom: 0.25rem; }
.page-subtitle { color: var(--text-2); font-size: var(--fs-lg); max-width: 720px; line-height: 1.5; }
.meta-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(min(100%, 360px), 1fr)); gap: 1.25rem; }
.loading-state, .empty-state { display: flex; flex-direction: column; align-items: center; justify-content: center; padding: 4rem 2rem; background: var(--surface-2); border: 1px dashed var(--border); border-radius: 12px; gap: 0.75rem; text-align: center; }
.empty-title { font-family: var(--font-heading); font-size: var(--fs-2xl); font-weight: var(--fw-semibold); color: var(--text-strong); }
.empty-description { color: var(--text-2); max-width: 460px; margin-bottom: 0.5rem; line-height: 1.5; }
.spinner { width: 2rem; height: 2rem; border: 3px solid var(--border); border-top-color: var(--brand); border-radius: 50%; animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
</style>

