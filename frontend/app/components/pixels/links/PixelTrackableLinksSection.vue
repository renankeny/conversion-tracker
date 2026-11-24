<template>
  <div class="links-section">
    <div class="top-bar">
      <div class="top-info">
        <h4 class="sec-title">Links Rastreáveis</h4>
        <p class="sec-desc">
          Gere links curtos para Bio do Instagram, Stories, YouTube ou anúncios diretos que redirecionam para o WhatsApp conectado e rastreiam cliques, conversas e vendas.
        </p>
      </div>
      <BaseButton v-if="canEdit" variant="primary" size="sm" @click="openCreate">
        <template #iconLeft><Plus :size="15" /></template>
        Novo Link
      </BaseButton>
    </div>

    <div v-if="overview && overview.total_links > 0" class="summary-strip">
      <div class="sum-cell">
        <span class="sum-lbl">Cliques totais</span>
        <strong class="sum-val">{{ overview.total_clicks }}</strong>
      </div>
      <div class="sum-cell">
        <span class="sum-lbl">Conversas geradas</span>
        <strong class="sum-val">{{ overview.total_chats }}</strong>
      </div>
      <div class="sum-cell">
        <span class="sum-lbl">Vendas atribuídas</span>
        <strong class="sum-val success">{{ overview.total_sales }}</strong>
      </div>
    </div>

    <div v-if="loading && !links.length" class="empty-box">
      <p class="empty-sub">Carregando links rastreáveis...</p>
    </div>

    <div v-else-if="!links.length" class="empty-box">
      <p class="empty-title">Nenhum link rastreável criado neste pixel</p>
      <p class="empty-sub">
        Ideal para colocar na Bio do Instagram ou Stories sem precisar de página de vendas.
      </p>
      <BaseButton v-if="canEdit" variant="outline" size="sm" @click="openCreate">
        Criar primeiro link rastreável
      </BaseButton>
    </div>

    <div v-else class="links-list">
      <PixelTrackableLinkRow
        v-for="item in links"
        :key="item.id"
        :link="item"
        :instances="pixelInstances"
        :can-edit="canEdit"
        :copied="copiedId === item.id"
        :currency="pixel.tracker_config?.product_currency || 'BRL'"
        @copy="copyLink"
        @edit="openEdit"
        @delete="askDelete"
      />
    </div>

    <PixelTrackableLinkModal
      v-model="modalOpen"
      :pixel="pixel"
      :editing="editingLink"
      :instances="pixelInstances"
      :saving="saving"
      @submit="onSave"
    />

    <ConfirmModal
      v-model="deleteOpen"
      title="Excluir link rastreável?"
      :message="deletingLink ? `O atalho &quot;/w/${deletingLink.slug}&quot; deixará de funcionar imediatamente. As conversas e vendas já atribuídas permanecem salvas.` : ''"
      confirm-label="Excluir link"
      variant="danger"
      :loading="deleting"
      @confirm="onConfirmDelete"
    />
  </div>
</template>

<script setup lang="ts">
import { Plus } from 'lucide-vue-next'
import ConfirmModal from '~/components/ConfirmModal.vue'
import PixelTrackableLinkRow from '~/components/pixels/links/PixelTrackableLinkRow.vue'
import PixelTrackableLinkModal from '~/components/pixels/links/PixelTrackableLinkModal.vue'
import { useWhatsAppStore } from '~/stores/useWhatsAppStore'
import { useLiveEventsStore } from '~/stores/useLiveEventsStore'
import { errorMessage } from '~/utils/errorMessage'
import type { Pixel, TrackableLink, TrackableLinkInput, TrackableLinksOverview } from '#shared/types'

const props = withDefaults(defineProps<{
  pixel: Pixel
  canEdit?: boolean
}>(), {
  canEdit: true,
})

const emit = defineEmits<{
  'count-updated': [count: number]
}>()

const api = useTrackableLinks()
const waStore = useWhatsAppStore()
const liveStore = useLiveEventsStore()
const toast = useToast()

const overview = ref<TrackableLinksOverview | null>(null)
const loading = ref(false)
const modalOpen = ref(false)
const editingLink = ref<TrackableLink | null>(null)
const saving = ref(false)
const deleteOpen = ref(false)
const deletingLink = ref<TrackableLink | null>(null)
const deleting = ref(false)
const copiedId = ref<number | null>(null)

const links = computed(() => overview.value?.links || [])
const pixelInstances = computed(() =>
  waStore.instances.filter((i) => i.pixel_id === props.pixel.id),
)

const load = async () => {
  if (!props.pixel?.id) return
  loading.value = true
  try {
    const [res] = await Promise.all([
      api.listByPixel(props.pixel.id),
      waStore.fetchAll(true).catch(() => {}),
    ])
    overview.value = res
    emit('count-updated', res.total_links || 0)
  } catch (e: unknown) {
    toast.error('Erro', errorMessage(e, 'Falha ao carregar links rastreáveis.'))
  } finally {
    loading.value = false
  }
}

watch(() => props.pixel?.id, () => { void load() }, { immediate: true })

let offWaStatus: (() => void) | null = null
onMounted(() => {
  offWaStatus = liveStore.onCustomEvent('whatsapp_status', (d: any) => {
    if (d?.instance_id) waStore.updateInstanceStatus(d.instance_id, d.status, d.owner_jid, d.profile_name, d.profile_pic_url)
  })
})
onUnmounted(() => offWaStatus?.())

const openCreate = () => { void waStore.fetchAll(true); editingLink.value = null; modalOpen.value = true }
const openEdit = (l: TrackableLink) => { void waStore.fetchAll(true); editingLink.value = l; modalOpen.value = true }
const askDelete = (l: TrackableLink) => { deletingLink.value = l; deleteOpen.value = true }

const copyLink = async (l: TrackableLink) => {
  const url = l.public_url || `${window.location.origin}/w/${l.slug}`
  try {
    await navigator.clipboard.writeText(url)
    copiedId.value = l.id
    toast.success('Link copiado!', url)
    setTimeout(() => { if (copiedId.value === l.id) copiedId.value = null }, 2000)
  } catch {
    toast.error('Erro', 'Não foi possível copiar automaticamente.')
  }
}

const onSave = async (input: TrackableLinkInput) => {
  saving.value = true
  try {
    if (editingLink.value) {
      await api.update(props.pixel.id, editingLink.value.id, input)
      toast.success('Link atualizado', 'Alterações salvas com sucesso.')
    } else {
      await api.create(props.pixel.id, input)
      toast.success('Link criado!', 'Pronto para colar na Bio, Stories ou anúncios.')
    }
    modalOpen.value = false
    await load()
  } catch (e: unknown) {
    toast.error('Erro ao salvar link', errorMessage(e, 'Verifique os campos e tente novamente.'))
  } finally {
    saving.value = false
  }
}

const onConfirmDelete = async () => {
  if (!deletingLink.value) return
  deleting.value = true
  try {
    await api.remove(props.pixel.id, deletingLink.value.id)
    toast.success('Link removido', 'O link rastreável foi excluído.')
    deleteOpen.value = false
    deletingLink.value = null
    await load()
  } catch (e: unknown) {
    toast.error('Erro ao remover', errorMessage(e, 'Tente novamente.'))
  } finally {
    deleting.value = false
  }
}
</script>

<style scoped>
.links-section { display: flex; flex-direction: column; gap: 1rem; }
.top-bar { display: flex; align-items: flex-start; justify-content: space-between; gap: 0.75rem; flex-wrap: wrap; }
.top-info { flex: 1; min-width: 240px; display: flex; flex-direction: column; gap: 0.2rem; }
.sec-title { font-size: var(--fs-lg); font-weight: var(--fw-semibold); color: var(--text-strong); margin: 0; }
.sec-desc { font-size: var(--fs-base); color: var(--text-2); margin: 0; line-height: 1.45; }
.summary-strip { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 0.625rem; padding: 0.75rem 1rem; background: var(--surface-deep); border: 1px solid var(--border-soft); border-radius: 10px; }
.sum-cell { display: flex; flex-direction: column; gap: 0.15rem; }
.sum-lbl { font-size: var(--fs-xs); color: var(--text-3); text-transform: uppercase; letter-spacing: 0.04em; }
.sum-val { font-size: var(--fs-xl); font-weight: var(--fw-bold); color: var(--text-strong); }
.sum-val.success { color: var(--success); }
.links-list { display: flex; flex-direction: column; gap: 0.625rem; }
.empty-box { display: flex; flex-direction: column; align-items: center; justify-content: center; text-align: center; gap: 0.5rem; padding: 2rem 1.25rem; background: var(--surface-soft-2); border: 1px dashed var(--border); border-radius: 10px; }
.empty-title { font-size: var(--fs-md); font-weight: var(--fw-semibold); color: var(--text-strong); margin: 0; }
.empty-sub { font-size: var(--fs-base); color: var(--text-2); max-width: 420px; margin: 0; }
@media (max-width: 560px) { .summary-strip { grid-template-columns: 1fr; } }
</style>
