<script setup lang="ts">
import { Sparkles } from 'lucide-vue-next'
import { API_KEY_SCOPE_OPTIONS, type Account, type APIKey, type APIKeyModuleScope } from '#shared/types'

const props = defineProps<{
  modelValue: boolean
  editingKey: APIKey | null
  accounts: Account[]
  isOwner: boolean
  loading: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [val: boolean]
  submit: [payload: { name: string; account_id?: number; scopes: string[]; read_only: boolean }]
}>()

const allScopeIds = API_KEY_SCOPE_OPTIONS.map((o) => o.value)

const name = ref('')
const accountId = ref<number | null>(null)
const readOnly = ref(false)
const selectedScopes = ref<APIKeyModuleScope[]>([...allScopeIds])

watch(
  () => [props.modelValue, props.editingKey] as const,
  ([open, key]) => {
    if (!open) return
    if (key) {
      name.value = key.name
      accountId.value = key.account_id
      readOnly.value = Boolean(key.read_only)
      const isWildcard = !key.scopes?.length || key.scopes.includes('*')
      selectedScopes.value = isWildcard ? [...allScopeIds] : (key.scopes.filter((s) => allScopeIds.includes(s as APIKeyModuleScope)) as APIKeyModuleScope[])
    } else {
      name.value = ''
      accountId.value = null
      readOnly.value = false
      selectedScopes.value = [...allScopeIds]
    }
  },
  { immediate: true }
)

const isAllSelected = computed(() => selectedScopes.value.length === allScopeIds.length)

const toggleAllScopes = (checked: boolean) => {
  selectedScopes.value = checked ? [...allScopeIds] : ['events']
}

const onSubmit = () => {
  const finalScopes = isAllSelected.value ? ['*'] : [...selectedScopes.value]
  emit('submit', {
    name: name.value.trim(),
    account_id: !props.editingKey && props.isOwner && accountId.value ? accountId.value : undefined,
    scopes: finalScopes,
    read_only: readOnly.value,
  })
}
</script>

<template>
  <BaseModal
    :model-value="modelValue"
    :title="editingKey ? 'Editar Chave e Permissões' : 'Nova Chave de API e MCP'"
    size="md"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <div class="modal-form">
      <div class="unified-banner">
        <Sparkles class="banner-icon" />
        <span>
          A mesma chave <code>trk_...</code> funciona tanto na <strong>API REST</strong> quanto para conectar agentes de IA no <strong>Servidor MCP</strong>, obedecendo exatamente às permissões abaixo.
        </span>
      </div>

      <BaseInput
        v-model="name"
        label="Nome de identificação"
        placeholder="Ex: Agente IA Comercial, n8n Checkout, Make CRM"
        required
      />

      <div v-if="isOwner && !editingKey">
        <BaseDropdown
          v-model="accountId"
          label="Empresa vinculada"
          placeholder="Empresa atual selecionada"
          :options="accounts.map((a) => ({ value: a.id, label: `${a.name} (#${a.id})` }))"
        />
      </div>

      <div class="mode-strip">
        <BaseCheckbox
          v-model="readOnly"
          label="Modo Somente Leitura"
          hint="Permite que a chave consulte dados na API e no MCP, mas bloqueia qualquer alteração, criação ou disparo de eventos."
        />
      </div>

      <div class="scopes-section">
        <div class="scopes-head">
          <span class="scopes-title">Permissões por Módulo</span>
          <BaseCheckbox
            :model-value="isAllSelected"
            size="sm"
            label="Liberar todos os 8 módulos"
            @change="toggleAllScopes"
          />
        </div>

        <div class="scopes-grid">
          <div v-for="opt in API_KEY_SCOPE_OPTIONS" :key="opt.value" class="scope-item">
            <BaseCheckbox
              v-model="selectedScopes"
              :value="opt.value"
              size="sm"
              :label="opt.label"
              :hint="opt.hint"
            />
          </div>
        </div>
      </div>
    </div>

    <template #footer>
      <BaseButton variant="ghost" @click="emit('update:modelValue', false)">Cancelar</BaseButton>
      <BaseButton variant="primary" :loading="loading" :disabled="!name.trim() || !selectedScopes.length" @click="onSubmit">
        {{ editingKey ? 'Salvar Alterações' : 'Criar Chave' }}
      </BaseButton>
    </template>
  </BaseModal>
</template>

<style scoped>
.modal-form { display: flex; flex-direction: column; gap: 1rem; }
.unified-banner { display: flex; align-items: flex-start; gap: 0.6rem; padding: 0.75rem 0.9rem; background: var(--brand-soft-2); border: 1px solid var(--brand-border); border-radius: 10px; font-size: var(--fs-sm); color: var(--text-1); line-height: 1.45; }
.unified-banner code { font-family: var(--font-mono); color: var(--brand); }
.banner-icon { width: 1rem; height: 1rem; color: var(--brand); flex-shrink: 0; margin-top: 0.1rem; }
.mode-strip { padding: 0.75rem 0.9rem; background: var(--surface-deep); border: 1px solid var(--border); border-radius: 10px; }
.scopes-section { display: flex; flex-direction: column; gap: 0.65rem; }
.scopes-head { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 0.5rem; }
.scopes-title { font-size: var(--fs-sm); font-weight: var(--fw-semibold); color: var(--text-strong); text-transform: uppercase; letter-spacing: 0.04em; }
.scopes-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 0.55rem; }
.scope-item { padding: 0.65rem 0.75rem; background: var(--surface-deep); border: 1px solid var(--border); border-radius: 9px; }
@media (max-width: 640px) {
  .scopes-grid { grid-template-columns: 1fr; }
}
</style>
