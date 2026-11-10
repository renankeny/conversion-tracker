<template>
  <div class="custom-webhooks-page">
    <div class="page-header">
      <div>
        <h1 class="page-title">Checkouts Personalizados</h1>
        <p class="page-subtitle">Crie integrações com qualquer plataforma de pagamento mapeando os campos do JSON.</p>
      </div>
      <BaseButton @click="openModal()">Novo Checkout</BaseButton>
    </div>

    <div v-if="loading" class="loading-state">
      Carregando...
    </div>
    <div v-else-if="items.length === 0" class="empty-state">
      <p>Nenhuma integração personalizada criada ainda.</p>
    </div>
    <div v-else class="webhooks-grid">
      <CustomWebhookCard
        v-for="item in items"
        :key="item.id"
        :item="item"
        @logs="openLogs"
        @edit="openModal"
        @delete="confirmDelete"
      />
    </div>

    <CustomWebhookTestModal
      v-model="showLogs"
      :webhook-id="logsWebhookId"
      :webhook-name="logsWebhookName"
    />

    <CustomWebhookModal
      v-model="showModal"
      :editing-item="editingItem"
      :saving="saving"
      @save="onSave"
    />

    <ConfirmModal
      v-model="showDeleteConfirm"
      title="Excluir Integração"
      message="Tem certeza que deseja excluir esta integração personalizada?"
      confirm-label="Excluir"
      variant="danger"
      @confirm="executeDelete"
    />
  </div>
</template>

<script setup lang="ts">
import CustomWebhookCard from '~/components/integrations/CustomWebhookCard.vue'
import CustomWebhookModal from '~/components/integrations/CustomWebhookModal.vue'
import CustomWebhookTestModal from '~/components/integrations/CustomWebhookTestModal.vue'
import { useCustomWebhooksStore } from '~/stores/customWebhooks'

definePageMeta({ layout: 'app', middleware: 'auth' })

const { formatTitle } = useBranding()
useHead({ title: computed(() => formatTitle('Checkouts Personalizados')) })

const store = useCustomWebhooksStore()
const tenantStore = useTenantStore()
const toast = useToast()

const loading = computed(() => store.loading)
const items = computed(() => store.items)

const showModal = ref(false)
const editingItem = ref<any | null>(null)
const saving = ref(false)

const showLogs = ref(false)
const logsWebhookName = ref('')
const logsWebhookId = ref('')

const openLogs = (item: any) => {
  logsWebhookId.value = String(item.id)
  logsWebhookName.value = item.name
  showLogs.value = true
}

const openModal = (item?: any) => {
  editingItem.value = item || null
  showModal.value = true
}

const onSave = async (payload: { id: string | null; name: string; is_global: boolean; mapping_config: any }) => {
  saving.value = true
  try {
    if (payload.id) {
      await store.updateItem(payload.id, { name: payload.name, mapping_config: payload.mapping_config })
    } else {
      await store.createItem({ name: payload.name, is_global: payload.is_global, mapping_config: payload.mapping_config })
    }
    showModal.value = false
    toast.success('Sucesso', 'Integração salva com sucesso!')
  } catch (e: any) {
    toast.error('Erro ao salvar', e?.message || 'Não foi possível salvar a integração.')
  } finally {
    saving.value = false
  }
}

const showDeleteConfirm = ref(false)
const deletingId = ref<string | null>(null)

const confirmDelete = (id: string) => {
  deletingId.value = id
  showDeleteConfirm.value = true
}

const executeDelete = async () => {
  if (!deletingId.value) return
  try {
    await store.deleteItem(deletingId.value)
    showDeleteConfirm.value = false
    toast.success('Excluído', 'Integração excluída com sucesso.')
  } catch (e: any) {
    toast.error('Erro ao excluir', e?.message || 'Não foi possível excluir.')
  } finally {
    deletingId.value = null
  }
}

watch(() => tenantStore.switchVersion, () => {
  showModal.value = false
  store.fetchItems(true)
})

onMounted(() => {
  store.fetchItems()
})
</script>

<style scoped>
.custom-webhooks-page { display: flex; flex-direction: column; gap: 1.5rem; }
.page-header { display: flex; justify-content: space-between; align-items: flex-end; flex-wrap: wrap; gap: 1rem; }
.page-title { font-family: 'Instrument Sans'; font-size: var(--fs-3xl); font-weight: var(--fw-bold); }
.page-subtitle { color: var(--text-2); font-size: var(--fs-lg); }
.webhooks-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(min(100%, 320px), 1fr)); gap: 1rem; }
</style>
