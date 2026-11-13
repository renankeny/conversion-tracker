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
            <span class="user-email">
              {{ a.name || a.email }}
              <BaseBadge variant="info" style="margin-left:8px">{{ a.role }}</BaseBadge>
            </span>
            <span class="user-meta">ID #{{ a.id }} · {{ a.email }} · criado {{ formatDate(a.created_at || '') }}</span>
            <span v-if="a.role === 'employee' && a.permissions?.edit_pixels" class="user-meta text-success">↳ Pode editar pixels</span>
          </div>
          <div v-if="a.id !== user?.id" class="row-actions">
            <BaseButton variant="outline" size="sm" @click="openEditAdmin(a)">Editar</BaseButton>
            <BaseButton variant="ghost" size="sm" @click="askDelete(a)"><span class="text-danger">Remover</span></BaseButton>
          </div>
          <BaseBadge v-else variant="info">Você</BaseBadge>
        </div>
      </div>
    </div>

    <BaseModal v-model="newAdminOpen" :title="editingAdminId ? 'Editar usuário' : 'Criar novo usuário'" size="sm">
      <div class="form">
        <BaseInput v-model="newAdminForm.email" type="email" label="E-mail" placeholder="admin@empresa.com.br" required />
        <BaseInput
          v-model="newAdminForm.password"
          type="password"
          :label="editingAdminId ? 'Senha (deixe em branco para manter)' : 'Senha (mínimo 6 caracteres)'"
          :required="!editingAdminId"
        />
        <BaseInput v-model="newAdminForm.name" label="Nome" placeholder="João Silva" />
        <BaseInput v-model="newAdminForm.phone" label="Telefone" placeholder="11999999999" />

        <div v-if="canCreateUsers" class="form-field">
          <BaseDropdown v-model="newAdminForm.role" label="Tipo de acesso" :options="roleOptions" />
        </div>

        <div v-if="newAdminForm.role === 'employee'" class="form-field">
          <label class="form-label">Permissões Específicas</label>
          <BaseCheckbox
            v-model="newAdminForm.permissions.edit_pixels"
            label="Pode editar pixels"
          />
        </div>

        <div v-if="['admin', 'manager', 'employee'].includes(newAdminForm.role)" class="form-field">
          <BaseDropdown v-model="newAdminAccountId" label="Empresa vinculada" :options="accountOptionsSettings" placeholder="Selecione uma empresa" />
        </div>
      </div>
      <template #footer>
        <BaseButton variant="ghost" @click="newAdminOpen = false">Cancelar</BaseButton>
        <BaseButton variant="primary" :loading="creatingAdmin" @click="onCreateAdmin">{{ editingAdminId ? 'Salvar' : 'Criar' }}</BaseButton>
      </template>
    </BaseModal>

    <ConfirmModal
      v-model="deleteAdminOpen"
      title="Remover administrador?"
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
const newAdminOpen = ref(false)
const editingAdminId = ref<number | null>(null)
const creatingAdmin = ref(false)
const deleteAdminOpen = ref(false)
const deletingAdmin = ref<AdminUser | null>(null)
const deletingNow = ref(false)

const newAdminForm = reactive({
  email: '', password: '', name: '', phone: '', role: 'employee',
  account_id: null as number | null,
  permissions: { edit_pixels: false },
})

const newAdminAccountId = computed({
  get: () => (newAdminForm.account_id ? String(newAdminForm.account_id) : ''),
  set: (val) => { newAdminForm.account_id = val ? Number(val) : null },
})

const roleOptions = computed(() => {
  const opts = []
  if (isOwner.value) opts.push({ value: 'admin', label: 'Administrador (acesso aos seus gerentes/funcionários)' })
  if (isAdmin.value || isOwner.value) opts.push({ value: 'manager', label: 'Gerente (cria empresas e funcionários)' })
  opts.push({ value: 'employee', label: 'Funcionário (acesso restrito)' })
  return opts
})

const accountOptionsSettings = computed(() => accounts.value.map((a) => ({ value: String(a.id), label: `${a.name} (#${a.id})` })))

const formatDate = (iso: string): string => formatDateTimeSP(iso)

const loadAdmins = async () => {
  loadingAdmins.value = true
  try {
    const res = await settings.listAdmins()
    admins.value = res.data || []
  } catch (error: unknown) {
    toast.error('Erro', errorMessage(error, 'Falha ao listar admins'))
  } finally {
    loadingAdmins.value = false
  }
}

const ensureAccounts = async () => {
  if (accounts.value.length) return
  try {
    const res = await accountsAPI.list()
    accounts.value = res.data || []
  } catch { /* ignore */ }
}

const openEditAdmin = async (a: AdminUser) => {
  await ensureAccounts()
  editingAdminId.value = a.id
  Object.assign(newAdminForm, {
    email: a.email, password: '', name: a.name || '', phone: a.phone || '',
    role: a.role || 'employee', account_id: a.account_id ?? null,
    permissions: { edit_pixels: Boolean(a.permissions?.edit_pixels) },
  })
  newAdminOpen.value = true
}

const openNewAdmin = async () => {
  await ensureAccounts()
  editingAdminId.value = null
  Object.assign(newAdminForm, {
    email: '', password: '', name: '', phone: '', role: 'employee', account_id: null, permissions: { edit_pixels: false },
  })
  newAdminOpen.value = true
}

const onCreateAdmin = async () => {
  if (!newAdminForm.email) return toast.error('Erro', 'E-mail é obrigatório')
  if (!editingAdminId.value && newAdminForm.password.length < 6) return toast.error('Erro', 'Senha deve ter pelo menos 6 caracteres')
  if (['admin', 'manager', 'employee'].includes(newAdminForm.role) && !newAdminForm.account_id) return toast.error('Erro', 'Selecione uma empresa vinculada')

  creatingAdmin.value = true
  try {
    const basePayload: UpdateAdminInput = {
      email: newAdminForm.email.trim().toLowerCase(), name: newAdminForm.name.trim(), phone: newAdminForm.phone.trim(),
      role: newAdminForm.role, permissions: newAdminForm.role === 'employee' ? { ...newAdminForm.permissions } : undefined,
      account_id: newAdminForm.account_id,
    }
    if (newAdminForm.password) basePayload.password = newAdminForm.password

    if (editingAdminId.value) {
      const updated = await settings.updateAdmin(editingAdminId.value, basePayload)
      admins.value = admins.value.map((a) => (a.id === updated.id ? updated : a))
      toast.success('Sucesso', 'Usuário atualizado com sucesso')
    } else {
      const created = await settings.createAdmin({ ...basePayload, password: newAdminForm.password } as CreateAdminInput)
      admins.value = [...admins.value, created]
      toast.success('Admin criado', `${created.email} já pode logar`)
    }
    newAdminOpen.value = false
  } catch (error: unknown) {
    toast.error('Erro', errorMessage(error, 'Falha ao salvar usuário'))
  } finally {
    creatingAdmin.value = false
  }
}

const askDelete = (admin: AdminUser) => { deletingAdmin.value = admin; deleteAdminOpen.value = true }

const onConfirmDeleteAdmin = async () => {
  if (!deletingAdmin.value) return
  deletingNow.value = true
  try {
    await settings.deleteAdmin(deletingAdmin.value.id)
    admins.value = admins.value.filter((a) => a.id !== deletingAdmin.value!.id)
    toast.success('Sucesso', 'Admin removido')
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
.card-title { font-family: var(--font-sans); font-size: var(--fs-xl); font-weight: var(--fw-semibold); color: var(--text-strong); }
.muted { color: var(--text-3); font-size: var(--fs-md); font-style: italic; }
.text-danger { color: var(--danger); }
.text-success { color: var(--success); font-size: var(--fs-sm); display: block; }
.user-list { display: flex; flex-direction: column; gap: 0.5rem; }
.user-row { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 0.75rem; padding: 0.75rem 1rem; border: 1px solid var(--border-soft); border-radius: 10px; background: var(--surface-soft-2); }
.user-info { display: flex; flex-direction: column; gap: 0.125rem; }
.user-email { font-weight: var(--fw-semibold); color: var(--text-1); }
.user-meta { font-size: var(--fs-sm); color: var(--text-3); }
.row-actions { display: flex; gap: 0.5rem; }
.form { display: flex; flex-direction: column; gap: 0.875rem; }
.form-field { display: flex; flex-direction: column; gap: 0.375rem; }
.form-label { font-size: var(--fs-sm); font-weight: var(--fw-medium); color: var(--text-2); }
</style>
