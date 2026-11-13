<template>
  <div class="triggers-box">
    <p class="box-title">Gatilhos automáticos por mensagem na conversa</p>
    <p class="box-hint">
      Separe múltiplos termos por vírgula. Qualquer valor na mensagem (ex: <code>R$ 1.500,00</code>, <code>350 reais</code>, <code>valor 347</code>, <code>1,5 mil</code>, <code>3x de 150</code>) é extraído automaticamente e enviado junto com a conversão.
    </p>
    <div class="grid-2">
      <BaseInput
        :model-value="leadTrigger"
        label="Gatilho de Lead Qualificado"
        placeholder="Ex: proposta enviada, orçamento"
        @update:model-value="$emit('update:leadTrigger', String($event ?? ''))"
      />
      <BaseInput
        :model-value="purchaseTrigger"
        label="Gatilho de Venda (Purchase)"
        placeholder="Ex: pagamento confirmado, pix recebido"
        @update:model-value="$emit('update:purchaseTrigger', String($event ?? ''))"
      />
    </div>
    <div class="grid-2">
      <BaseInput
        :model-value="customEventName"
        label="Evento Personalizado (opcional)"
        placeholder="Ex: Schedule"
        @update:model-value="$emit('update:customEventName', String($event ?? ''))"
      />
      <BaseInput
        :model-value="customEventTrigger"
        label="Gatilho do Evento Personalizado"
        placeholder="Ex: reunião agendada, horário marcado"
        @update:model-value="$emit('update:customEventTrigger', String($event ?? ''))"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  leadTrigger: string
  purchaseTrigger: string
  customEventName: string
  customEventTrigger: string
}>()

defineEmits<{
  'update:leadTrigger': [val: string]
  'update:purchaseTrigger': [val: string]
  'update:customEventName': [val: string]
  'update:customEventTrigger': [val: string]
}>()
</script>

<style scoped>
.triggers-box { display: flex; flex-direction: column; gap: 0.75rem; padding: 0.875rem; background: var(--surface-deep); border: 1px solid var(--border-soft); border-radius: 10px; }
.box-title { font-size: var(--fs-md); font-weight: var(--fw-semibold); color: var(--text-1); margin: 0; }
.box-hint { font-size: var(--fs-sm); color: var(--text-3); margin: -0.2rem 0 0; line-height: 1.45; }
.box-hint code { font-family: var(--font-mono); font-size: var(--fs-xs); color: var(--text-2); background: var(--surface-2); padding: 0.1rem 0.3rem; border-radius: 4px; }
.grid-2 { display: grid; grid-template-columns: 1fr 1fr; gap: 0.75rem; }
@media (max-width: 640px) { .grid-2 { grid-template-columns: 1fr; } }
</style>

