<template>
  <div class="filters-secondary-row">
    <div class="dates-group">
      <BaseInput
        :model-value="fromInput"
        label="De"
        type="datetime-local"
        @update:model-value="(v) => onDateInput('from', v)"
      />
      <BaseInput
        :model-value="toInput"
        label="Até"
        type="datetime-local"
        @update:model-value="(v) => onDateInput('to', v)"
      />
    </div>

    <div class="presets-and-actions">
      <div class="presets-group" role="group" aria-label="Atalhos de período">
        <button
          v-for="preset in PERIOD_PRESETS"
          :key="preset.id"
          type="button"
          class="preset-btn"
          :class="{ 'is-active': activePreset === preset.id }"
          @click="applyPreset(preset.id)"
        >
          {{ preset.label }}
        </button>
      </div>

      <BaseButton variant="ghost" size="sm" @click="$emit('clear')">
        Limpar
      </BaseButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import {
  PERIOD_PRESETS,
  buildPresetFromISO,
  isoToLocalInput,
  type PeriodPresetId,
} from '~/components/events/eventsFilterOptions'

const props = defineProps<{
  from?: string
  to?: string
}>()

const emit = defineEmits<{
  updateRange: [range: { from?: string; to?: string }]
  clear: []
}>()

const activePreset = ref<PeriodPresetId | ''>('')

watch(
  () => props.from,
  (next) => {
    if (!next) activePreset.value = ''
  },
)

const fromInput = computed(() => isoToLocalInput(props.from))
const toInput = computed(() => isoToLocalInput(props.to))

const onDateInput = (key: 'from' | 'to', input: string) => {
  activePreset.value = ''
  if (!input) {
    emit('updateRange', {
      from: key === 'from' ? undefined : props.from,
      to: key === 'to' ? undefined : props.to,
    })
    return
  }
  const d = new Date(input)
  if (Number.isNaN(d.getTime())) return
  const iso = d.toISOString()
  emit('updateRange', {
    from: key === 'from' ? iso : props.from,
    to: key === 'to' ? iso : props.to,
  })
}

const applyPreset = (preset: PeriodPresetId) => {
  if (activePreset.value === preset) {
    activePreset.value = ''
    emit('updateRange', { from: undefined, to: undefined })
    return
  }
  activePreset.value = preset
  emit('updateRange', { from: buildPresetFromISO(preset), to: undefined })
}
</script>

<style scoped>
.filters-secondary-row { display: flex; align-items: flex-end; justify-content: space-between; gap: 0.75rem; flex-wrap: wrap; padding-top: 0.625rem; border-top: 1px solid var(--border-soft); }
.dates-group { display: grid; grid-template-columns: repeat(2, minmax(170px, 210px)); gap: 0.75rem; flex: 1; min-width: 260px; }
.presets-and-actions { display: flex; align-items: center; gap: 0.625rem; flex-wrap: wrap; }
.presets-group { display: inline-flex; align-items: center; gap: 0.25rem; padding: 0.1875rem; background: var(--surface-deep); border: 1px solid var(--border-soft); border-radius: 8px; }
.preset-btn { padding: 0.3125rem 0.625rem; border-radius: 6px; border: 1px solid transparent; background: transparent; color: var(--text-2); font-size: var(--fs-sm); font-weight: var(--fw-medium); cursor: pointer; transition: all 0.15s ease; white-space: nowrap; }
.preset-btn:hover { color: var(--text-1); background: var(--surface-soft); }
.preset-btn.is-active { background: var(--brand-soft); color: var(--brand); border-color: var(--brand-border); font-weight: var(--fw-semibold); }
@media (max-width: 767px) {
  .dates-group { grid-template-columns: 1fr; width: 100%; }
  .presets-and-actions { width: 100%; justify-content: space-between; }
}
</style>
