<template>
  <BaseModal
    :model-value="modelValue"
    title="Contestar decisão da IA"
    description="A marcação será revertida no Kanban e o seu motivo ensinará a IA a não repetir esse comportamento nas próximas conversas."
    size="sm"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <form class="contest-form" @submit.prevent="onSubmit">
      <div v-if="event" class="summary-strip">
        <span class="sum-type">{{ event.event_type === 'purchase' ? 'Venda identificada' : 'Qualificação identificada' }}</span>
        <p v-if="event.ai_reason" class="sum-reason">{{ event.ai_reason }}</p>
      </div>

      <div class="field-group">
        <label class="field-label" for="contest-reason-input">Motivo da contestação</label>
        <textarea
          id="contest-reason-input"
          v-model="reason"
          rows="4"
          class="reason-textarea"
          placeholder="Ex: O cliente pediu apenas um orçamento e ainda não enviou comprovante de PIX pago."
        />
        <span v-if="errorMsg" class="field-err">{{ errorMsg }}</span>
      </div>

      <BaseCheckbox
        v-model="addToPrompt"
        label="Adicionar este motivo ao prompt da IA para sempre levar em consideração"
      />
    </form>

    <template #footer>
      <BaseButton variant="ghost" @click="emit('update:modelValue', false)">
        Cancelar
      </BaseButton>
      <BaseButton variant="danger" :loading="submitting" @click="onSubmit">
        Confirmar contestação
      </BaseButton>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import type { WhatsAppAIEvent } from '#shared/types'
import { errorMessage } from '~/utils/errorMessage'

const props = defineProps<{
  modelValue: boolean
  event: WhatsAppAIEvent | null
}>()

const emit = defineEmits<{
  'update:modelValue': [val: boolean]
  contested: [updated: WhatsAppAIEvent]
}>()

const wa = useWhatsApp()
const toast = useToast()

const reason = ref('')
const addToPrompt = ref(true)
const errorMsg = ref('')
const submitting = ref(false)

watch(
  () => props.modelValue,
  (open) => {
    if (open) {
      reason.value = ''
      addToPrompt.value = true
      errorMsg.value = ''
    }
  },
)

const onSubmit = async () => {
  if (!props.event) return
  errorMsg.value = ''
  const clean = reason.value.trim()
  if (!clean) {
    errorMsg.value = 'Descreva o motivo da contestação.'
    return
  }
  submitting.value = true
  try {
    const updated = await wa.contestAIEvent(props.event.id, {
      reason: clean,
      add_to_prompt: addToPrompt.value,
    })
    toast.success('Marcação contestada', addToPrompt.value ? 'A etapa foi revertida e a regra foi adicionada ao prompt da IA.' : 'A marcação foi revertida.')
    emit('contested', updated)
    emit('update:modelValue', false)
  } catch (e: unknown) {
    toast.error('Erro ao contestar', errorMessage(e, 'Não foi possível contestar este evento.'))
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
.contest-form { display: flex; flex-direction: column; gap: 0.9rem; }
.summary-strip { padding: 0.65rem 0.8rem; border-radius: 8px; background: var(--surface-deep); border: 1px solid var(--border-soft); display: flex; flex-direction: column; gap: 0.25rem; }
.sum-type { font-size: var(--fs-xs); font-weight: var(--fw-semibold); color: var(--brand); text-transform: uppercase; letter-spacing: 0.04em; }
.sum-reason { margin: 0; font-size: var(--fs-xs); color: var(--text-2); line-height: 1.4; }
.field-group { display: flex; flex-direction: column; gap: 0.4rem; }
.field-label { font-size: var(--fs-sm); font-weight: var(--fw-semibold); color: var(--text-1); }
.reason-textarea { width: 100%; padding: 0.7rem 0.8rem; border-radius: 8px; background: var(--surface-deep); border: 1px solid var(--border); color: var(--text-1); font-family: var(--font-sans); font-size: var(--fs-base); line-height: 1.45; resize: vertical; outline: none; }
.reason-textarea:focus { border-color: var(--brand); }
.field-err { font-size: var(--fs-xs); color: var(--danger); }
</style>

