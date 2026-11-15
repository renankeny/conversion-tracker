<template>
  <div class="pixels-page">
    <div class="page-header">
      <div>
        <h1 class="page-title">Pixels</h1>
        <p class="page-subtitle">
          Gerencie seus scripts de rastreamento, regras de captura automática e canais de conversão integrados.
        </p>
      </div>
      <BaseButton v-if="canManagePixels" variant="primary" @click="openCreateModal">
        <template #iconLeft>
          <Plus :size="16" />
        </template>
        Novo Pixel
      </BaseButton>
    </div>

    <PixelsToolbar
      v-if="pixels.length > 0"
      :pixels="pixels"
      v-model:search="searchQuery"
      v-model:status-filter="statusFilter"
    />

    <div v-if="loading && !pixels.length" class="loading-state">
      <div class="spinner" />
      <p>Carregando pixels...</p>
    </div>

    <div v-else-if="!pixels.length" class="empty-state">
      <h2 class="empty-title">Nenhum pixel cadastrado</h2>
      <p class="empty-description">Crie seu primeiro pixel para começar a rastrear leads, vendas e conversões.</p>
      <BaseButton v-if="canManagePixels" variant="primary" @click="openCreateModal">Criar Pixel</BaseButton>
    </div>

    <div v-else-if="!filteredPixels.length" class="empty-state">
      <h2 class="empty-title">Nenhum pixel encontrado</h2>
      <p class="empty-description">Nenhum pixel corresponde aos filtros aplicados.</p>
      <BaseButton variant="ghost" @click="clearFilters">Limpar filtros</BaseButton>
    </div>

    <div v-else class="pixels-grid">
      <PixelCard
        v-for="pixel in filteredPixels"
        :key="pixel.id"
        :pixel="pixel"
        :can-edit="canManagePixels"
        :can-transfer="canTransferPixels"
        :account-name="showCompanyBadge ? (accountNamesById[pixel.account_id] || '') : ''"
        :meta-account="pixel.meta_account_id ? (metaById[pixel.meta_account_id] || null) : null"
        :google-ads-account="pixel.google_ads_account_id ? (gadsById[pixel.google_ads_account_id] || null) : null"
        @edit="openEditModal"
        @delete="openDeleteModal"
        @install="openInstallModal"
        @transfer="openTransferModal"
      />
    </div>

    <BaseModal
      v-model="formModalOpen"
      :title="editing ? 'Editar Pixel' : 'Novo Pixel'"
      :description="editing ? 'Altere as configurações do pixel. Tokens em branco mantêm os atuais.' : 'Configure um novo pixel para começar a enviar eventos.'"
      size="lg"
    >
      <PixelForm ref="formRef" :pixel="editing" :initial-tab="formInitialTab" @submit="onFormSubmit" />
      <template #footer>
        <BaseButton variant="ghost" @click="formModalOpen = false">Cancelar</BaseButton>
        <BaseButton variant="primary" :loading="saving" @click="formRef?.submit()">
          {{ editing ? 'Salvar alterações' : 'Criar pixel' }}
        </BaseButton>
      </template>
    </BaseModal>

    <PixelInstallModal
      v-model="installModalOpen"
      :pixel="installingPixel"
      :can-edit="canManagePixels"
      @edit-config="onEditFromInstall"
      @token-regenerated="onTokenRegenerated"
    />

    <ConfirmModal
      v-model="deleteModalOpen"
      title="Excluir pixel?"
      description="Esta operação é irreversível"
      :message="deletingPixel ? `O pixel &quot;${deletingPixel.label || deletingPixel.name}&quot; será removido. O histórico de leads e conversões já registrado permanecerá salvo no painel, mas novos eventos deixarão de ser aceitos imediatamente.` : ''"
      confirm-label="Excluir definitivamente"
      variant="danger"
      :loading="deleting"
      require-typing
      :typing-phrase="deletingPixel?.name || ''"
      @confirm="onConfirmDelete"
      @cancel="deletingPixel = null"
    />

    <TransferPixelModal
      v-model="transferModalOpen"
      :pixel="transferringPixel"
      :saving="transferring"
      @confirm="onConfirmTransfer"
    />
  </div>
</template>

<script setup lang="ts">
import { Plus } from 'lucide-vue-next'
import PixelCard from '~/components/pixels/PixelCard.vue'
import PixelsToolbar from '~/components/pixels/PixelsToolbar.vue'
import PixelForm from '~/components/pixels/PixelForm.vue'
import PixelInstallModal from '~/components/pixels/PixelInstallModal.vue'
import TransferPixelModal from '~/components/pixels/TransferPixelModal.vue'

definePageMeta({ layout: 'app', middleware: 'auth' })
const { formatTitle } = useBranding()
useHead({ title: computed(() => formatTitle('Pixels')) })

const {
  pixels, loading, canManagePixels, canTransferPixels, showCompanyBadge,
  searchQuery, statusFilter, filteredPixels, clearFilters,
  accountNamesById, metaById, gadsById,
  formModalOpen, editing, saving, formRef, formInitialTab,
  deleteModalOpen, deletingPixel, deleting,
  installModalOpen, installingPixel,
  transferModalOpen, transferringPixel, transferring,
  openCreateModal, openEditModal, openDeleteModal, openInstallModal, openTransferModal,
  onEditFromInstall, onFormSubmit, onConfirmDelete, onTokenRegenerated, onConfirmTransfer,
} = usePixelsPage()
</script>

<style scoped>
.pixels-page { display: flex; flex-direction: column; gap: 1.5rem; }
.page-header { display: flex; justify-content: space-between; align-items: flex-start; gap: 1rem; flex-wrap: wrap; }
.page-header > div:first-child { flex: 1; min-width: 280px; }
.page-title { font-family: var(--font-heading); font-size: var(--fs-3xl); font-weight: var(--fw-bold); color: var(--text-strong); margin-bottom: 0.25rem; }
.page-subtitle { color: var(--text-2); font-size: var(--fs-lg); max-width: 640px; }
.pixels-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(min(100%, 320px), 1fr)); gap: 1rem; }
.loading-state, .empty-state { display: flex; flex-direction: column; align-items: center; justify-content: center; padding: 4rem 2rem; background: var(--surface-2); border: 1px dashed var(--border); border-radius: 12px; gap: 0.75rem; text-align: center; }
.empty-title { font-family: var(--font-heading); font-size: var(--fs-2xl); font-weight: var(--fw-semibold); color: var(--text-strong); }
.empty-description { color: var(--text-2); max-width: 420px; margin-bottom: 0.5rem; }
.spinner { width: 2rem; height: 2rem; border: 3px solid var(--border); border-top-color: var(--brand); border-radius: 50%; animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
</style>
