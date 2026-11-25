<template>
  <BaseModal
    :model-value="modelValue"
    :title="editingId ? 'Editar usuário' : 'Criar novo usuário'"
    size="sm"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <div class="form">
      <BaseInput v-model="form.email" type="email" label="E-mail" placeholder="admin@empresa.com.br" required />
      <BaseInput
        v-model="form.password"
        type="password"
        :label="editingId ? 'Senha (deixe em branco para manter)' : 'Senha (mínimo 6 caracteres)'"
        :required="!editingId"
      />
      <BaseInput v-model="form.name" label="Nome" placeholder="João Silva" />
      <BaseInput v-model="form.phone" label="Telefone" placeholder="11999999999" />

      <div v-if="canCreateUsers" class="form-field">
        <BaseDropdown v-model="form.role" label="Tipo de acesso" :options="roleOptions" />
      </div>

      <div v-if="form.role === 'employee'" class="form-field">
        <label class="form-label">Permissões Específicas</label>
        <BaseCheckbox
          v-model="form.permissions.edit_pixels"
          label="Pode editar pixels"
        />
      </div>

      <div v-if="['admin', 'manager', 'employee'].includes(form.role)" class="form-field">
        <MultiSelectDropdown
          v-model="form.account_ids"
          label="Empresas vinculadas"
          :options="accountOptions"
          placeholder="Selecione quantas empresas quiser"
        />
        <span class="field-hint">
          O usuário poderá alternar no painel entre todas as empresas selecionadas.
        </span>
      </div>
    </div>

    <template #footer>
      <BaseButton variant="ghost" @click="emit('update:modelValue', false)">Cancelar</BaseButton>
      <BaseButton variant="primary" :loading="saving" @click="onSubmit">
        {{ editingId ? 'Salvar' : 'Criar' }}
      </BaseButton>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import type { AdminUser } from '#shared/types'

export interface AdminUserFormSubmitPayload {
  email: string
  password: string
  name: string
  phone: string
  role: string
  account_ids: number[]
  permissions: { edit_pixels: boolean }
}

const props = defineProps<{
  modelValue: boolean
  editingUser: AdminUser | null
  saving: boolean
  canCreateUsers: boolean
  roleOptions: Array<{ value: string; label: string }>
  accountOptions: Array<{ value: string; label: string }>
}>()

const emit = defineEmits<{
  'update:modelValue': [val: boolean]
  submit: [payload: AdminUserFormSubmitPayload]
}>()

const editingId = computed(() => props.editingUser?.id ?? null)

const form = reactive({
  email: '',
  password: '',
  name: '',
  phone: '',
  role: 'employee',
  account_ids: [] as string[],
  permissions: { edit_pixels: false },
})

watch(
  () => [props.modelValue, props.editingUser] as const,
  ([open, target]) => {
    if (!open) return
    if (target) {
      const ids = target.account_ids?.length
        ? target.account_ids.map(String)
        : (target.account_id ? [String(target.account_id)] : [])
      Object.assign(form, {
        email: target.email,
        password: '',
        name: target.name || '',
        phone: target.phone || '',
        role: target.role || 'employee',
        account_ids: ids,
        permissions: { edit_pixels: Boolean(target.permissions?.edit_pixels) },
      })
    } else {
      Object.assign(form, {
        email: '',
        password: '',
        name: '',
        phone: '',
        role: 'employee',
        account_ids: [],
        permissions: { edit_pixels: false },
      })
    }
  },
  { immediate: true },
)

const onSubmit = () => {
  emit('submit', {
    email: form.email,
    password: form.password,
    name: form.name,
    phone: form.phone,
    role: form.role,
    account_ids: form.account_ids.map(Number).filter((n) => Number.isFinite(n) && n > 0),
    permissions: { ...form.permissions },
  })
}
</script>

<style scoped>
.form { display: flex; flex-direction: column; gap: 0.875rem; }
.form-field { display: flex; flex-direction: column; gap: 0.375rem; }
.form-label { font-size: var(--fs-sm); font-weight: var(--fw-medium); color: var(--text-2); }
.field-hint { font-size: var(--fs-xs); color: var(--text-3); line-height: 1.4; }
</style>
