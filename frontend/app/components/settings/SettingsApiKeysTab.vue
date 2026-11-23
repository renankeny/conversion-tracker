<script setup lang="ts">
import { Plus } from 'lucide-vue-next'
import ApiKeyFormModal from '~/components/settings/ApiKeyFormModal.vue'
import ApiKeyRowCard from '~/components/settings/ApiKeyRowCard.vue'
import type { Account, APIKey } from '#shared/types'
import { errorMessage } from '~/utils/errorMessage'

const apiKeysAPI = useAPIKeys()
const accountsAPI = useAccounts()
const tenantStore = useTenantStore()
const { isOwner } = useAuth()
const toast = useToast()

const apiKeys = ref<APIKey[]>([])
const accounts = ref<Account[]>([])
const loadingKeys = ref(false)
const formModalOpen = ref(false)
const editingKey = ref<APIKey | null>(null)
const savingKey = ref(false)

const revealedIds = reactive<Record<number, boolean>>({})
const copiedId = ref<number | null>(null)

const revokeKeyOpen = ref(false)
const revokingKey = ref<APIKey | null>(null)
const revokingKeyNow = ref(false)

const regenKeyOpen = ref(false)
const regenTargetKey = ref<APIKey | null>(null)
const regeneratingNow = ref(false)

const loadAPIKeys = async () => {
  loadingKeys.value = true
  try {
    const res = await apiKeysAPI.list()
    apiKeys.value = res.data || []
  } catch (error: unknown) {
    toast.error('Erro', errorMessage(error, 'Falha ao listar chaves'))
  } finally {
    loadingKeys.value = false
  }
}

const openCreateModal = async () => {
  if (isOwner.value && !accounts.value.length) {
    try {
      const res = await accountsAPI.list()
      accounts.value = res.data || []
    } catch { /* ignore */ }
  }
  editingKey.value = null
  formModalOpen.value = true
}

const openEditModal = (k: APIKey) => {
  editingKey.value = k
  formModalOpen.value = true
}

const onSubmitForm = async (payload: { name: string; account_id?: number; scopes: string[]; read_only: boolean }) => {
  savingKey.value = true
  try {
    if (editingKey.value) {
      const res = await apiKeysAPI.update(editingKey.value.id, {
        name: payload.name,
        scopes: payload.scopes,
        read_only: payload.read_only,
      })
      apiKeys.value = apiKeys.value.map((item) => (item.id === res.key.id ? res.key : item))
      toast.success('Salvo', 'Nome e permissões da chave atualizados')
    } else {
      const res = await apiKeysAPI.create(payload)
      const created: APIKey = { ...res.key, plaintext: res.plaintext }
      apiKeys.value = [created, ...apiKeys.value]
      revealedIds[created.id] = true
      toast.success('Chave criada', 'Sua chave já está visível e pronta para copiar')
    }
    formModalOpen.value = false
  } catch (error: unknown) {
    toast.error('Erro', errorMessage(error, 'Falha ao salvar chave'))
  } finally {
    savingKey.value = false
  }
}

const copyPlaintext = async (k: APIKey) => {
  if (!k.plaintext) return
  await navigator.clipboard.writeText(k.plaintext)
  copiedId.value = k.id
  toast.success('Copiado', 'Chave copiada para a área de transferência')
  setTimeout(() => {
    if (copiedId.value === k.id) copiedId.value = null
  }, 1800)
}

const askRegenerate = (k: APIKey) => {
  regenTargetKey.value = k
  regenKeyOpen.value = true
}

const onConfirmRegenerate = async () => {
  if (!regenTargetKey.value) return
  regeneratingNow.value = true
  try {
    const res = await apiKeysAPI.regenerate(regenTargetKey.value.id)
    const fresh: APIKey = { ...res.key, plaintext: res.plaintext }
    apiKeys.value = apiKeys.value.map((item) => (item.id === fresh.id ? fresh : item))
    revealedIds[fresh.id] = true
    regenKeyOpen.value = false
    regenTargetKey.value = null
    toast.success('Segredo atualizado', 'Novo código gerado mantendo as permissões')
  } catch (error: unknown) {
    toast.error('Erro', errorMessage(error, 'Falha ao regerar chave'))
  } finally {
    regeneratingNow.value = false
  }
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
    toast.success('Sucesso', 'Chave revogada')
    revokeKeyOpen.value = false
    revokingKey.value = null
  } catch (error: unknown) {
    toast.error('Erro', errorMessage(error, 'Falha ao revogar'))
  } finally {
    revokingKeyNow.value = false
  }
}

watch(() => tenantStore.switchVersion, () => loadAPIKeys())
onMounted(loadAPIKeys)
</script>

<template>
  <section class="api-keys-section">
    <div class="card">
      <div class="card-header">
        <div>
          <h3 class="card-title">Chaves de API e MCP</h3>
          <p class="card-hint">
            A mesma chave funciona na <strong>API REST</strong> e no <strong>Servidor MCP</strong>.
            Você pode revelar, copiar e editar as permissões de cada chave quando quiser.
          </p>
        </div>
        <BaseButton variant="primary" size="sm" @click="openCreateModal">
          <Plus class="icon-xs" />
          Nova Chave
        </BaseButton>
      </div>

      <div v-if="loadingKeys" class="muted">Carregando chaves...</div>
      <div v-else-if="!apiKeys.length" class="muted">Nenhuma chave cadastrada. Clique em "Nova Chave" para começar.</div>

      <div v-else class="key-list">
        <ApiKeyRowCard
          v-for="k in apiKeys"
          :key="k.id"
          :item="k"
          :revealed="Boolean(revealedIds[k.id])"
          :copied="copiedId === k.id"
          @toggle-reveal="revealedIds[k.id] = !revealedIds[k.id]"
          @copy="copyPlaintext(k)"
          @edit="openEditModal(k)"
          @regenerate="askRegenerate(k)"
          @revoke="askRevokeKey(k)"
        />
      </div>
    </div>

    <ApiKeyFormModal
      v-model="formModalOpen"
      :editing-key="editingKey"
      :accounts="accounts"
      :is-owner="isOwner"
      :loading="savingKey"
      @submit="onSubmitForm"
    />

    <ConfirmModal
      v-model="regenKeyOpen"
      title="Regerar segredo da chave?"
      :message="regenTargetKey ? `Um novo código trk_... será gerado para '${regenTargetKey.name}' mantendo o nome e as permissões. O código antigo deixará de funcionar imediatamente.` : ''"
      confirm-label="Regerar agora"
      variant="primary"
      :loading="regeneratingNow"
      @confirm="onConfirmRegenerate"
    />

    <ConfirmModal
      v-model="revokeKeyOpen"
      title="Revogar Chave?"
      :message="revokingKey ? `A chave '${revokingKey.name}' (${revokingKey.key_prefix}...) deixará de funcionar imediatamente na API e no MCP.` : ''"
      confirm-label="Revogar"
      variant="danger"
      :loading="revokingKeyNow"
      @confirm="onConfirmRevokeKey"
    />
  </section>
</template>

<style scoped>
.card { background: var(--surface-2); border: 1px solid var(--border); border-radius: 12px; padding: 1.35rem; }
.card-header { display: flex; justify-content: space-between; align-items: flex-start; flex-wrap: wrap; gap: 0.75rem; margin-bottom: 1rem; }
.card-title { font-family: var(--font-sans); font-size: var(--fs-xl); font-weight: var(--fw-semibold); color: var(--text-strong); margin-bottom: 0.25rem; }
.card-hint { font-size: var(--fs-md); color: var(--text-2); line-height: 1.5; max-width: 64ch; }
.muted { color: var(--text-3); font-size: var(--fs-md); }
.icon-xs { width: 0.85rem; height: 0.85rem; }
.key-list { display: flex; flex-direction: column; gap: 0.75rem; }
</style>
