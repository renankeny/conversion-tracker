<template>
  <BaseModal v-model="isOpen" :title="title" :description="description" size="md">
    <div class="picker-toolbar">
      <div v-if="hasCategories" class="cat-tabs">
        <button
          v-for="tab in categoryTabs"
          :key="tab.value"
          type="button"
          class="cat-tab"
          :class="{ active: activeTab === tab.value }"
          @click="activeTab = tab.value"
        >
          <MetaAdsIcon v-if="tab.value === 'meta'" :size="11" />
          <GoogleAdsIcon v-else-if="tab.value === 'google'" :size="11" />
          <span>{{ tab.label }}</span>
        </button>
      </div>

      <div class="toolbar-meta">
        <span class="drag-hint">
          <GripVertical :size="12" />
          <span>Arraste para reordenar</span>
        </span>
        <button
          v-if="localSelected.length > 0 && activeTab === 'all'"
          type="button"
          class="group-selected-btn"
          title="Agrupar os itens marcados no topo da lista"
          @click="moveSelectedToTop"
        >
          <ArrowUpToLine :size="12" />
          <span>Ativos no topo</span>
        </button>
      </div>
    </div>

    <draggable
      v-model="draggableOrder"
      class="event-picker-list"
      :item-key="(id: string) => id"
      handle=".drag-handle"
      :animation="180"
      ghost-class="picker-ghost"
      chosen-class="picker-chosen"
      :fallback-tolerance="4"
    >
      <template #item="{ element: id }">
        <div class="event-picker-item" :class="{ 'is-selected': selectedSet.has(id) }">
          <button
            type="button"
            class="drag-handle"
            title="Arrastar para reordenar"
            aria-label="Arrastar para reordenar"
          >
            <GripVertical :size="15" />
          </button>

          <span v-if="selectedPositionOf(id) > 0" class="order-badge">
            {{ selectedPositionOf(id) }}
          </span>

          <div class="event-picker-check">
            <BaseCheckbox
              v-model="localSelected"
              :value="id"
              :label="labelOf(id)"
            />
            <span v-if="categoryBadgeOf(id)" class="cat-pill" :class="`cat-${categoryOf(id)}`">
              <MetaAdsIcon v-if="categoryOf(id) === 'meta'" :size="10" />
              <GoogleAdsIcon v-else-if="categoryOf(id) === 'google'" :size="10" />
              <span>{{ categoryBadgeOf(id) }}</span>
            </span>
          </div>

          <input
            v-if="editableLabels"
            v-model="localLabels[id]"
            type="text"
            class="event-picker-label-input"
            :placeholder="labelOf(id)"
          >
        </div>
      </template>
    </draggable>

    <template #footer>
      <span class="footer-count">{{ localSelected.length }} selecionado(s)</span>
      <div class="footer-actions">
        <BaseButton variant="ghost" size="sm" @click="isOpen = false">Cancelar</BaseButton>
        <BaseButton variant="primary" size="sm" :loading="saving" @click="onSave">Salvar</BaseButton>
      </div>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import { GripVertical, ArrowUpToLine } from 'lucide-vue-next'
import draggable from 'vuedraggable'
import { MetaAdsIcon, GoogleAdsIcon } from '~/utils/adIcons'
import BaseModal from '~/components/BaseModal.vue'
import BaseButton from '~/components/BaseButton.vue'
import type { CardCategory } from '~/utils/dashboardEventCatalog'

const props = withDefaults(defineProps<{
  modelValue: boolean
  title?: string
  description?: string
  events: Array<{ id: string; label: string; category?: CardCategory }>
  selected: string[]
  order?: string[]
  saving?: boolean
  editableLabels?: boolean
  labelOverrides?: Record<string, string>
}>(), {
  title: 'Escolher eventos',
  description: '',
  saving: false,
  editableLabels: false,
  labelOverrides: () => ({}),
  order: undefined,
})

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  save: [payload: { ids: string[]; order: string[]; labels: Record<string, string> }]
}>()

const isOpen = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v),
})

const activeTab = ref<'all' | CardCategory>('all')
const hasCategories = computed(() => props.events.some((e) => Boolean(e.category)))
const categoryTabs: Array<{ value: 'all' | CardCategory; label: string }> = [
  { value: 'all', label: 'Todos' },
  { value: 'event', label: 'Eventos' },
  { value: 'meta', label: 'Meta Ads' },
  { value: 'google', label: 'Google Ads' },
  { value: 'general', label: 'Geral & ROAS' },
]

const optionById = computed(() => new Map(props.events.map((e) => [e.id, e])))
const labelOf = (id: string) => optionById.value.get(id)?.label || id
const categoryOf = (id: string) => optionById.value.get(id)?.category || 'event'
const categoryBadgeOf = (id: string) => {
  const cat = optionById.value.get(id)?.category
  if (cat === 'meta') return 'Meta Ads'
  if (cat === 'google') return 'Google Ads'
  if (cat === 'general') return 'Geral'
  return ''
}

const buildOrder = (): string[] => {
  const catalogIds = props.events.map((e) => e.id)
  const base = props.order && props.order.length ? props.order : catalogIds
  const merged = [...base.filter((id) => catalogIds.includes(id))]
  for (const id of catalogIds) {
    if (!merged.includes(id)) merged.push(id)
  }
  return merged
}

const localSelected = ref<string[]>([...props.selected])
const localLabels = ref<Record<string, string>>({ ...props.labelOverrides })
const localOrder = ref<string[]>(buildOrder())

const selectedSet = computed(() => new Set(localSelected.value))
const selectedPositions = computed(() => {
  const map = new Map<string, number>()
  let pos = 1
  for (const id of localOrder.value) {
    if (selectedSet.value.has(id)) map.set(id, pos++)
  }
  return map
})
const selectedPositionOf = (id: string) => selectedPositions.value.get(id) ?? 0

const draggableOrder = computed<string[]>({
  get: () => {
    if (!hasCategories.value || activeTab.value === 'all') return localOrder.value
    return localOrder.value.filter((id) => categoryOf(id) === activeTab.value)
  },
  set: (reordered) => {
    if (!hasCategories.value || activeTab.value === 'all') {
      localOrder.value = reordered
      return
    }
    const cat = activeTab.value
    let ptr = 0
    localOrder.value = localOrder.value.map((id) => (categoryOf(id) === cat ? (reordered[ptr++] ?? id) : id))
  },
})

const moveSelectedToTop = () => {
  const active = localOrder.value.filter((id) => selectedSet.value.has(id))
  const inactive = localOrder.value.filter((id) => !selectedSet.value.has(id))
  localOrder.value = [...active, ...inactive]
}

watch(() => props.modelValue, (open) => {
  if (!open) return
  activeTab.value = 'all'
  localSelected.value = [...props.selected]
  localLabels.value = { ...props.labelOverrides }
  localOrder.value = buildOrder()
})

const onSave = () => {
  emit('save', { ids: localSelected.value, order: localOrder.value, labels: localLabels.value })
}
</script>

<style scoped>
.picker-toolbar { display: flex; flex-direction: column; gap: 0.6rem; margin-bottom: 0.75rem; padding-bottom: 0.625rem; border-bottom: 1px solid var(--border-soft); }
.cat-tabs { display: flex; gap: 0.375rem; flex-wrap: wrap; }
.cat-tab { display: inline-flex; align-items: center; gap: 0.3rem; padding: 0.35rem 0.7rem; border-radius: 8px; border: 1px solid var(--border-soft); background: var(--surface-deep); color: var(--text-2); font-size: var(--fs-xs); font-weight: var(--fw-semibold); cursor: pointer; transition: all 0.15s ease; }
.cat-tab:hover { border-color: var(--border); color: var(--text-1); }
.cat-tab.active { background: var(--brand-soft); border-color: var(--brand-border); color: var(--brand); }
.toolbar-meta { display: flex; align-items: center; justify-content: space-between; gap: 0.5rem; }
.drag-hint { display: inline-flex; align-items: center; gap: 0.3rem; font-size: var(--fs-xs); color: var(--text-3); }
.group-selected-btn { display: inline-flex; align-items: center; gap: 0.3rem; padding: 0.22rem 0.55rem; border-radius: 6px; border: 1px solid var(--border-soft); background: var(--surface-deep); color: var(--text-2); font-size: var(--fs-xs); font-weight: var(--fw-medium); cursor: pointer; transition: all 0.15s ease; }
.group-selected-btn:hover { border-color: var(--brand-border); color: var(--brand); background: var(--brand-soft); }
.event-picker-list { display: flex; flex-direction: column; gap: 0.45rem; max-height: 55vh; overflow-y: auto; padding-right: 0.25rem; }
.event-picker-item { display: flex; align-items: center; gap: 0.625rem; padding: 0.5rem 0.7rem; border-radius: 8px; border: 1px solid var(--border-soft); background: var(--surface-soft-2); transition: background 0.15s, border-color 0.15s; }
.event-picker-item:hover { background: var(--surface-soft); border-color: var(--border); }
.event-picker-item.is-selected { border-color: var(--brand-border); background: var(--surface-soft); }
.drag-handle { background: transparent; border: none; color: var(--text-3); cursor: grab; padding: 0.2rem; display: inline-flex; align-items: center; justify-content: center; border-radius: 5px; flex-shrink: 0; touch-action: none; }
.drag-handle:hover { color: var(--text-1); background: var(--surface-soft); }
.drag-handle:active { cursor: grabbing; color: var(--brand); }
.order-badge { min-width: 1.25rem; height: 1.25rem; padding: 0 0.3rem; border-radius: 999px; background: var(--brand-soft); border: 1px solid var(--brand-border); color: var(--brand); font-size: var(--fs-2xs); font-weight: var(--fw-semibold); display: inline-flex; align-items: center; justify-content: center; flex-shrink: 0; }
.event-picker-check { display: flex; align-items: center; gap: 0.5rem; flex: 1; min-width: 0; flex-wrap: wrap; }
.cat-pill { display: inline-flex; align-items: center; gap: 0.25rem; font-size: var(--fs-2xs); font-weight: var(--fw-semibold); padding: 0.12rem 0.45rem; border-radius: 999px; border: 1px solid var(--border-soft); background: var(--surface-deep); color: var(--text-3); }
.cat-meta, .cat-google, .cat-general { color: var(--brand); border-color: var(--brand-border); }
.event-picker-label-input { width: 40%; background: var(--surface-2); border: 1px solid var(--border); border-radius: 6px; padding: 0.35rem 0.5rem; font-size: var(--fs-sm); color: var(--text-1); outline: none; }
.event-picker-label-input:focus { border-color: var(--brand); }
:deep(.picker-ghost) { opacity: 0.35; background: var(--brand-soft); border: 1px dashed var(--brand); }
:deep(.picker-chosen) { border-color: var(--brand); box-shadow: var(--shadow-card); }
.footer-count { margin-right: auto; font-size: var(--fs-xs); color: var(--text-3); }
.footer-actions { display: flex; align-items: center; gap: 0.5rem; }
@media (max-width: 480px) {
  .event-picker-item { flex-wrap: wrap; }
  .event-picker-label-input { width: 100%; margin-left: 1.75rem; }
}
</style>
