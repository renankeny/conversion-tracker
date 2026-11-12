<template>
  <BaseModal v-model="isOpen" :title="title" :description="description" size="md">
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

    <div class="event-picker-list">
      <div v-for="id in filteredOrder" :key="id" class="event-picker-item">
        <div class="event-picker-reorder">
          <button
            type="button"
            class="reorder-btn"
            :disabled="indexOfId(id) === 0"
            title="Mover para cima"
            @click="moveById(id, -1)"
          >
            <ChevronUp :size="14" />
          </button>
          <button
            type="button"
            class="reorder-btn"
            :disabled="indexOfId(id) === localOrder.length - 1"
            title="Mover para baixo"
            @click="moveById(id, 1)"
          >
            <ChevronDown :size="14" />
          </button>
        </div>

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
    </div>

    <template #footer>
      <BaseButton variant="ghost" size="sm" @click="isOpen = false">Cancelar</BaseButton>
      <BaseButton variant="primary" size="sm" :loading="saving" @click="onSave">Salvar</BaseButton>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import { ChevronUp, ChevronDown } from 'lucide-vue-next'
import { MetaAdsIcon, GoogleAdsIcon } from '~/utils/adIcons'
import BaseModal from '~/components/BaseModal.vue'
import BaseButton from '~/components/BaseButton.vue'
import type { CardCategory } from '~/utils/dashboardEventCatalog'

interface EventOption {
  id: string
  label: string
  category?: CardCategory
}

interface Props {
  modelValue: boolean
  title?: string
  description?: string
  events: EventOption[]
  selected: string[]
  order?: string[]
  saving?: boolean
  editableLabels?: boolean
  labelOverrides?: Record<string, string>
}

const props = withDefaults(defineProps<Props>(), {
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

const filteredOrder = computed(() => {
  if (!hasCategories.value || activeTab.value === 'all') return localOrder.value
  return localOrder.value.filter((id) => categoryOf(id) === activeTab.value)
})

watch(() => props.modelValue, (open) => {
  if (!open) return
  activeTab.value = 'all'
  localSelected.value = [...props.selected]
  localLabels.value = { ...props.labelOverrides }
  localOrder.value = buildOrder()
})

const indexOfId = (id: string) => localOrder.value.indexOf(id)

const moveById = (id: string, dir: -1 | 1) => {
  const list = filteredOrder.value
  const posInFiltered = list.indexOf(id)
  const targetFilteredPos = posInFiltered + dir
  if (posInFiltered < 0 || targetFilteredPos < 0 || targetFilteredPos >= list.length) return
  const targetId = list[targetFilteredPos]
  if (!targetId) return
  const idxA = localOrder.value.indexOf(id)
  const idxB = localOrder.value.indexOf(targetId)
  if (idxA < 0 || idxB < 0) return
  const arr = [...localOrder.value]
  arr[idxA] = targetId
  arr[idxB] = id
  localOrder.value = arr
}

const onSave = () => {
  emit('save', { ids: localSelected.value, order: localOrder.value, labels: localLabels.value })
}
</script>

<style scoped>
.cat-tabs { display: flex; gap: 0.375rem; flex-wrap: wrap; margin-bottom: 0.875rem; padding-bottom: 0.625rem; border-bottom: 1px solid var(--border-soft); }
.cat-tab { display: inline-flex; align-items: center; gap: 0.3rem; padding: 0.35rem 0.7rem; border-radius: 8px; border: 1px solid var(--border-soft); background: var(--surface-deep); color: var(--text-2); font-size: var(--fs-xs); font-weight: var(--fw-semibold); cursor: pointer; transition: all 0.15s ease; }
.cat-tab:hover { border-color: var(--border); color: var(--text-1); }
.cat-tab.active { background: var(--brand-soft); border-color: var(--brand-border); color: var(--brand); }
.event-picker-list { display: flex; flex-direction: column; gap: 0.5rem; max-height: 58vh; overflow-y: auto; padding-right: 0.25rem; }
.event-picker-item { display: flex; align-items: center; gap: 0.75rem; padding: 0.55rem 0.75rem; border-radius: 8px; border: 1px solid var(--border-soft); background: var(--surface-soft-2); transition: all 0.15s; }
.event-picker-item:hover { background: var(--surface-soft); border-color: var(--border); }
.event-picker-reorder { display: flex; flex-direction: column; flex-shrink: 0; }
.reorder-btn { background: transparent; border: none; color: var(--text-3); cursor: pointer; padding: 0.0625rem; display: flex; align-items: center; justify-content: center; border-radius: 4px; }
.reorder-btn:hover:not(:disabled) { color: var(--text-1); background: var(--surface-soft); }
.reorder-btn:disabled { opacity: 0.25; cursor: not-allowed; }
.event-picker-check { display: flex; align-items: center; gap: 0.5rem; flex: 1; min-width: 0; flex-wrap: wrap; }
.cat-pill { display: inline-flex; align-items: center; gap: 0.25rem; font-size: var(--fs-2xs); font-weight: var(--fw-semibold); padding: 0.12rem 0.45rem; border-radius: 999px; border: 1px solid var(--border-soft); background: var(--surface-deep); color: var(--text-3); }
.cat-meta, .cat-google, .cat-general { color: var(--brand); border-color: var(--brand-border); }
.event-picker-label-input { width: 42%; background: var(--surface-2); border: 1px solid var(--border); border-radius: 6px; padding: 0.35rem 0.5rem; font-size: var(--fs-sm); color: var(--text-1); outline: none; }
.event-picker-label-input:focus { border-color: var(--brand); }
@media (max-width: 480px) {
  .event-picker-item { flex-wrap: wrap; }
  .event-picker-label-input { width: 100%; margin-left: 1.75rem; }
}
</style>
