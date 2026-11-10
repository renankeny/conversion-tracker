<template>
  <div class="accounts-page">
    <div class="page-header">
      <div>
        <h1 class="page-title">Empresas</h1>
        <p class="page-subtitle">
          Cadastre e gerencie as empresas clientes. Cada empresa opera como um ambiente isolado de pixels, eventos, CRM e relatórios.
        </p>
      </div>
      <BaseButton variant="primary" @click="openNewAccount">
        <template #iconLeft>
          <Plus :size="16" />
        </template>
        Nova Empresa
      </BaseButton>
    </div>

    <div class="kpi-strip">
      <div class="kpi-box">
        <span class="kpi-label">Total de Empresas</span>
        <strong class="kpi-value">{{ accounts.length }}</strong>
      </div>
      <div class="kpi-box">
        <span class="kpi-label">Empresas Ativas</span>
        <strong class="kpi-value">{{ activeAccountsCount }}</strong>
      </div>
      <div class="kpi-box">
        <span class="kpi-label">Pixels Vinculados</span>
        <strong class="kpi-value">{{ totalPixelsCount }}</strong>
      </div>
      <div class="kpi-box highlight">
        <span class="kpi-label">Selecionada no Painel</span>
        <strong class="kpi-value truncate">{{ activeAccount?.name || '—' }}</strong>
      </div>
    </div>

    <div class="filters-bar">
      <div class="search-field">
        <BaseInput v-model="searchQuery" placeholder="Buscar por nome, slug ou ID..." />
      </div>
      <div class="status-filter">
        <BaseDropdown v-model="statusFilter" :options="statusFilterOptions" />
      </div>
    </div>

    <div v-if="loading && !accounts.length" class="state-card">
      <div class="spinner" />
      <p class="state-text">Carregando empresas...</p>
    </div>

    <div v-else-if="!filteredAccounts.length" class="state-card">
      <Building2 :size="36" class="state-icon" />
      <h2 class="state-title">
        {{ accounts.length ? 'Nenhuma empresa encontrada no filtro' : 'Nenhuma empresa cadastrada' }}
      </h2>
      <p class="state-text">
        {{ accounts.length ? 'Ajuste os termos da busca ou o filtro de status.' : 'Cadastre sua primeira empresa para começar.' }}
      </p>
      <BaseButton v-if="!accounts.length" variant="primary" size="sm" @click="openNewAccount">
        + Nova Empresa
      </BaseButton>
    </div>

    <div v-else class="accounts-list">
      <AccountCard
        v-for="acc in filteredAccounts"
        :key="acc.id"
        :account="acc"
        :is-active-in-panel="String(acc.id) === String(selectedAccountId)"
        @select="onSelectInPanel"
        @edit="openEditAccount"
        @delete="askDeleteAccount"
      />
    </div>

    <AccountFormModal
      v-model="modalOpen"
      :editing="editingAccount"
      :saving="saving"
      @submit="onSaveAccount"
    />

    <ConfirmModal
      v-model="deleteOpen"
      title="Remover empresa?"
      :message="deleteConfirmMessage"
      confirm-label="Remover empresa"
      variant="danger"
      :loading="deletingNow"
      @confirm="onConfirmDelete"
    />
  </div>
</template>

<script setup lang="ts">
import { Building2, Plus } from 'lucide-vue-next'
import AccountCard from '~/components/accounts/AccountCard.vue'
import AccountFormModal from '~/components/accounts/AccountFormModal.vue'
import type { Account, CreateAccountInput, UpdateAccountInput } from '#shared/types'

definePageMeta({ layout: 'app', middleware: 'auth' })
const { formatTitle } = useBranding()
useHead({ title: computed(() => formatTitle('Empresas')) })

const accountsAPI = useAccounts()
const { accounts, selectedAccountId, activeAccount, switchAccount } = useAccountNavigation()
const toast = useToast()

const loading = ref(false)
const searchQuery = ref('')
const statusFilter = ref<'all' | 'enabled' | 'disabled'>('all')
const modalOpen = ref(false)
const editingAccount = ref<Account | null>(null)
const saving = ref(false)
const deleteOpen = ref(false)
const deletingAccount = ref<Account | null>(null)
const deletingNow = ref(false)

const statusFilterOptions = [
  { value: 'all', label: 'Todos os status' },
  { value: 'enabled', label: 'Apenas ativas' },
  { value: 'disabled', label: 'Apenas desativadas' },
]

const activeAccountsCount = computed(() => accounts.value.filter((a) => a.enabled).length)
const totalPixelsCount = computed(() => accounts.value.reduce((sum, a) => sum + (a.pixels_count ?? 0), 0))

const filteredAccounts = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  return accounts.value.filter((a) => {
    if (statusFilter.value === 'enabled' && !a.enabled) return false
    if (statusFilter.value === 'disabled' && a.enabled) return false
    if (!q) return true
    return (
      a.name.toLowerCase().includes(q) ||
      a.slug.toLowerCase().includes(q) ||
      String(a.id).includes(q.replace('#', ''))
    )
  })
})

const deleteConfirmMessage = computed(() => {
  const acc = deletingAccount.value
  if (!acc) return ''
  if ((acc.pixels_count ?? 0) > 0) {
    return `A empresa "${acc.name}" possui ${acc.pixels_count} pixel(s) vinculado(s). Ao removê-la, os pixels, eventos e leads desta empresa serão excluídos permanentemente.`
  }
  return `Tem certeza que deseja remover a empresa "${acc.name}"? Esta ação não pode ser desfeita.`
})

const fetchAccounts = async () => {
  loading.value = true
  try {
    await accountsAPI.list()
  } catch (error: any) {
    toast.error('Erro', error.message || 'Falha ao carregar empresas')
  } finally {
    loading.value = false
  }
}

const openNewAccount = () => { editingAccount.value = null; modalOpen.value = true }
const openEditAccount = (acc: Account) => { editingAccount.value = acc; modalOpen.value = true }
const askDeleteAccount = (acc: Account) => { deletingAccount.value = acc; deleteOpen.value = true }

const onSelectInPanel = (acc: Account) => {
  toast.success('Empresa selecionada', `Alternando painel para "${acc.name}"...`)
  switchAccount(acc.id)
}

const onSaveAccount = async (payload: CreateAccountInput & UpdateAccountInput) => {
  if (!payload.name?.trim()) {
    toast.error('Erro', 'O nome da empresa é obrigatório')
    return
  }
  saving.value = true
  try {
    if (editingAccount.value) {
      const updated = await accountsAPI.update(editingAccount.value.id, payload)
      toast.success('Empresa atualizada', `"${updated.name}" foi salva com sucesso`)
    } else {
      const created = await accountsAPI.create(payload)
      toast.success('Empresa cadastrada', `"${created.name}" já está disponível no painel`)
    }
    modalOpen.value = false
    editingAccount.value = null
  } catch (error: any) {
    toast.error('Erro ao salvar', error.message || 'Não foi possível salvar a empresa')
  } finally {
    saving.value = false
  }
}

const onConfirmDelete = async () => {
  if (!deletingAccount.value) return
  deletingNow.value = true
  try {
    const removedName = deletingAccount.value.name
    await accountsAPI.remove(deletingAccount.value.id)
    toast.success('Empresa removida', `"${removedName}" foi removida com sucesso`)
    deleteOpen.value = false
    deletingAccount.value = null
  } catch (error: any) {
    toast.error('Erro ao remover', error.message || 'Não foi possível remover a empresa')
  } finally {
    deletingNow.value = false
  }
}

onMounted(fetchAccounts)
</script>

<style scoped>
.accounts-page { display: flex; flex-direction: column; gap: 1.5rem; }
.page-header { display: flex; justify-content: space-between; align-items: flex-start; gap: 1rem; flex-wrap: wrap; }
.page-title { font-family: var(--font-heading); font-size: var(--fs-3xl); font-weight: var(--fw-bold); color: var(--text-strong); margin-bottom: 0.25rem; }
.page-subtitle { color: var(--text-2); font-size: var(--fs-md); max-width: 660px; }
.kpi-strip { display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); gap: 0.75rem; }
.kpi-box { background: var(--surface-2); border: 1px solid var(--border); border-radius: 10px; padding: 0.875rem 1rem; display: flex; flex-direction: column; gap: 0.25rem; }
.kpi-box.highlight { border-color: var(--brand-border); background: var(--surface-2); }
.kpi-label { font-size: var(--fs-xs); color: var(--text-3); text-transform: uppercase; letter-spacing: 0.04em; }
.kpi-value { font-family: var(--font-heading); font-size: var(--fs-xl); font-weight: var(--fw-semibold); color: var(--text-strong); }
.kpi-box.highlight .kpi-value { color: var(--brand); }
.filters-bar { display: flex; align-items: center; gap: 0.75rem; flex-wrap: wrap; }
.search-field { flex: 1; min-width: 240px; }
.status-filter { width: 210px; }
.accounts-list { display: flex; flex-direction: column; gap: 0.75rem; }
.state-card { display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 0.625rem; padding: 3.5rem 1.5rem; background: var(--surface-2); border: 1px dashed var(--border); border-radius: 12px; text-align: center; }
.state-icon { color: var(--text-3); }
.state-title { font-family: var(--font-heading); font-size: var(--fs-xl); font-weight: var(--fw-semibold); color: var(--text-strong); }
.state-text { font-size: var(--fs-md); color: var(--text-2); max-width: 420px; }
.spinner { width: 1.75rem; height: 1.75rem; border: 2px solid var(--border); border-top-color: var(--brand); border-radius: 50%; animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
</style>
