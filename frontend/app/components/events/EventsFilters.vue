<template>
  <div class="filters-bar">
    <BaseInput
      :model-value="modelValue.pixel || ''"
      label="Pixel"
      placeholder="Nome do pixel"
      @update:model-value="(v) => update('pixel', v)"
    />
    <BaseDropdown
      :model-value="modelValue.event || ''"
      label="Evento"
      :options="eventOptions"
      @update:model-value="(v) => update('event', v)"
    />
    <BaseDropdown
      :model-value="modelValue.status || ''"
      label="Status"
      :options="statusOptions"
      @update:model-value="(v) => update('status', v)"
    />
    <BaseInput
      :model-value="fromInput"
      label="De"
      type="datetime-local"
      @update:model-value="(v) => updateDate('from', v)"
    />
    <BaseInput
      :model-value="toInput"
      label="Até"
      type="datetime-local"
      @update:model-value="(v) => updateDate('to', v)"
    />
    <div class="filters-actions">
      <BaseButton variant="ghost" size="sm" @click="clear">Limpar</BaseButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import BaseDropdown from '~/components/BaseDropdown.vue'
import type { EventFilters } from '#shared/types'

interface Props {
  modelValue: EventFilters
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:modelValue': [value: EventFilters]
}>()

const eventOptions = [
  { value: '', label: 'Todos os eventos' },
  { value: 'PageView', label: 'PageView' },
  { value: 'ViewContent', label: 'ViewContent' },
  { value: 'AddToCart', label: 'AddToCart' },
  { value: 'InitiateCheckout', label: 'InitiateCheckout' },
  { value: 'Lead', label: 'Lead' },
  { value: 'Purchase', label: 'Purchase' },
  { value: 'AddPaymentInfo', label: 'AddPaymentInfo' },
  { value: 'CompleteRegistration', label: 'CompleteRegistration' },
  { value: 'Contact', label: 'Contact' },
]

const statusOptions = [
  { value: '', label: 'Todos os status' },
  { value: 'sent', label: 'Enviado' },
  { value: 'pending', label: 'Pendente' },
  { value: 'failed', label: 'Falhou' },
]

const fromInput = computed(() => isoToInput(props.modelValue.from))
const toInput = computed(() => isoToInput(props.modelValue.to))

const update = (key: keyof EventFilters, value: string) => {
  emit('update:modelValue', { ...props.modelValue, [key]: value || undefined })
}

const updateDate = (key: 'from' | 'to', input: string) => {
  if (!input) {
    update(key, '')
    return
  }
  // datetime-local vem sem timezone; assume horário local do navegador.
  const d = new Date(input)
  if (Number.isNaN(d.getTime())) return
  update(key, d.toISOString())
}

const clear = () => {
  emit('update:modelValue', {})
}

function isoToInput(iso?: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  // Format como datetime-local (YYYY-MM-DDTHH:MM) usando timezone local.
  const pad = (n: number) => String(n).padStart(2, '0')
  return (
    d.getFullYear() +
    '-' +
    pad(d.getMonth() + 1) +
    '-' +
    pad(d.getDate()) +
    'T' +
    pad(d.getHours()) +
    ':' +
    pad(d.getMinutes())
  )
}
</script>

<style scoped>
.filters-bar {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(170px, 1fr));
  gap: 0.75rem;
  align-items: end;
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 1rem;
}

.filters-actions {
  display: flex;
  align-items: flex-end;
  justify-content: flex-end;
}
</style>
