<template>
  <div class="filters-bar">
    <PixelDropdown
      :model-value="modelValue.pixel || ''"
      label="Pixel"
      @update:model-value="(v) => update('pixel', v)"
    />
    <MultiSelectDropdown
      :model-value="selectedEvents"
      label="Evento"
      placeholder="Todos os eventos"
      :options="eventOptions"
      @update:model-value="updateEvents"
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
import MultiSelectDropdown from '~/components/MultiSelectDropdown.vue'
import type { EventFilters } from '#shared/types'

interface Props {
  modelValue: EventFilters
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:modelValue': [value: EventFilters]
}>()

// PageView e ViewContent não aparecem — ficam de fora de sent_events
// (alto volume, só agregados no dashboard). Ver domain.ShouldStoreFullPayload.
// "Todos os eventos" não é uma opção aqui: nenhum evento selecionado já
// significa "sem filtro" (mostra tudo) no multi-select.
const eventOptions = [
  { value: 'AddToCart', label: 'AddToCart' },
  { value: 'AddToWishlist', label: 'AddToWishlist' },
  { value: 'InitiateCheckout', label: 'InitiateCheckout' },
  { value: 'Lead', label: 'Lead' },
  { value: 'LeadSubmitted', label: 'LeadSubmitted' },
  { value: 'Purchase', label: 'Purchase' },
  { value: 'AddPaymentInfo', label: 'AddPaymentInfo' },
  { value: 'CompleteRegistration', label: 'CompleteRegistration' },
  { value: 'Contact', label: 'Contact' },
  // Eventos-padrão da Meta disponíveis como "Evento de Disparo" no CRM
  // (crm/index.vue) — precisam existir aqui também pra poder filtrar.
  { value: 'Search', label: 'Search' },
  { value: 'Subscribe', label: 'Subscribe' },
  { value: 'FindLocation', label: 'FindLocation' },
  { value: 'Schedule', label: 'Schedule' },
  { value: 'StartTrial', label: 'StartTrial' },
  { value: 'SubmitApplication', label: 'SubmitApplication' },
  { value: 'Donate', label: 'Donate' },
  { value: 'CustomizeProduct', label: 'CustomizeProduct' },
  { value: 'Scroll_25', label: 'Scroll_25' },
  { value: 'Scroll_50', label: 'Scroll_50' },
  { value: 'Scroll_75', label: 'Scroll_75' },
  { value: 'Scroll_90', label: 'Scroll_90' },
  { value: 'Timer_1min', label: 'Timer_1min' },
  { value: 'PlayVideo', label: 'PlayVideo' },
  { value: 'ViewVideo_25', label: 'ViewVideo_25' },
  { value: 'ViewVideo_50', label: 'ViewVideo_50' },
  { value: 'ViewVideo_75', label: 'ViewVideo_75' },
  { value: 'ViewVideo_90', label: 'ViewVideo_90' },
]

const statusOptions = [
  { value: '', label: 'Todos os status' },
  { value: 'sent', label: 'Enviado' },
  { value: 'queued', label: 'Na fila (Pendente)' },
  { value: 'failed', label: 'Falhou' },
]

const fromInput = computed(() => isoToInput(props.modelValue.from))
const toInput = computed(() => isoToInput(props.modelValue.to))

// modelValue.event trafega como CSV ("Lead,Purchase") — o back-end aceita
// múltiplos valores nesse único query param. O dropdown trabalha com array.
const selectedEvents = computed<string[]>(() =>
  (props.modelValue.event || '').split(',').map((s) => s.trim()).filter(Boolean)
)

const updateEvents = (values: string[]) => {
  update('event', values.join(','))
}

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
