<template>
  <div class="wa-page">
    <div class="page-header">
      <div>
        <h1 class="page-title">WhatsApp & Campanhas CTWA</h1>
        <p class="page-subtitle">
          Conecte seu WhatsApp via QR Code (Evolution Go), API Premium ou Uazapi para rastrear anúncios Click-to-WhatsApp, protocolos de site/bio e gatilhos automáticos de venda.
        </p>
      </div>
      <BaseButton
        variant="primary"
        @click="openCreate"
      >
        Nova conexão WhatsApp
      </BaseButton>
    </div>

    <WhatsAppServerConfigCard
      :server="server"
      @saved="loadData(true)"
    />

    <div v-if="loading && !instances.length" class="state-box">
      <div class="spinner" />
      <p>Carregando conexões...</p>
    </div>

    <div v-else-if="!instances.length" class="state-box">
      <h2 class="empty-title">Nenhum aparelho WhatsApp conectado</h2>
      <p class="empty-desc">
        Crie uma conexão via API Premium, Uazapi (Webhook) ou Conexão Direta via QR Code.
      </p>
      <BaseButton variant="primary" @click="openCreate">
        Criar primeira conexão
      </BaseButton>
    </div>

    <div v-else class="inst-grid">
      <WhatsAppInstanceCard
        v-for="inst in instances"
        :key="inst.id"
        :instance="inst"
        :refreshing="refreshingId === inst.id"
        :disconnecting="disconnectingId === inst.id"
        @connect="openConnect"
        @disconnect="onDisconnect"
        @refresh="onRefreshInstance"
        @edit="openEdit"
        @delete="askDelete"
      />
    </div>

    <WhatsAppInstanceFormModal
      v-model="showFormModal"
      :editing="editingInstance"
      :pixels="pixels"
      :stages="crmStages"
      :saving="savingInstance"
      :has-evolution-server="Boolean(server?.has_api_key)"
      @submit="onSaveInstance"
    />

    <WhatsAppConnectModal
      v-model="showConnectModal"
      :instance="connectingInstance"
      @connected="loadData(true)"
    />

    <ConfirmModal
      v-model="showDeleteModal"
      title="Excluir conexão WhatsApp?"
      description="A instância será removida da Evolution Go e do painel."
      :message="deletingInstance ? `Tem certeza que deseja remover &quot;${deletingInstance.name}&quot;?` : ''"
      confirm-label="Excluir conexão"
      variant="danger"
      :loading="deletingLoading"
      @confirm="onConfirmDelete"
    />
  </div>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import type { WhatsAppInstance, WhatsAppInstanceInput } from '#shared/types'
import WhatsAppServerConfigCard from '~/components/whatsapp/WhatsAppServerConfigCard.vue'
import WhatsAppInstanceCard from '~/components/whatsapp/WhatsAppInstanceCard.vue'
import WhatsAppInstanceFormModal from '~/components/whatsapp/WhatsAppInstanceFormModal.vue'
import WhatsAppConnectModal from '~/components/whatsapp/WhatsAppConnectModal.vue'
import { useWhatsAppStore } from '~/stores/useWhatsAppStore'
import { usePixelStore } from '~/stores/usePixelStore'
import { useTenantStore } from '~/stores/useTenantStore'
import { useLiveEventsStore } from '~/stores/useLiveEventsStore'
import { errorMessage } from '~/utils/errorMessage'

definePageMeta({ layout: 'app', middleware: 'auth' })
const { formatTitle } = useBranding()
useHead({ title: computed(() => formatTitle('WhatsApp (CTWA)')) })

const api = useAPI()
const wa = useWhatsApp()
const waStore = useWhatsAppStore()
const pixelsStore = usePixelStore()
const tenantStore = useTenantStore()
const liveStore = useLiveEventsStore()
const toast = useToast()

const { server, instances, loading } = storeToRefs(waStore)
const { pixels } = storeToRefs(pixelsStore)

const crmStages = ref<Array<{ id: number; name: string }>>([])
const showFormModal = ref(false)
const editingInstance = ref<WhatsAppInstance | null>(null)
const savingInstance = ref(false)
const showConnectModal = ref(false)
const connectingInstance = ref<WhatsAppInstance | null>(null)
const showDeleteModal = ref(false)
const deletingInstance = ref<WhatsAppInstance | null>(null)
const deletingLoading = ref(false)
const refreshingId = ref<number | null>(null)
const disconnectingId = ref<number | null>(null)

const loadData = async (force = false) => {
  try {
    await Promise.all([
      waStore.fetchAll(force),
      pixelsStore.fetch(force),
      api.get<{ stages?: Array<{ id: number; name: string }> } | Array<{ id: number; name: string }>>('/admin/ext-api/crm/stages')
        .then((res) => { crmStages.value = Array.isArray(res) ? res : (res?.stages ?? []) })
        .catch(() => {}),
    ])
  } catch (e: unknown) {
    toast.error('Erro ao carregar WhatsApp', errorMessage(e, 'Tente novamente.'))
  }
}

const openCreate = () => { editingInstance.value = null; showFormModal.value = true }
const openEdit = (inst: WhatsAppInstance) => { editingInstance.value = inst; showFormModal.value = true }
const openConnect = (inst: WhatsAppInstance) => { connectingInstance.value = inst; showConnectModal.value = true }
const askDelete = (inst: WhatsAppInstance) => { deletingInstance.value = inst; showDeleteModal.value = true }

const onSaveInstance = async (input: WhatsAppInstanceInput) => {
  savingInstance.value = true
  try {
    if (editingInstance.value) {
      await wa.updateInstance(editingInstance.value.id, input)
      toast.success('Conexão atualizada', 'Alterações salvas.')
      showFormModal.value = false
    } else {
      const created = await wa.createInstance(input)
      showFormModal.value = false
      connectingInstance.value = created
      showConnectModal.value = true
    }
  } catch (e: unknown) {
    toast.error('Erro ao salvar', errorMessage(e, 'Falha ao configurar instância.'))
  } finally {
    savingInstance.value = false
  }
}

const onRefreshInstance = async (inst: WhatsAppInstance) => {
  refreshingId.value = inst.id
  try {
    await wa.refreshStatus(inst.id)
  } catch (e: unknown) {
    toast.error('Erro ao sincronizar', errorMessage(e, 'Verifique o servidor Evolution Go.'))
  } finally {
    refreshingId.value = null
  }
}

const onDisconnect = async (inst: WhatsAppInstance) => {
  disconnectingId.value = inst.id
  try {
    await wa.disconnectInstance(inst.id)
    toast.success('Desconectado', `Sessão de "${inst.name}" encerrada.`)
  } catch (e: unknown) {
    toast.error('Erro ao desconectar', errorMessage(e, 'Tente novamente.'))
  } finally {
    disconnectingId.value = null
  }
}

const onConfirmDelete = async () => {
  if (!deletingInstance.value) return
  deletingLoading.value = true
  try {
    await wa.deleteInstance(deletingInstance.value.id)
    toast.success('Conexão removida', `"${deletingInstance.value.name}" foi excluída.`)
    showDeleteModal.value = false
    deletingInstance.value = null
  } catch (e: unknown) {
    toast.error('Erro ao excluir', errorMessage(e, 'Falha ao remover instância.'))
  } finally {
    deletingLoading.value = false
  }
}

watch(() => tenantStore.switchVersion, () => {
  waStore.invalidate()
  void loadData(true)
})

let offLiveStatus: (() => void) | null = null
onMounted(() => {
  void loadData()
  offLiveStatus = liveStore.onCustomEvent('whatsapp_status', (d: any) => {
    if (d?.instance_id) {
      waStore.updateInstanceStatus(d.instance_id, d.status, d.owner_jid, d.profile_name, d.profile_pic_url)
    }
  })
})
onUnmounted(() => offLiveStatus?.())
</script>

<style scoped>
.wa-page { display: flex; flex-direction: column; gap: 1.75rem; }
.page-header { display: flex; justify-content: space-between; align-items: flex-start; gap: 1rem; flex-wrap: wrap; }
.page-header > div:first-child { flex: 1; min-width: 260px; }
.page-title { font-family: var(--font-heading); font-size: var(--fs-3xl); font-weight: var(--fw-bold); color: var(--text-strong); margin: 0 0 0.25rem; }
.page-subtitle { color: var(--text-2); font-size: var(--fs-lg); max-width: 740px; line-height: 1.5; margin: 0; }
.inst-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(min(100%, 360px), 1fr)); gap: 1.25rem; }
.state-box {
  display: flex; flex-direction: column; align-items: center; justify-content: center;
  padding: 3.5rem 1.5rem; background: var(--surface-2); border: 1px dashed var(--border);
  border-radius: 12px; gap: 0.75rem; text-align: center;
}
.empty-title { font-family: var(--font-heading); font-size: var(--fs-2xl); font-weight: var(--fw-semibold); color: var(--text-strong); margin: 0; }
.empty-desc { color: var(--text-2); font-size: var(--fs-md); max-width: 480px; margin: 0 0 0.25rem; line-height: 1.5; }
.spinner { width: 2rem; height: 2rem; border: 3px solid var(--border); border-top-color: var(--brand); border-radius: 50%; animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
</style>
