<template>
  <section class="admins-section">
    <div class="card">
      <div class="card-header">
        <h3 class="card-title">Usuários (Equipe)</h3>
        <BaseButton variant="primary" size="sm" @click="openNewAdmin">+ Novo usuário</BaseButton>
      </div>

      <div v-if="loadingAdmins" class="muted">Carregando...</div>
      <div v-else-if="!admins.length" class="muted">Nenhum usuário cadastrado.</div>

      <div v-else class="user-list">
        <div v-for="a in admins" :key="a.id" class="user-row">
          <div class="user-info">
            <div class="user-title-row">
              <span class="user-email">{{ a.name || a.email }}</span>
              <BaseBadge variant="info">{{ a.role }}</BaseBadge>
            </div>
            <span class="user-meta">ID #{{ a.id }} · {{ a.email }} · criado {{ formatDate(a.created_at || '') }}</span>
            <span class="user-companies">{{ formatUserAccounts(a) }}</span>
            <span v-if="a.role === 'employee' && a.permissions?.edit_pixels" class="user-meta text-success">
              ↳ Pode editar pixels
            </span>
          </div>
          <div v-if="a.id !== user?.id" class="row-actions">
            <BaseButton variant="outline" size="sm" @click="openEditAdmin(a)">Editar</BaseButton>
            <BaseButton variant="ghost" size="sm" @click="askDelete(a)">
              <span class="text-danger">Remover</span>
            </BaseButton>
          </div>
          <BaseBadge v-else variant="info">Você</BaseBadge>
        </div>
      </div>
    </div>

    <AdminUserFormModal
      v-model="modalOpen"
      :editing-user="editingAdmin"
      :saving="savingAdmin"
      :can-create-users="canCreateUsers"
      :role-options="roleOptions"
      :account-options="accountOptionsSettings"
      @submit="onSaveAdmin"
    />

    <ConfirmModal
      v-model="deleteAdminOpen"
      title="Remover usuário?"
      :message="deletingAdmin ? `O usuário '${deletingAdmin.email}' perderá acesso ao painel imediatamente.` : ''"
      confirm-label="Remover"
      variant="danger"
      :loading="deletingNow"
      @confirm="onConfirmDeleteAdmin"
    />
  </section>
</template>

<script setup lang="ts">
import type { Account, AdminUser, CreateAdminInput, UpdateAdminInput } from '#shared/types'
import AdminUserFormModal, { type AdminUserFormSubmitPayload } from '~/components/settings/AdminUserFormModal.vue'
import { formatDateTimeSP } from '~/utils/formatDate'
import { errorMessage } from '~/utils/errorMessage'

const settings = useSettings()
const accountsAPI = useAccounts()
const tenantStore = useTenantStore()
const { user, isOwner } = useAuth()
const toast = useToast()

const isAdmin = computed(() => user.value?.role === 'admin' || user.value?.role === 'superadmin')
const canCreateUsers = computed(() => user.value?.role !== 'employee')

const admins = ref<AdminUser[]>([])
const accounts = ref<Account[]>([])
const loadingAdmins = ref(false)
const modalOpen = ref(false)
const editingAdmin = ref<AdminUser | null>(null)
const savingAdmin = ref(false)
const deleteAdminOpen = ref(false)
const deletingAdmin = ref<AdminUser | null>(null)
const deletingNow = ref(false)

const roleOptions = computed(() => {
  const opts = []
  if (isOwner.value) opts.push({ value: 'admin', label: 'Administrador (acesso aos seus gerentes/funcionários)' })
  if (isAdmin.value || isOwner.value) opts.push({ value: 'manager', label: 'Gerente (cria empresas e funcionários)' })
  opts.push({ value: 'employee', label: 'Funcionário (acesso restrito)' })
  return opts
})

const accountOptionsSettings = computed(() =>
  accounts.value.map((a) => ({ value: String(a.id), label: `${a.name} (#${a.id})` })),
)

const accountNameById = computed(() => {
  const map = new Map<number, string>()
  for (const a of accounts.value) map.set(a.id, a.name)
  return map
})

const formatDate = (iso: string): string => formatDateTimeSP(iso)

const formatUserAccounts = (u: AdminUser): string => {
  if (u.role === 'superadmin' || u.role === 'owner') return 'Acesso global a todas as empresas'
  const ids = u.account_ids?.length ? u.account_ids : (u.account_id ? [u.account_id] : [])
  if (!ids.length) return 'Nenhuma empresa vinculada'
  const labels = ids.map((id) => accountNameById.value.get(id) || `#${id}`)
  if (labels.length <= 3) {
    return `${labels.length === 1 ? 'Empresa' : 'Empresas'}: ${labels.join(', ')}`
  }
  return `Empresas (${labels.length}): ${labels.slice(0, 3).join(', ')} +${labels.length - 3}`
}

const ensureAccounts = async () => {
  try {
    const res = await accountsAPI.list()
    accounts.value = res.data || []
  } catch { /* ignore */ }
}

const loadAdmins = async () => {
  loadingAdmins.value = true
  try {
    const [res] = await Promise.all([settings.listAdmins(), ensureAccounts()])
    admins.value = res.data || []
  } catch (error: unknown) {
    toast.error('Erro', errorMessage(error, 'Falha ao listar usuários'))
  } finally {
    loadingAdmins.value = false
  }
}

const openNewAdmin = async () => {
  if (!accounts.value.length) await ensureAccounts()
  editingAdmin.value = null
  modalOpen.value = true
}

const openEditAdmin = async (a: AdminUser) => {
  if (!accounts.value.length) await ensureAccounts()
  editingAdmin.value = a
  modalOpen.value = true
}

const onSaveAdmin = async (form: AdminUserFormSubmitPayload) => {
  if (!form.email.trim()) return toast.error('Erro', 'E-mail é obrigatório')
  if (!editingAdmin.value && form.password.length < 6) return toast.error('Erro', 'Senha deve ter pelo menos 6 caracteres')
  if (['admin', 'manager', 'employee'].includes(form.role) && !form.account_ids.length) {
    return toast.error('Erro', 'Selecione pelo menos uma empresa vinculada')
  }

  savingAdmin.value = true
  try {
    const basePayload: UpdateAdminInput = {
      email: form.email.trim().toLowerCase(),
      name: form.name.trim(),
      phone: form.phone.trim(),
      role: form.role,
      account_id: form.account_ids[0] ?? null,
      account_ids: form.account_ids,
      permissions: form.role === 'employee' ? { ...form.permissions } : undefined,
    }
    if (form.password) basePayload.password = form.password

    if (editingAdmin.value) {
      const updated = await settings.updateAdmin(editingAdmin.value.id, basePayload)
      admins.value = admins.value.map((a) => (a.id === updated.id ? updated : a))
      toast.success('Sucesso', 'Usuário atualizado com sucesso')
    } else {
      const created = await settings.createAdmin({ ...basePayload, password: form.password } as CreateAdminInput)
      admins.value = [...admins.value, created]
      toast.success('Usuário criado', `${created.email} já pode acessar as empresas selecionadas`)
    }
    modalOpen.value = false
  } catch (error: unknown) {
    toast.error('Erro', errorMessage(error, 'Falha ao salvar usuário'))
  } finally {
    savingAdmin.value = false
  }
}

const askDelete = (admin: AdminUser) => { deletingAdmin.value = admin; deleteAdminOpen.value = true }

const onConfirmDeleteAdmin = async () => {
  if (!deletingAdmin.value) return
  deletingNow.value = true
  try {
    await settings.deleteAdmin(deletingAdmin.value.id)
    admins.value = admins.value.filter((a) => a.id !== deletingAdmin.value!.id)
    toast.success('Sucesso', 'Usuário removido')
    deleteAdminOpen.value = false
    deletingAdmin.value = null
  } catch (error: unknown) {
    toast.error('Erro', errorMessage(error, 'Falha ao remover'))
  } finally {
    deletingNow.value = false
  }
}

watch(() => tenantStore.switchVersion, () => loadAdmins())
onMounted(loadAdmins)
</script>

<style scoped>
.card { background: var(--surface-2); border: 1px solid var(--border); border-radius: 12px; padding: 1.5rem; box-shadow: var(--shadow-card); }
.card-header { display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 0.75rem; margin-bottom: 1rem; }
.card-title { font-family: var(--font-sans); font-size: var(--fs-xl); font-weight: var(--fw-semibold); color: var(--text-strong); margin: 0; }
.muted { color: var(--text-3); font-size: var(--fs-md); font-style: italic; }
.text-danger { color: var(--danger); }
.text-success { color: var(--success); font-size: var(--fs-sm); display: block; }
.user-list { display: flex; flex-direction: column; gap: 0.5rem; }
.user-row { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 0.75rem; padding: 0.75rem 1rem; border: 1px solid var(--border-soft); border-radius: 10px; background: var(--surface-soft-2); }
.user-info { display: flex; flex-direction: column; gap: 0.15rem; }
.user-title-row { display: flex; align-items: center; gap: 0.5rem; flex-wrap: wrap; }
.user-email { font-weight: var(--fw-semibold); color: var(--text-1); font-size: var(--fs-md); }
.user-meta { font-size: var(--fs-sm); color: var(--text-3); }
.user-companies { font-size: var(--fs-xs); color: var(--text-2); font-weight: var(--fw-medium); }
.row-actions { display: flex; gap: 0.5rem; }
</style>
