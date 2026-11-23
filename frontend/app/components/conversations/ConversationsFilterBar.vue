<template>
  <div class="filter-panel">
    <div class="filter-row-top">
      <div class="search-box">
        <Search :size="15" class="search-icon" />
        <input
          :value="search"
          type="text"
          class="search-input"
          placeholder="Buscar por telefone, nome ou campanha..."
          @input="$emit('update:search', ($event.target as HTMLInputElement).value)"
        >
        <button v-if="search" type="button" class="clear-search" @click="$emit('update:search', '')">
          <X :size="13" />
        </button>
      </div>

      <div class="dropdowns-wrap">
        <PixelDropdown
          v-if="!publicMode"
          :model-value="selectedPixel"
          allow-all
          @update:model-value="$emit('update:selectedPixel', $event)"
        />
        <BaseDropdown
          v-if="!publicMode && instances.length > 1"
          :model-value="selectedInstanceId"
          :options="instanceOptions"
          @update:model-value="$emit('update:selectedInstanceId', String($event ?? ''))"
        />
        <BaseDropdown
          :model-value="slaStatus"
          :options="slaOptions"
          @update:model-value="$emit('update:slaStatus', String($event ?? 'all'))"
        />
        <BaseDropdown
          v-if="stages.length > 0"
          :model-value="stageId"
          :options="stageOptions"
          @update:model-value="$emit('update:stageId', String($event ?? ''))"
        />
        <BaseDropdown
          v-if="tags.length > 0"
          :model-value="tagId"
          :options="tagOptions"
          @update:model-value="$emit('update:tagId', String($event ?? ''))"
        />
        <PeriodSelector
          :model-value="period"
          :custom-from="customFrom"
          :custom-to="customTo"
          @update:model-value="$emit('update:period', $event as PeriodValue)"
          @update:custom-from="$emit('update:customFrom', $event)"
          @update:custom-to="$emit('update:customTo', $event)"
        />
      </div>
    </div>

    <div class="filter-row-bottom">
      <div class="origin-strip">
        <span class="origin-label">De onde veio:</span>
        <button
          v-for="pill in visibleOriginPills"
          :key="pill.value"
          type="button"
          class="origin-pill"
          :class="{ active: origin === pill.value }"
          @click="$emit('update:origin', pill.value)"
        >
          <span>{{ pill.label }}</span>
          <span class="pill-count">{{ pill.count }}</span>
        </button>

        <BaseDropdown
          v-if="campaigns.length > 0"
          :model-value="campaign"
          size="sm"
          :options="campaignOptions"
          @update:model-value="$emit('update:campaign', String($event ?? ''))"
        />
        <BaseDropdown
          v-if="utms.length > 0"
          :model-value="utm"
          size="sm"
          :options="utmOptions"
          @update:model-value="$emit('update:utm', String($event ?? ''))"
        />
      </div>

      <div class="bottom-actions">
        <BaseButton v-if="hasActiveFilters" variant="ghost" size="sm" @click="$emit('clear')">
          Limpar filtros
        </BaseButton>
        <BaseButton variant="outline" size="sm" :loading="exporting" :disabled="total === 0" @click="$emit('export')">
          <Download :size="14" />
          <span>Exportar {{ total }} {{ total === 1 ? 'conversa' : 'conversas' }} (CSV)</span>
        </BaseButton>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Search, X, Download } from 'lucide-vue-next'
import PeriodSelector from '~/components/dashboard/PeriodSelector.vue'
import type {
  ConversationOriginCategory,
  ConversationOriginCounts,
  CrmStage,
  CrmTag,
  WhatsAppInstance,
} from '#shared/types'
import type { PeriodValue } from '~/stores/useDashboardStore'

const props = defineProps<{
  publicMode?: boolean
  search: string
  period: PeriodValue
  customFrom: string
  customTo: string
  selectedPixel: string
  selectedInstanceId: string
  slaStatus: string
  stageId: string
  tagId: string
  origin: ConversationOriginCategory
  campaign: string
  utm: string
  total: number
  exporting: boolean
  hasActiveFilters: boolean
  originCounts: ConversationOriginCounts
  campaigns: string[]
  utms: string[]
  stages: CrmStage[]
  tags: CrmTag[]
  instances: WhatsAppInstance[]
}>()

defineEmits<{
  'update:search': [v: string]
  'update:period': [v: PeriodValue]
  'update:customFrom': [v: string]
  'update:customTo': [v: string]
  'update:selectedPixel': [v: string]
  'update:selectedInstanceId': [v: string]
  'update:slaStatus': [v: string]
  'update:stageId': [v: string]
  'update:tagId': [v: string]
  'update:origin': [v: ConversationOriginCategory]
  'update:campaign': [v: string]
  'update:utm': [v: string]
  clear: []
  export: []
}>()

const slaOptions = [
  { value: 'all', label: 'Todos os atendimentos' },
  { value: 'pending', label: 'Pendentes de resposta' },
  { value: 'unanswered', label: 'Sem 1ª resposta' },
  { value: 'waiting_reply', label: 'Aguardando retorno' },
  { value: 'replied', label: 'Respondidos' },
]

const instanceOptions = computed(() => [
  { value: '', label: 'Todos os números' },
  ...props.instances.map((i) => ({ value: String(i.id), label: i.name || i.instance_name })),
])

const stageOptions = computed(() => [
  { value: '', label: 'Todas as etapas' },
  ...props.stages.map((s) => ({ value: String(s.id), label: s.name })),
])

const tagOptions = computed(() => [
  { value: '', label: 'Todas as tags' },
  ...props.tags.map((t) => ({ value: String(t.id), label: t.name })),
])

const campaignOptions = computed(() => [
  { value: '', label: 'Campanha' },
  ...props.campaigns.map((c) => ({ value: c, label: c })),
])

const utmOptions = computed(() => [
  { value: '', label: 'UTM Source' },
  ...props.utms.map((u) => ({ value: u, label: u })),
])

const visibleOriginPills = computed(() => {
  const c = props.originCounts
  const list: Array<{ value: ConversationOriginCategory; label: string; count: number }> = [
    { value: 'all', label: 'Todas', count: c.all },
    { value: 'meta_ads', label: 'Meta Ads', count: c.meta_ads },
    { value: 'google_ads', label: 'Google Ads', count: c.google_ads },
  ]
  if (c.tiktok_ads > 0 || props.origin === 'tiktok_ads') {
    list.push({ value: 'tiktok_ads', label: 'TikTok Ads', count: c.tiktok_ads })
  }
  list.push(
    { value: 'bio_link', label: 'Link na Bio', count: c.bio_link },
    { value: 'site', label: 'Site', count: c.site },
  )
  return list
})
</script>

<style scoped>
.filter-panel {
  background: var(--surface-2); border: 1px solid var(--border-soft);
  border-radius: 12px; padding: 0.875rem 1rem; display: flex; flex-direction: column; gap: 0.75rem;
}
.filter-row-top { display: flex; align-items: center; gap: 0.625rem; flex-wrap: wrap; }
.search-box {
  flex: 1 1 240px; min-width: 200px; position: relative; display: flex; align-items: center;
  background: var(--surface-deep); border: 1px solid var(--border); border-radius: 8px; padding: 0 0.65rem; height: 36px;
}
.search-box:focus-within { border-color: var(--brand); }
.search-icon { color: var(--text-3); flex-shrink: 0; margin-right: 0.5rem; }
.search-input {
  flex: 1; background: transparent; border: none; outline: none;
  color: var(--text-1); font-size: var(--fs-base); min-width: 0;
}
.search-input::placeholder { color: var(--text-3); }
.clear-search { background: none; border: none; color: var(--text-3); cursor: pointer; display: flex; padding: 2px; }
.dropdowns-wrap { display: flex; align-items: center; gap: 0.5rem; flex-wrap: wrap; }
.filter-row-bottom {
  display: flex; align-items: center; justify-content: space-between; gap: 0.75rem;
  flex-wrap: wrap; padding-top: 0.625rem; border-top: 1px solid var(--border-soft);
}
.origin-strip { display: flex; align-items: center; gap: 0.4rem; flex-wrap: wrap; }
.origin-label { font-size: var(--fs-xs); font-weight: var(--fw-semibold); color: var(--text-3); margin-right: 0.25rem; }
.origin-pill {
  display: inline-flex; align-items: center; gap: 0.35rem; padding: 0.3rem 0.65rem;
  border-radius: 999px; border: 1px solid var(--border); background: var(--surface-deep);
  color: var(--text-2); font-size: var(--fs-xs); font-weight: var(--fw-semibold); cursor: pointer; transition: all 0.15s ease;
}
.origin-pill:hover { border-color: var(--border-hover); color: var(--text-1); }
.origin-pill.active { background: var(--brand); border-color: var(--brand); color: var(--brand-text); }
.pill-count { opacity: 0.85; font-family: var(--font-mono); font-size: var(--fs-2xs); }
.bottom-actions { display: flex; align-items: center; gap: 0.5rem; flex-wrap: wrap; }
</style>
