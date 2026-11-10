<template>
  <BaseModal v-model="isOpen" :title="title" :description="description" size="sm">
    <div class="event-picker-list">
      <div v-for="(id, idx) in localOrder" :key="id" class="event-picker-item">
        <div class="event-picker-reorder">
          <button
            type="button"
            class="reorder-btn"
            :disabled="idx === 0"
            title="Mover para cima"
            @click="move(idx, -1)"
          >
            <ChevronUp :size="14" />
          </button>
          <button
            type="button"
            class="reorder-btn"
            :disabled="idx === localOrder.length - 1"
            title="Mover para baixo"
            @click="move(idx, 1)"
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
        </div>
        <input
          v-if="editableLabels"
          v-model="localLabels[id]"
          type="text"
          class="event-picker-label-input"
          :placeholder="labelOf(id)"
        />
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
import BaseModal from '~/components/BaseModal.vue'
import BaseButton from '~/components/BaseButton.vue'

interface EventOption {
  id: string
  label: string
}

interface Props {
  modelValue: boolean
  title?: string
  description?: string
  events: EventOption[]
  selected: string[]
  /** Ordem de exibição (ids do catálogo). Se omitido, usa a ordem de `events`. */
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

const labelById = computed(() => {
  const map = new Map<string, string>()
  for (const ev of props.events) map.set(ev.id, ev.label)
  return map
})

const labelOf = (id: string) => labelById.value.get(id) || id

const buildOrder = (): string[] => {
  const catalogIds = props.events.map((e) => e.id)
  const base = props.order && props.order.length ? props.order : catalogIds
  // Garante que todo id do catálogo apareça (mesmo que ausente da ordem salva).
  const merged = [...base.filter((id) => catalogIds.includes(id))]
  for (const id of catalogIds) {
    if (!merged.includes(id)) merged.push(id)
  }
  return merged
}

const localSelected = ref<string[]>([...props.selected])
const localLabels = ref<Record<string, string>>({ ...props.labelOverrides })
const localOrder = ref<string[]>(buildOrder())

watch(() => props.modelValue, (open) => {
  if (!open) return
  localSelected.value = [...props.selected]
  localLabels.value = { ...props.labelOverrides }
  localOrder.value = buildOrder()
})

const move = (idx: number, dir: -1 | 1) => {
  const target = idx + dir
  if (target < 0 || target >= localOrder.value.length) return
  const itemA = localOrder.value[idx]
  const itemB = localOrder.value[target]
  if (itemA === undefined || itemB === undefined) return
  const arr = [...localOrder.value]
  arr[idx] = itemB
  arr[target] = itemA
  localOrder.value = arr
}

const onSave = () => {
  emit('save', { ids: localSelected.value, order: localOrder.value, labels: localLabels.value })
}
</script>

<style scoped>
.event-picker-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.event-picker-item {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.625rem 0.75rem;
  border-radius: 8px;
  border: 1px solid var(--border-soft);
  background: var(--surface-soft-2);
  transition: all 0.15s;
}

.event-picker-item:hover {
  background: var(--surface-soft);
  border-color: var(--border);
}

.event-picker-reorder {
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
}

.reorder-btn {
  background: transparent;
  border: none;
  color: var(--text-3);
  cursor: pointer;
  padding: 0.0625rem;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
}

.reorder-btn:hover:not(:disabled) {
  color: var(--text-1);
  background: var(--surface-soft);
}

.reorder-btn:disabled {
  opacity: 0.25;
  cursor: not-allowed;
}

.event-picker-check {
  display: flex;
  align-items: center;
  flex: 1;
  min-width: 0;
}

.event-picker-label-input {
  width: 45%;
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: 6px;
  padding: 0.375rem 0.5rem;
  font-size: var(--fs-base);
  color: var(--text-1);
  outline: none;
}

.event-picker-label-input:focus {
  border-color: var(--brand);
}

/* Em telas estreitas o input de renomear vai pra própria linha, com largura
   total, em vez de espremer do lado do checkbox. */
@media (max-width: 480px) {
  .event-picker-item {
    flex-wrap: wrap;
  }

  .event-picker-label-input {
    width: 100%;
    margin-left: calc(1rem + 0.75rem);
  }
}
</style>
