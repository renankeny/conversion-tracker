<template>
  <section class="api-keys-section">
    <div class="card">
      <div class="card-header">
        <div>
          <h3 class="card-title">API Keys</h3>
          <p class="card-hint">
            <strong>O segredo aparece uma única vez na criação</strong> — copie e guarde.
            Cada key é vinculada a uma empresa e atua como member dela.
          </p>
        </div>
        <BaseButton variant="primary" size="sm" @click="openNewKey">
          + Nova key
        </BaseButton>
      </div>

      <div v-if="loadingKeys" class="muted">Carregando...</div>
      <div v-else-if="!apiKeys.length" class="muted">Nenhuma API key cadastrada.</div>

      <div v-else class="account-list">
        <div v-for="k in apiKeys" :key="k.id" class="account-row">
          <div class="account-info">
            <div class="account-name">
              {{ k.name }}
              <BaseBadge v-if="k.revoked_at" variant="danger">revogada</BaseBadge>
              <BaseBadge v-else variant="success">ativa</BaseBadge>
            </div>
            <span class="account-meta">
              <code>{{ k.key_prefix }}...</code> · empresa #{{ k.account_id }} ·
              criada {{ formatDate(k.created_at) }}
              <span v-if="k.last_used_at"> · usada {{ formatDate(k.last_used_at) }}</span>
              <span v-else> · nunca usada</span>
            </span>
          </div>
          <div class="account-actions">
            <BaseButton
              v-if="!k.revoked_at"
              variant="ghost"
              size="sm"
              @click="askRevokeKey(k)"
            >
              <span class="text-danger">Revogar</span>
            </BaseButton>
          </div>
        </div>
      </div>
    </div>

    <BaseModal v-model="newKeyOpen" title="Criar API Key" size="sm">
      <div class="form">
        <BaseInput
          v-model="keyForm.name"
          label="Nome da key"
          placeholder="Integração CRM, Bot WhatsApp, etc"
          hint="Apenas pra você se localizar no painel."
          required
        />
        <div v-if="isOwner" class="form-field">
          <BaseDropdown
            v-model="keyForm.account_id"
            label="Empresa vinculada"
            placeholder="Selecione uma empresa"
            :options="accounts.map(a => ({ value: a.id, label: `${a.name} (#${a.id})` }))"
          />
        </div>
      </div>
      <template #footer>
        <BaseButton variant="ghost" @click="newKeyOpen = false">Cancelar</BaseButton>
        <BaseButton variant="primary" :loading="creatingKey" @click="onCreateKey">
          Criar
        </BaseButton>
      </template>
    </BaseModal>

    <BaseModal v-model="newKeyResultOpen" title="API Key criada" size="sm" :close-on-backdrop="false">
      <div class="form">
        <p class="card-hint">
          <strong>Copie agora.</strong> O segredo não aparece de novo em parte alguma.
          Ao fechar este modal, fica registrado apenas o prefixo no painel.
        </p>
        <div class="key-secret">
          <code>{{ createdKeyPlaintext }}</code>
          <BaseButton size="sm" variant="ghost" @click="copyKey">Copiar</BaseButton>
        </div>
        <p class="card-hint">
          Use no header HTTP:
          <code>Authorization: Bearer {{ createdKeyPlaintext || 'trk_xxx' }}</code>
        </p>
      </div>
      <template #footer>
        <BaseButton variant="primary" @click="closeKeyResult">Entendi, fechar</BaseButton>
      </template>
    </BaseModal>

    <ConfirmModal
      v-model="revokeKeyOpen"
      title="Revogar API Key?"
      :message="revokingKey
        ? `A key '${revokingKey.name}' (${revokingKey.key_prefix}...) deixará de funcionar imediatamente. Operação irreversível.`
        : ''"
      confirm-label="Revogar"
      variant="danger"
      :loading="revokingKeyNow"
      @confirm="onConfirmRevokeKey"
    />
  </section>
</template>

<script setup lang="ts">
import type { Account, APIKey } from '#shared/types'

const apiKeysAPI = useAPIKeys()
const accountsAPI = useAccounts()
const tenantStore = useTenantStore()
const { isOwner } = useAuth()
const toast = useToast()

const apiKeys = ref<APIKey[]>([])
const accounts = ref<Account[]>([])
const loadingKeys = ref(false)
const newKeyOpen = ref(false)
const keyForm = reactive<{ name: string; account_id: number | null }>({ name: '', account_id: null })
const creatingKey = ref(false)
const newKeyResultOpen = ref(false)
const createdKeyPlaintext = ref('')
const revokeKeyOpen = ref(false)
const revokingKey = ref<APIKey | null>(null)
const revokingKeyNow = ref(false)

const formatDate = (iso: string): string =>
  iso ? new Date(iso).toLocaleString('pt-BR', { day: '2-digit', month: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit' }) : ''

const loadAPIKeys = async () => {
  loadingKeys.value = true
  try {
    const res = await apiKeysAPI.list()
    apiKeys.value = res.data || []
  } catch (error: any) {
    toast.error('Erro', error.message || 'Falha ao listar keys')
  } finally {
    loadingKeys.value = false
  }
}

const openNewKey = async () => {
  if (isOwner.value && !accounts.value.length) {
    try {
      const res = await accountsAPI.list()
      accounts.value = res.data || []
    } catch { /* ignore */ }
  }
  newKeyOpen.value = true
}

const onCreateKey = async () => {
  if (!keyForm.name.trim()) return toast.error('Erro', 'Nome é obrigatório')
  creatingKey.value = true
  try {
    const res = await apiKeysAPI.create({
      name: keyForm.name.trim(),
      account_id: isOwner.value && keyForm.account_id ? keyForm.account_id : undefined,
    })
    apiKeys.value = [res.key, ...apiKeys.value]
    createdKeyPlaintext.value = res.plaintext
    newKeyOpen.value = false
    newKeyResultOpen.value = true
    keyForm.name = ''
    keyForm.account_id = null
  } catch (error: any) {
    toast.error('Erro', error.message || 'Falha ao criar key')
  } finally {
    creatingKey.value = false
  }
}

const copyKey = async () => {
  try {
    await navigator.clipboard.writeText(createdKeyPlaintext.value)
    toast.success('Copiado', 'A key foi copiada pra área de transferência')
  } catch {
    toast.error('Erro', 'Não foi possível copiar — selecione manualmente')
  }
}

const closeKeyResult = () => {
  newKeyResultOpen.value = false
  createdKeyPlaintext.value = ''
}

const askRevokeKey = (k: APIKey) => {
  revokingKey.value = k
  revokeKeyOpen.value = true
}

const onConfirmRevokeKey = async () => {
  if (!revokingKey.value) return
  revokingKeyNow.value = true
  try {
    await apiKeysAPI.revoke(revokingKey.value.id)
    apiKeys.value = apiKeys.value.map((k) =>
      k.id === revokingKey.value!.id ? { ...k, revoked_at: new Date().toISOString() } : k
    )
    toast.success('Sucesso', 'Key revogada')
    revokeKeyOpen.value = false
    revokingKey.value = null
  } catch (error: any) {
    toast.error('Erro', error.message || 'Falha ao revogar')
  } finally {
    revokingKeyNow.value = false
  }
}

watch(() => tenantStore.switchVersion, () => loadAPIKeys())
onMounted(loadAPIKeys)
</script>

<style scoped>
.card { background: var(--surface-2); border: 1px solid var(--border); border-radius: 12px; padding: 1.5rem; box-shadow: var(--shadow-card); }
.card-header { display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 0.75rem; margin-bottom: 1rem; }
.card-title { font-family: 'Instrument Sans', sans-serif; font-size: var(--fs-xl); font-weight: var(--fw-semibold); color: var(--text-strong); margin-bottom: 0.5rem; }
.card-hint { font-size: var(--fs-md); color: var(--text-2); margin-bottom: 1rem; line-height: 1.5; }
.card-hint code { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; background: var(--surface-soft); padding: 0.0625rem 0.375rem; border-radius: 4px; }
.muted { color: var(--text-3); font-size: var(--fs-md); font-style: italic; }
.text-danger { color: var(--danger); }
.account-list { display: flex; flex-direction: column; gap: 0.5rem; }
.account-row { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 0.75rem; padding: 0.75rem 1rem; border: 1px solid var(--border-soft); border-radius: 10px; background: var(--surface-soft-2); }
.account-info { display: flex; flex-direction: column; gap: 0.25rem; }
.account-name { display: flex; align-items: center; gap: 0.5rem; font-weight: var(--fw-semibold); color: var(--text-1); }
.account-meta { font-size: var(--fs-sm); color: var(--text-3); }
.account-meta code { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; background: var(--surface-soft); padding: 0.0625rem 0.375rem; border-radius: 4px; }
.form { display: flex; flex-direction: column; gap: 0.875rem; }
.form-field { display: flex; flex-direction: column; gap: 0.375rem; }
.form-label { font-size: var(--fs-sm); font-weight: var(--fw-medium); color: var(--text-2); }
.form-select { width: 100%; padding: 0.625rem 0.875rem; background: var(--surface-deep); border: 1px solid var(--border); border-radius: 8px; color: var(--text-1); font-size: var(--fs-lg); outline: none; }
.key-secret { display: flex; align-items: center; gap: 0.5rem; padding: 0.75rem 0.875rem; background: var(--surface-deep); border: 1px solid var(--brand-border); border-radius: 8px; }
.key-secret code { flex: 1; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: var(--fs-base); color: var(--brand); word-break: break-all; }
</style>

