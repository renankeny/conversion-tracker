<template>
  <div class="public-crm-page">
    <div class="page-header">
      <div>
        <div class="title-wrap">
          <h1 class="page-title">CRM</h1>
          <span class="mode-pill" :class="allowEdit ? 'editable' : 'readonly'">
            {{ allowEdit ? 'Modo comercial ativo' : 'Somente visualização' }}
          </span>
        </div>
        <p class="page-subtitle">
          {{ state.pixelLabel ? `Funil comercial e leads de ${state.pixelLabel}` : 'Acompanhamento do funil de vendas' }}
        </p>
      </div>

      <div class="header-actions">
        <BaseButton
          variant="ghost"
          size="sm"
          :title="privacyMode ? 'Mostrar valores financeiros' : 'Ocultar valores financeiros'"
          @click="privacyMode = !privacyMode"
        >
          <EyeOff v-if="privacyMode" :size="15" />
          <Eye v-else :size="15" />
        </BaseButton>
        <BaseButton v-if="allowEdit" variant="primary" size="sm" @click="openCreateLeadModal()">
          <Plus :size="15" />
          <span>Novo Lead</span>
        </BaseButton>
        <BaseButton variant="outline" size="sm" :loading="isLoading" @click="fetchBoard(false)">
          Atualizar
        </BaseButton>
      </div>
    </div>

    <div v-if="loadError" class="error-banner">
      <span>{{ loadError }}</span>
      <BaseButton variant="primary" size="sm" @click="fetchBoard(false)">Tentar novamente</BaseButton>
    </div>

    <CrmMetricsStrip
      :total-pipeline-value="totalPipelineValue"
      :total-leads-count="isFilterActive ? filteredTotalCount : totalLeadsCount"
      :total-won-value="totalWonValue"
      :average-ticket="averageTicket"
      :blurred="privacyMode"
    />

    <CrmFilterBar
      v-model:search-query="searchQuery"
      v-model:selected-tag-filter="selectedTagFilter"
      v-model:selected-origin-filter="selectedOriginFilter"
      :all-tags="allTags"
      :is-filter-active="isFilterActive"
      :filtered-total-count="filteredTotalCount"
      :total-leads-count="totalLeadsCount"
      @clear="clearFilters"
    />

    <div v-if="!isLoading && isFilterActive && filteredTotalCount === 0" class="empty-filter-banner">
      <span>Nenhum lead encontrado com os filtros aplicados.</span>
      <BaseButton variant="outline" size="sm" @click="clearFilters">Limpar filtros</BaseButton>
    </div>

    <div v-if="isLoading" class="loading-state">
      <Loader2 :size="34" class="spin icon-brand" />
      <p>Carregando funil de vendas...</p>
    </div>

    <div v-else class="kanban-board">
      <CrmStageColumn
        v-for="stage in stages"
        :key="stage.id"
        v-model="leadsByStage[stage.id]"
        :stage="stage"
        :is-collapsed="!!collapsedStages[stage.id]"
        :can-manage-stages="false"
        :can-edit-leads="allowEdit"
        :blurred="privacyMode"
        :matches-filter="matchesFilter"
        @toggle-collapse="toggleStageCollapse"
        @add-lead="openCreateLeadModal"
        @select-lead="openLeadDetails"
        @lead-drag-end="onDragEnd"
      />
    </div>

    <ClientOnly>
      <CrmCreateLeadModal
        v-if="allowEdit"
        v-model="showCreateLeadModal"
        :form="newLeadForm"
        :stages="stages"
        :loading="isCreatingManualLead"
        @submit="createManualLead"
      />

      <CrmActionModals
        v-model:show-delete-lead="unusedDeleteLeadModal"
        v-model:show-delete-note="showDeleteNoteModal"
        v-model:show-delete-stage="unusedDeleteStageModal"
        v-model:show-purchase="showPurchaseModal"
        v-model:purchase-value="purchaseValue"
        v-model:show-create-tag="showCreateTagModal"
        :is-deleting-stage="false"
        :is-confirming-purchase="isConfirmingPurchase"
        :is-creating-tag="isCreatingTag"
        :new-tag="newTag"
        @confirm-delete-note="doDeleteNote"
        @cancel-purchase="cancelPurchase"
        @confirm-purchase="confirmPurchase"
        @cancel-create-tag="cancelCreateTagModal"
        @confirm-create-tag="createTag"
      />

      <CrmLeadDetailsModal
        v-model="showLeadModal"
        v-model:selected-tag-to-add="selectedTagToAdd"
        v-model:new-note-content="newNoteContent"
        :lead-id="selectedLeadId"
        :lead-edit="leadEdit"
        :tracked-user="selectedLeadTrackedUser"
        :stages="stages"
        :all-tags="allTags"
        :is-saving-lead="isSavingLead"
        :is-adding-note="isAddingNote"
        :can-delete-leads="false"
        :read-only="!allowEdit"
        :public-token="token"
        @create-tag="openCreateTagModal"
        @add-tag="addTagToLead"
        @remove-tag="removeTag"
        @save-lead="saveLeadInfo"
        @add-note="addNote"
        @delete-note="confirmDeleteNote"
      />
    </ClientOnly>
  </div>
</template>

<script setup lang="ts">
import { Loader2, Eye, EyeOff, Plus } from 'lucide-vue-next'
import CrmMetricsStrip from '~/components/crm/CrmMetricsStrip.vue'
import CrmFilterBar from '~/components/crm/CrmFilterBar.vue'
import CrmStageColumn from '~/components/crm/CrmStageColumn.vue'
import CrmCreateLeadModal from '~/components/crm/CrmCreateLeadModal.vue'
import CrmActionModals from '~/components/crm/CrmActionModals.vue'
import CrmLeadDetailsModal from '~/components/crm/CrmLeadDetailsModal.vue'
import { useClientPortal } from '~/composables/useClientPortal'
import { usePublicCrmBoard } from '~/composables/usePublicCrmBoard'
import { usePublicCrmActions } from '~/composables/usePublicCrmActions'

const route = useRoute()
const token = route.params.token as string
const { formatTitle } = useBranding()
const { state, privacyMode } = useClientPortal(token)

useHead({ title: computed(() => formatTitle(state.value.pixelLabel ? `CRM • ${state.value.pixelLabel}` : 'CRM')) })

const unusedDeleteLeadModal = ref(false)
const unusedDeleteStageModal = ref(false)

const {
  isLoading, loadError, allowEdit, stages, leadsByStage, collapsedStages, allTags,
  searchQuery, selectedTagFilter, selectedOriginFilter, isFilterActive, clearFilters, matchesFilter,
  totalPipelineValue, totalLeadsCount, totalWonValue, averageTicket, filteredTotalCount,
  toggleStageCollapse, fetchBoard,
  showPurchaseModal, purchaseValue, isConfirmingPurchase, onDragEnd, cancelPurchase, confirmPurchase,
} = usePublicCrmBoard(token)

const {
  showCreateLeadModal, isCreatingManualLead, newLeadForm, openCreateLeadModal, createManualLead,
  showLeadModal, selectedLeadId, selectedLeadTrackedUser, leadEdit, openLeadDetails, saveLeadInfo,
  selectedTagToAdd, newNoteContent, isSavingLead, isAddingNote,
  addNote, showDeleteNoteModal, confirmDeleteNote, doDeleteNote,
  showCreateTagModal, isCreatingTag, newTag, openCreateTagModal, cancelCreateTagModal, createTag, addTagToLead, removeTag,
} = usePublicCrmActions(token, stages, allTags, allowEdit, fetchBoard)
</script>

<style scoped>
.public-crm-page { display: flex; flex-direction: column; gap: 1.25rem; height: calc(100vh - 4.5rem); }
.page-header { display: flex; justify-content: space-between; align-items: flex-end; gap: 1rem; flex-wrap: wrap; }
.title-wrap { display: flex; align-items: center; gap: 0.6rem; flex-wrap: wrap; }
.page-title { font-family: var(--font-sans); font-size: var(--fs-3xl); font-weight: var(--fw-bold); color: var(--text-strong); letter-spacing: -0.02em; margin: 0; }
.page-subtitle { color: var(--text-3); font-size: var(--fs-base); margin: 0.25rem 0 0; }
.mode-pill { padding: 0.18rem 0.55rem; border-radius: 999px; font-size: var(--fs-xs); font-weight: var(--fw-semibold); border: 1px solid var(--border-soft); }
.mode-pill.readonly { background: var(--surface-2); color: var(--text-2); }
.mode-pill.editable { background: var(--brand-soft); border-color: var(--brand-border); color: var(--brand); }
.header-actions { display: flex; align-items: center; gap: 0.5rem; flex-wrap: wrap; }
.kanban-board { display: flex; flex: 1; gap: 1.25rem; overflow-x: auto; padding-bottom: 0.75rem; align-items: stretch; }
.loading-state { display: flex; flex-direction: column; align-items: center; justify-content: center; flex: 1; color: var(--text-3); gap: 1rem; }
.error-banner, .empty-filter-banner {
  display: flex; align-items: center; justify-content: space-between; gap: 0.75rem;
  padding: 0.625rem 1rem; border-radius: 8px; background: var(--surface-2);
  border: 1px solid var(--border-soft); color: var(--text-2); font-size: var(--fs-base);
}
.error-banner { background: var(--danger-soft); border-color: var(--danger); color: var(--danger); }
.icon-brand { color: var(--brand); }
.spin { animation: spin 1s linear infinite; }
@keyframes spin { 100% { transform: rotate(360deg); } }
@media (max-width: 1023px) {
  .public-crm-page { height: auto; min-height: auto; gap: 1rem; overflow: visible; }
  .kanban-board {
    display: flex; gap: 0.875rem; overflow-x: auto; padding-bottom: 2rem;
    margin: 0 -1rem; padding-left: 1rem; padding-right: 1rem; scroll-snap-type: x mandatory;
  }
}
</style>
