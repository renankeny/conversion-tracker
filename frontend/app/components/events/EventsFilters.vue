<template>
  <div class="filters-panel">
    <div class="filters-grid">
      <div class="search-field">
        <BaseInput
          :model-value="searchInput"
          label="Busca rápida"
          placeholder="ID do evento, usuário, e-mail ou telefone"
          @update:model-value="onSearchInput"
        >
          <template #icon>
            <Search class="w-4 h-4" />
          </template>
        </BaseInput>
      </div>

      <PixelDropdown
        :model-value="modelValue.pixel || ''"
        label="Pixel"
        @update:model-value="(v) => update('pixel', v)"
      />

      <MultiSelectDropdown
        :model-value="selectedEvents"
        label="Evento"
        placeholder="Todos os eventos"
        :options="EVENT_FILTER_OPTIONS"
        @update:model-value="updateEvents"
      />

      <BaseDropdown
        :model-value="modelValue.channel || ''"
        label="Canal"
        :options="CHANNEL_FILTER_OPTIONS"
        @update:model-value="(v) => update('channel', String(v || ''))"
      />

      <BaseDropdown
        :model-value="modelValue.status || ''"
        label="Status"
        :options="STATUS_FILTER_OPTIONS"
        @update:model-value="(v) => updateStatus(String(v || ''))"
      />
    </div>

    <EventsPeriodPresets
      :from="modelValue.from"
      :to="modelValue.to"
      @update-range="onUpdateRange"
      @clear="onClear"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onBeforeUnmount } from 'vue'
import { Search } from 'lucide-vue-next'
import BaseDropdown from '~/components/BaseDropdown.vue'
import MultiSelectDropdown from '~/components/MultiSelectDropdown.vue'
import EventsPeriodPresets from '~/components/events/EventsPeriodPresets.vue'
import {
  EVENT_FILTER_OPTIONS,
  CHANNEL_FILTER_OPTIONS,
  STATUS_FILTER_OPTIONS,
} from '~/components/events/eventsFilterOptions'
import type { EventFilters, EventStatus } from '#shared/types'

const props = defineProps<{
  modelValue: EventFilters
}>()

const emit = defineEmits<{
  'update:modelValue': [value: EventFilters]
  clear: []
}>()

const searchInput = ref(props.modelValue.search || '')
let searchTimer: ReturnType<typeof setTimeout> | null = null

watch(
  () => props.modelValue.search,
  (next) => {
    const val = next || ''
    if (val !== searchInput.value) searchInput.value = val
  },
)

onBeforeUnmount(() => {
  if (searchTimer) clearTimeout(searchTimer)
})

const onSearchInput = (val: string) => {
  searchInput.value = val
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    emit('update:modelValue', { ...props.modelValue, search: val.trim() || undefined })
  }, 350)
}

const selectedEvents = computed<string[]>(() =>
  (props.modelValue.event || '').split(',').map((s) => s.trim()).filter(Boolean),
)

const update = (key: keyof EventFilters, value: string) => {
  emit('update:modelValue', { ...props.modelValue, [key]: value || undefined })
}

const updateStatus = (value: string) => {
  emit('update:modelValue', {
    ...props.modelValue,
    status: (value || undefined) as EventStatus | undefined,
  })
}

const updateEvents = (values: string[]) => {
  update('event', values.join(','))
}

const onUpdateRange = (range: { from?: string; to?: string }) => {
  emit('update:modelValue', { ...props.modelValue, from: range.from, to: range.to })
}

const onClear = () => {
  if (searchTimer) clearTimeout(searchTimer)
  searchInput.value = ''
  emit('clear')
}
</script>

<style scoped>
.filters-panel {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 1rem;
}

.filters-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(170px, 1fr));
  gap: 0.75rem;
  align-items: end;
}

@media (min-width: 1200px) {
  .filters-grid {
    grid-template-columns: 1.4fr 1fr 1fr 1fr 1fr;
  }
}
</style>
