<template>
  <div class="triggers-box">
    <div class="triggers-head">
      <p class="box-title">Gatilhos automáticos por mensagem na conversa</p>
      <p class="box-hint">
        Separe múltiplos termos por vírgula. Valores na mensagem (ex: <code>R$ 1.500,00</code>, <code>350 reais</code>, <code>3x de 150</code>) são extraídos automaticamente, e se o evento estiver vinculado a uma etapa do CRM o card é movido sozinho.
      </p>
    </div>

    <div class="triggers-list">
      <div
        v-for="(row, idx) in modelValue"
        :key="row.uid"
        class="trigger-item"
      >
        <div class="trigger-main-row">
          <div class="col-event">
            <BaseDropdown
              :model-value="row.selectedEvent"
              :label="idx === 0 ? 'Evento disparado' : undefined"
              :options="WHATSAPP_TRIGGER_EVENT_OPTIONS"
              @update:model-value="updateRow(idx, { selectedEvent: String($event ?? 'Purchase') })"
            />
          </div>

          <div class="col-phrase">
            <BaseInput
              :model-value="row.trigger"
              :label="idx === 0 ? 'Gatilho na conversa (atendente)' : undefined"
              :placeholder="getWhatsAppTriggerPlaceholder(row.selectedEvent)"
              @update:model-value="updateRow(idx, { trigger: String($event ?? '') })"
            />
          </div>

          <div class="col-action">
            <button
              type="button"
              class="remove-btn"
              :disabled="isRemoveDisabled(row)"
              :title="modelValue.length > 1 ? 'Remover gatilho' : 'Limpar gatilho'"
              @click="removeOrClearRow(idx)"
            >
              <Trash2 :size="15" />
            </button>
          </div>
        </div>

        <div v-if="row.selectedEvent === WHATSAPP_CUSTOM_EVENT_VALUE" class="custom-name-row">
          <BaseInput
            :model-value="row.customEventName"
            label="Nome do evento personalizado"
            placeholder="Ex: PropostaEnviada, ReiaoConfirmada"
            @update:model-value="updateRow(idx, { customEventName: String($event ?? '') })"
          />
        </div>
      </div>
    </div>

    <div class="triggers-footer">
      <BaseButton variant="outline" size="sm" @click="addRow">
        <template #iconLeft>
          <Plus :size="14" />
        </template>
        Adicionar gatilho de evento
      </BaseButton>
    </div>

    <div class="ticket-row">
      <BaseInput
        :model-value="averageTicket"
        :label="`Ticket médio do produto (${currency || 'BRL'} — opcional)`"
        placeholder="Ex: 297,00"
        hint="Caso a mensagem não informe o valor pago, a venda será marcada com este ticket médio do produto."
        @update:model-value="emit('update:averageTicket', String($event ?? ''))"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { Plus, Trash2 } from 'lucide-vue-next'
import {
  WHATSAPP_CUSTOM_EVENT_VALUE,
  WHATSAPP_TRIGGER_EVENT_OPTIONS,
  getWhatsAppTriggerPlaceholder,
  nextTriggerRowUid,
  pickSuggestedNextEvent,
  type EditableWhatsAppTriggerRow,
} from '~/utils/whatsappEventOptions'

const props = defineProps<{
  modelValue: EditableWhatsAppTriggerRow[]
  averageTicket: string
  currency?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [rows: EditableWhatsAppTriggerRow[]]
  'update:averageTicket': [val: string]
}>()

const updateRow = (idx: number, patch: Partial<EditableWhatsAppTriggerRow>) => {
  const next = props.modelValue.map((r, i) => (i === idx ? { ...r, ...patch } : r))
  emit('update:modelValue', next)
}

const addRow = () => {
  const suggested = pickSuggestedNextEvent(props.modelValue)
  emit('update:modelValue', [
    ...props.modelValue,
    { uid: nextTriggerRowUid(), selectedEvent: suggested, customEventName: '', trigger: '' },
  ])
}

const isRemoveDisabled = (row: EditableWhatsAppTriggerRow): boolean => {
  if (props.modelValue.length > 1) return false
  return row.selectedEvent === 'Purchase' && !row.trigger.trim() && !row.customEventName.trim()
}

const removeOrClearRow = (idx: number) => {
  if (props.modelValue.length > 1) {
    emit('update:modelValue', props.modelValue.filter((_, i) => i !== idx))
    return
  }
  emit('update:modelValue', [
    { uid: nextTriggerRowUid(), selectedEvent: 'Purchase', customEventName: '', trigger: '' },
  ])
}
</script>

<style scoped>
.triggers-box { display: flex; flex-direction: column; gap: 0.75rem; padding: 0.875rem; background: var(--surface-deep); border: 1px solid var(--border-soft); border-radius: 10px; }
.triggers-head { display: flex; flex-direction: column; gap: 0.25rem; }
.box-title { font-size: var(--fs-md); font-weight: var(--fw-semibold); color: var(--text-1); margin: 0; }
.box-hint { font-size: var(--fs-sm); color: var(--text-3); margin: 0; line-height: 1.45; }
.box-hint code { font-family: var(--font-mono); font-size: var(--fs-xs); color: var(--text-2); background: var(--surface-2); padding: 0.1rem 0.3rem; border-radius: 4px; }
.triggers-list { display: flex; flex-direction: column; gap: 0.625rem; }
.trigger-item { display: flex; flex-direction: column; gap: 0.5rem; }
.trigger-main-row { display: grid; grid-template-columns: minmax(0, 1fr) minmax(0, 1.15fr) auto; gap: 0.625rem; align-items: end; }
.col-action { display: flex; align-items: center; }
.remove-btn { width: 36px; height: 36px; display: inline-flex; align-items: center; justify-content: center; border-radius: 6px; border: 1px solid var(--border); background: var(--surface-2); color: var(--text-3); cursor: pointer; transition: all 0.15s ease; }
.remove-btn:hover:not(:disabled) { color: var(--danger); border-color: var(--danger); background: var(--danger-soft); }
.remove-btn:disabled { opacity: 0.35; cursor: not-allowed; }
.custom-name-row { padding-left: 0.625rem; border-left: 2px solid var(--brand-border); }
.triggers-footer { display: flex; align-items: center; padding-top: 0.125rem; }
.ticket-row { padding-top: 0.75rem; border-top: 1px solid var(--border-soft); }
@media (max-width: 640px) {
  .trigger-item { padding-bottom: 0.625rem; border-bottom: 1px solid var(--border-faint); }
  .trigger-main-row { grid-template-columns: 1fr auto; }
  .col-event { grid-column: 1 / -1; }
}
</style>
