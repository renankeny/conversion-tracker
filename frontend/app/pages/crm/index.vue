<template>
  <div class="crm-page">
    <CrmHeaderBar
      v-model:selected-pixel="selectedPixel"
      :is-loading="isLoading"
      :can-manage-stages="canManageStages"
      @create-lead="openCreateLeadModal()"
      @create-stage="openNewStageModal(false)"
      @open-pipeline="openPipelineSettings"
      @refresh="fetchBoard(false)"
    />

    <CrmMetricsStrip
      :total-pipeline-value="totalPipelineValue"
      :total-leads-count="isFilterActive ? filteredTotalCount : totalLeadsCount"
      :total-won-value="totalWonValue"
      :average-ticket="averageTicket"
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
      <Loader2 :size="36" class="spin icon-brand" />
      <p>Carregando funil de vendas...</p>
    </div>

    <draggable
      v-else
      v-model="stages"
      class="kanban-board"
      group="stages"
      item-key="id"
      :animation="200"
      :disabled="!canManageStages"
      handle=".column-drag-handle"
      @end="onStageDragEnd"
    >
      <template #item="{ element: stage }">
        <CrmStageColumn
          v-model="leadsByStage[stage.id]"
          :stage="stage"
          :is-collapsed="!!collapsedStages[stage.id]"
          :can-manage-stages="canManageStages"
          :matches-filter="matchesFilter"
          @toggle-collapse="toggleStageCollapse"
          @add-lead="openCreateLeadModal"
          @config-stage="openStageConfig($event, false)"
          @select-lead="openLeadDetails"
          @lead-drag-end="onDragEnd"
        />
      </template>
    </draggable>

    <ClientOnly>
      <CrmCreateLeadModal
        v-model="showCreateLeadModal"
        :form="newLeadForm"
        :stages="stages"
        :loading="isCreatingManualLead"
        @submit="createManualLead"
      />

      <CrmActionModals
        v-model:show-delete-lead="showDeleteLeadModal"
        v-model:show-delete-note="showDeleteNoteModal"
        v-model:show-delete-stage="showDeleteStageModal"
        v-model:show-purchase="showPurchaseModal"
        v-model:purchase-value="purchaseValue"
        v-model:show-create-tag="showCreateTagModal"
        :is-deleting-stage="isSavingStage"
        :is-confirming-purchase="isConfirmingPurchase"
        :is-creating-tag="isCreatingTag"
        :new-tag="newTag"
        @confirm-delete-lead="doDeleteLead"
        @cancel-delete-lead="cancelDeleteLead"
        @confirm-delete-note="doDeleteNote"
        @confirm-delete-stage="deleteStage"
        @cancel-purchase="cancelPurchase"
        @confirm-purchase="confirmPurchase"
        @cancel-create-tag="cancelCreateTagModal"
        @confirm-create-tag="createTag"
      />

      <CrmStageModal
        v-model="showStageModal"
        :is-creating="isCreatingStage"
        :loading="isSavingStage"
        :is-loading-gads-actions="isLoadingGadsActions"
        :can-delete-stage="stages.length > 1"
        :stage="selectedStage"
        :gads-accounts="gadsAccounts"
        :gads-actions="gadsActions"
        @change-gads-account="onStageGoogleAdsAccountChange"
        @close="closeStageModal"
        @save="saveStage"
        @delete="askDeleteStage"
      />

      <CrmPipelineModal
        v-model="showPipelineModal"
        :stages="stages"
        :get-leads-for-stage="getLeadsForStage"
        @move-up="moveStageUp"
        @move-down="moveStageDown"
        @edit-stage="openStageConfig($event, true)"
        @new-stage="openNewStageModal(true)"
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
        :can-delete-leads="canDeleteLeads"
        @open-journey="openTrackedUserJourney"
        @create-tag="openCreateTagModal"
        @add-tag="addTagToLead"
        @remove-tag="removeTag"
        @delete-account-tag="deleteTagFromAccount"
        @delete-lead="openConfirmDeleteLead"
        @save-lead="saveLeadInfo"
        @add-note="addNote"
        @delete-note="confirmDeleteNote"
      />

      <UserDetailsModal v-model="showTrackedUserModal" :user="selectedTrackedUser" />
    </ClientOnly>
  </div>
</template>

<script setup lang="ts">
import { onMounted, computed } from 'vue'
import { Loader2 } from 'lucide-vue-next'
import draggable from 'vuedraggable'
import CrmHeaderBar from '~/components/crm/CrmHeaderBar.vue'
import CrmMetricsStrip from '~/components/crm/CrmMetricsStrip.vue'
import CrmFilterBar from '~/components/crm/CrmFilterBar.vue'
import CrmStageColumn from '~/components/crm/CrmStageColumn.vue'
import CrmCreateLeadModal from '~/components/crm/CrmCreateLeadModal.vue'
import CrmActionModals from '~/components/crm/CrmActionModals.vue'
import CrmStageModal from '~/components/crm/CrmStageModal.vue'
import CrmPipelineModal from '~/components/crm/CrmPipelineModal.vue'
import CrmLeadDetailsModal from '~/components/crm/CrmLeadDetailsModal.vue'
import UserDetailsModal from '~/components/users/UserDetailsModal.vue'
import { useCrmBoard } from '~/composables/useCrmBoard'
import { useCrmModals } from '~/composables/useCrmModals'
import { useCrmLeadActions } from '~/composables/useCrmLeadActions'
import { useCrmTagActions } from '~/composables/useCrmTagActions'

definePageMeta({ layout: 'app', middleware: 'auth' })

const { formatTitle } = useBranding()
useHead({ title: computed(() => formatTitle('CRM Kanban')) })

const {
  isLoading, stages, leadsByStage, selectedPixel, collapsedStages, allTags, gadsAccounts,
  canManageStages, canDeleteLeads,
  searchQuery, selectedTagFilter, selectedOriginFilter, isFilterActive, clearFilters, matchesFilter,
  totalPipelineValue, totalLeadsCount, totalWonValue, averageTicket, filteredTotalCount,
  toggleStageCollapse, getLeadsForStage, fetchBoard, resetPixelAndFetchBoard, loadGadsAccounts, loadAllTags,
  moveStageUp, moveStageDown, onStageDragEnd,
  showPurchaseModal, purchaseValue, isConfirmingPurchase, onDragEnd, cancelPurchase, confirmPurchase,
} = useCrmBoard()

const modals = useCrmModals({ stages, allTags, gadsAccounts, selectedPixel, fetchBoard, loadAllTags })
const {
  showPipelineModal, showStageModal, isCreatingStage, isSavingStage, isLoadingGadsActions,
  selectedStage, gadsActions, onStageGoogleAdsAccountChange,
  openPipelineSettings, openNewStageModal, openStageConfig, closeStageModal, saveStage, askDeleteStage, deleteStage,
  showCreateLeadModal, isCreatingManualLead, newLeadForm,
  showLeadModal, selectedLeadId, selectedLeadTrackedUser, showTrackedUserModal, selectedTrackedUser,
  leadEdit, showCreateTagModal, isCreatingTag, newTag, selectedTagToAdd, newNoteContent,
  isSavingLead, isAddingNote, showDeleteLeadModal, showDeleteNoteModal, showDeleteStageModal,
} = modals

const {
  openCreateLeadModal, createManualLead, openLeadDetails, openTrackedUserJourney, saveLeadInfo,
  addNote, confirmDeleteNote, doDeleteNote, openConfirmDeleteLead, cancelDeleteLead, doDeleteLead,
} = useCrmLeadActions(modals, stages, allTags, selectedPixel, fetchBoard, loadAllTags)

const {
  openCreateTagModal, cancelCreateTagModal, createTag, addTagToLead, removeTag, deleteTagFromAccount,
} = useCrmTagActions(modals, allTags, fetchBoard)

const tenantStore = useTenantStore()
watch(() => tenantStore.switchVersion, () => {
  void resetPixelAndFetchBoard()
  void loadGadsAccounts()
  void loadAllTags()
})

onMounted(() => {
  void fetchBoard(false)
  void loadGadsAccounts()
  void loadAllTags()
})
</script>

<style scoped>
.crm-page { display: flex; flex-direction: column; gap: 1.25rem; height: calc(100vh - 4.5rem); }
.kanban-board { display: flex; flex: 1; gap: 1.25rem; overflow-x: auto; padding-bottom: 0.75rem; align-items: stretch; }
.loading-state { display: flex; flex-direction: column; align-items: center; justify-content: center; flex: 1; color: var(--text-3); gap: 1rem; }
.empty-filter-banner {
  display: flex; align-items: center; justify-content: space-between; gap: 0.75rem;
  padding: 0.625rem 1rem; border-radius: 8px;
  background: var(--surface-2); border: 1px solid var(--border-soft);
  color: var(--text-2); font-size: var(--fs-base);
}
.icon-brand { color: var(--brand); }
.spin { animation: spin 1s linear infinite; }
@keyframes spin { 100% { transform: rotate(360deg); } }

@media (max-width: 1023px) {
  .crm-page { height: auto; min-height: auto; gap: 1rem; overflow: visible; }
  .kanban-board {
    display: flex; gap: 0.875rem; overflow-x: auto; padding-bottom: 2rem;
    margin: 0 -1rem; padding-left: 1rem; padding-right: 1rem;
    scroll-padding: 1rem; scroll-snap-type: x mandatory; -webkit-overflow-scrolling: touch;
  }
}
</style>
