<template>
  <div class="conversations-page">
    <div class="page-header">
      <div>
        <h1 class="page-title">Conversas</h1>
        <p class="page-subtitle">Acompanhe as conversas rastreadas do WhatsApp, suas atribuições e o SLA comercial</p>
      </div>

      <div class="header-actions">
        <BaseButton
          variant="ghost"
          size="sm"
          :title="dashboardStore.privacyMode ? 'Mostrar valores financeiros' : 'Ocultar valores financeiros'"
          @click="dashboardStore.togglePrivacyMode()"
        >
          <EyeOff v-if="dashboardStore.privacyMode" :size="15" />
          <Eye v-else :size="15" />
        </BaseButton>
        <BaseButton variant="outline" size="sm" :loading="loading" @click="fetchConversations(false)">
          Atualizar
        </BaseButton>
      </div>
    </div>

    <div v-if="loadError" class="error-banner">
      <span>{{ loadError }}</span>
      <BaseButton variant="primary" size="sm" @click="fetchConversations(false)">Tentar novamente</BaseButton>
    </div>

    <ConversationsKpiStrip
      :kpis="kpis"
      :blurred="dashboardStore.privacyMode"
    />

    <ConversationsFilterBar
      v-model:search="search"
      v-model:period="period"
      v-model:custom-from="customFrom"
      v-model:custom-to="customTo"
      v-model:selected-pixel="selectedPixel"
      v-model:selected-instance-id="selectedInstanceId"
      v-model:sla-status="slaStatus"
      v-model:stage-id="stageId"
      v-model:tag-id="tagId"
      v-model:origin="origin"
      v-model:campaign="campaign"
      v-model:utm="utm"
      :total="total"
      :exporting="exporting"
      :has-active-filters="hasActiveFilters"
      :origin-counts="originCounts"
      :campaigns="campaigns"
      :utms="utms"
      :stages="stages"
      :tags="tags"
      :instances="instances"
      @clear="clearFilters"
      @export="exportCSV"
    />

    <ConversationsTable
      v-model:page="page"
      :items="items"
      :total="total"
      :loading="loading"
      :sort-by="sortBy"
      :sort-dir="sortDir"
      :page-size="pageSize"
      :blurred="dashboardStore.privacyMode"
      @sort="toggleSort"
      @inspect="inspectConversation"
    />

    <ConversationInspectDrawer
      v-model="drawerOpen"
      :item="selectedItem"
    />
  </div>
</template>

<script setup lang="ts">
import { Eye, EyeOff } from 'lucide-vue-next'
import ConversationsKpiStrip from '~/components/conversations/ConversationsKpiStrip.vue'
import ConversationsFilterBar from '~/components/conversations/ConversationsFilterBar.vue'
import ConversationsTable from '~/components/conversations/ConversationsTable.vue'
import ConversationInspectDrawer from '~/components/conversations/ConversationInspectDrawer.vue'
import { useConversationsAudit } from '~/composables/useConversationsAudit'
import { useDashboardStore } from '~/stores/useDashboardStore'

definePageMeta({ layout: 'app', middleware: 'auth' })

const { formatTitle } = useBranding()
useHead({ title: computed(() => formatTitle('Conversas')) })

const dashboardStore = useDashboardStore()
const tenantStore = useTenantStore()

const {
  loading, exporting, loadError,
  period, customFrom, customTo, selectedPixel, selectedInstanceId,
  search, slaStatus, stageId, tagId, origin, campaign, utm,
  sortBy, sortDir, page, pageSize, hasActiveFilters,
  items, total, kpis, originCounts, campaigns, utms, stages, tags, instances,
  selectedItem, drawerOpen,
  fetchConversations, loadMetaFilters, toggleSort, clearFilters, exportCSV, inspectConversation,
} = useConversationsAudit()

watch(() => tenantStore.switchVersion, () => {
  drawerOpen.value = false
  selectedItem.value = null
  selectedPixel.value = ''
  selectedInstanceId.value = ''
  page.value = 1
  void loadMetaFilters()
  void fetchConversations(false)
})
</script>

<style scoped>
.conversations-page { display: flex; flex-direction: column; gap: 1.25rem; }
.page-header { display: flex; justify-content: space-between; align-items: flex-end; gap: 1rem; flex-wrap: wrap; }
.page-title { font-family: var(--font-sans); font-size: var(--fs-3xl); font-weight: var(--fw-bold); color: var(--text-strong); letter-spacing: -0.02em; margin: 0; }
.page-subtitle { color: var(--text-3); font-size: var(--fs-base); margin: 0.25rem 0 0; }
.header-actions { display: flex; align-items: center; gap: 0.5rem; }
.error-banner {
  display: flex; align-items: center; justify-content: space-between; gap: 1rem;
  padding: 0.75rem 1rem; border-radius: 10px; background: var(--danger-soft);
  border: 1px solid var(--danger); color: var(--danger); font-size: var(--fs-md); flex-wrap: wrap;
}
</style>
