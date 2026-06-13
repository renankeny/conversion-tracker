<template>
  <div class="pixels-page">
    <div class="page-header">
      <div>
        <h1 class="page-title">Pixels</h1>
        <p class="page-subtitle">
          Gerencie os pixels de conversão da Meta. Cada pixel pode espelhar eventos para
          múltiplos pixel IDs simultaneamente.
        </p>
      </div>
      <BaseButton variant="primary" @click="openCreateModal">
        <template #iconLeft>
          <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
            <path
              fill-rule="evenodd"
              d="M10 5a1 1 0 011 1v3h3a1 1 0 110 2h-3v3a1 1 0 11-2 0v-3H6a1 1 0 110-2h3V6a1 1 0 011-1z"
              clip-rule="evenodd"
            />
          </svg>
        </template>
        Novo Pixel
      </BaseButton>
    </div>

    <div v-if="loading && !pixels.length" class="loading-state">
      <div class="spinner" />
      <p>Carregando pixels...</p>
    </div>

    <div v-else-if="!pixels.length" class="empty-state">
      <div class="empty-icon">
        <svg xmlns="http://www.w3.org/2000/svg" class="h-12 w-12" viewBox="0 0 20 20" fill="currentColor">
          <path d="M5 3a2 2 0 00-2 2v2a2 2 0 002 2h2a2 2 0 002-2V5a2 2 0 00-2-2H5zM5 11a2 2 0 00-2 2v2a2 2 0 002 2h2a2 2 0 002-2v-2a2 2 0 00-2-2H5zM11 5a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V5zM11 13a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z" />
        </svg>
      </div>
      <h2 class="empty-title">Nenhum pixel cadastrado</h2>
      <p class="empty-description">
        Crie seu primeiro pixel para começar a rastrear conversões.
      </p>
      <BaseButton variant="primary" @click="openCreateModal">Criar Pixel</BaseButton>
    </div>

    <div v-else class="pixels-grid">
      <PixelCard
        v-for="pixel in pixels"
        :key="pixel.id"
        :pixel="pixel"
        @edit="openEditModal"
        @delete="openDeleteModal"
        @install="openInstallModal"
      />
    </div>

    <!-- Modal Create/Edit -->
    <BaseModal
      v-model="formModalOpen"
      :title="editing ? 'Editar Pixel' : 'Novo Pixel'"
      :description="editing
        ? 'Altere as configurações do pixel. Tokens em branco mantêm os atuais.'
        : 'Configure um novo pixel para começar a enviar eventos.'"
      size="lg"
    >
      <PixelForm ref="formRef" :pixel="editing" @submit="onFormSubmit" />

      <template #footer>
        <BaseButton variant="ghost" @click="formModalOpen = false">Cancelar</BaseButton>
        <BaseButton variant="primary" :loading="saving" @click="formRef?.submit()">
          {{ editing ? 'Salvar alterações' : 'Criar pixel' }}
        </BaseButton>
      </template>
    </BaseModal>

    <!-- Modal Install -->
    <PixelInstallModal v-model="installModalOpen" :pixel="installingPixel" />

    <!-- Modal Delete -->
    <ConfirmModal
      v-model="deleteModalOpen"
      title="Excluir pixel?"
      description="Esta operação é irreversível"
      :message="deletingPixel
        ? `O pixel \&quot;${deletingPixel.name}\&quot; será removido. O vínculo histórico em tracked_users e sent_events não será apagado, mas novos eventos pararão de ser aceitos.`
        : ''"
      confirm-label="Excluir definitivamente"
      variant="danger"
      :loading="deleting"
      require-typing
      :typing-phrase="deletingPixel?.name || ''"
      @confirm="onConfirmDelete"
      @cancel="deletingPixel = null"
    />
  </div>
</template>

<script setup lang="ts">
import PixelCard from '~/components/pixels/PixelCard.vue'
import PixelForm from '~/components/pixels/PixelForm.vue'
import PixelInstallModal from '~/components/pixels/PixelInstallModal.vue'
import type { Pixel, PixelInput } from '#shared/types'

definePageMeta({
  layout: 'app',
  middleware: 'auth',
})

useHead({ title: 'Pixels - TrackFind' })

const pixelsApi = usePixels()
const toast = useToast()

const pixels = ref<Pixel[]>([])
const loading = ref(false)

const formModalOpen = ref(false)
const editing = ref<Pixel | null>(null)
const saving = ref(false)
const formRef = ref<{ submit: () => void } | null>(null)

const deleteModalOpen = ref(false)
const deletingPixel = ref<Pixel | null>(null)
const deleting = ref(false)

const installModalOpen = ref(false)
const installingPixel = ref<Pixel | null>(null)

const fetchPixels = async () => {
  loading.value = true
  try {
    pixels.value = await pixelsApi.list()
  } catch (error: any) {
    toast.error('Erro ao carregar', error.message || 'Não foi possível listar os pixels')
  } finally {
    loading.value = false
  }
}

const openCreateModal = () => {
  editing.value = null
  formModalOpen.value = true
}

const openEditModal = (pixel: Pixel) => {
  editing.value = pixel
  formModalOpen.value = true
}

const openDeleteModal = (pixel: Pixel) => {
  deletingPixel.value = pixel
  deleteModalOpen.value = true
}

const openInstallModal = (pixel: Pixel) => {
  installingPixel.value = pixel
  installModalOpen.value = true
}

const onFormSubmit = async (input: PixelInput) => {
  saving.value = true
  try {
    if (editing.value) {
      const updated = await pixelsApi.update(editing.value.id, input)
      const idx = pixels.value.findIndex((p) => p.id === updated.id)
      if (idx !== -1) pixels.value[idx] = updated
      toast.success('Pixel atualizado', `"${updated.name}" salvo com sucesso`)
    } else {
      const created = await pixelsApi.create(input)
      pixels.value = [created, ...pixels.value]
      toast.success('Pixel criado', `"${created.name}" pronto para receber eventos`)
    }
    formModalOpen.value = false
    editing.value = null
  } catch (error: any) {
    toast.error('Erro ao salvar', error.message || 'Não foi possível salvar o pixel')
  } finally {
    saving.value = false
  }
}

const onConfirmDelete = async () => {
  if (!deletingPixel.value) return
  deleting.value = true
  try {
    await pixelsApi.remove(deletingPixel.value.id)
    pixels.value = pixels.value.filter((p) => p.id !== deletingPixel.value!.id)
    toast.success('Pixel excluído', `"${deletingPixel.value.name}" foi removido`)
    deleteModalOpen.value = false
    deletingPixel.value = null
  } catch (error: any) {
    toast.error('Erro ao excluir', error.message || 'Não foi possível excluir o pixel')
  } finally {
    deleting.value = false
  }
}

onMounted(() => {
  fetchPixels()
})
</script>

<style scoped>
.pixels-page {
  display: flex;
  flex-direction: column;
  gap: 2rem;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 1rem;
  flex-wrap: wrap;
}

.page-header > div:first-child {
  flex: 1;
  min-width: 280px;
}

.page-title {
  font-family: 'Instrument Sans', sans-serif;
  font-size: 1.875rem;
  font-weight: 700;
  color: var(--text-strong);
  margin-bottom: 0.25rem;
}

.page-subtitle {
  color: var(--text-2);
  font-size: 0.9375rem;
  max-width: 640px;
}

.pixels-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(340px, 1fr));
  gap: 1rem;
}

.loading-state,
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 4rem 2rem;
  background: var(--surface-2);
  border: 1px dashed var(--border);
  border-radius: 12px;
  gap: 0.75rem;
  text-align: center;
}

.empty-icon {
  color: var(--text-3);
  margin-bottom: 0.5rem;
}

.empty-title {
  font-family: 'Instrument Sans', sans-serif;
  font-size: 1.5rem;
  font-weight: 600;
  color: var(--text-strong);
}

.empty-description {
  color: var(--text-2);
  max-width: 420px;
  margin-bottom: 0.5rem;
}

.spinner {
  width: 2rem;
  height: 2rem;
  border: 3px solid var(--border);
  border-top-color: var(--brand);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
