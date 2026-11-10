<template>
  <BaseModal
    :model-value="modelValue"
    title="Cadastrar Novo Lead"
    size="md"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <p class="modal-desc">Adicione um novo lead diretamente no seu funil com oportunidade e etapa inicial.</p>

    <div class="form-grid-2">
      <BaseInput
        v-model="form.first_name"
        label="Nome"
        placeholder="Ex: João"
      />
      <BaseInput
        v-model="form.last_name"
        label="Sobrenome"
        placeholder="Ex: Silva"
      />
    </div>

    <div class="form-grid-2">
      <BaseInput
        v-model="form.email"
        type="email"
        label="E-mail"
        placeholder="cliente@email.com"
      />
      <BaseInput
        v-model="form.phone"
        label="Telefone / WhatsApp"
        placeholder="(11) 99999-9999"
      />
    </div>

    <div class="form-grid-2">
      <BaseInput
        :model-value="form.value || ''"
        type="number"
        label="Valor da Oportunidade (R$)"
        placeholder="0.00"
        @update:model-value="form.value = $event === '' ? 0 : Number($event)"
      />
      <BaseDropdown
        v-model="form.stage_id"
        label="Etapa Inicial"
        :options="stages.map((st) => ({ value: st.id, label: st.name }))"
      />
    </div>

    <div class="modal-actions">
      <BaseButton variant="ghost" @click="emit('update:modelValue', false)">Cancelar</BaseButton>
      <BaseButton variant="primary" :loading="loading" @click="emit('submit')">
        Salvar Lead
      </BaseButton>
    </div>
  </BaseModal>
</template>

<script setup lang="ts">
import type { CrmCreateLeadForm, CrmStage } from '#shared/types'

defineProps<{
  modelValue: boolean
  form: CrmCreateLeadForm
  stages: CrmStage[]
  loading: boolean
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', val: boolean): void
  (e: 'submit'): void
}>()
</script>

<style scoped>
.modal-desc {
  color: var(--text-3);
  font-size: var(--fs-base);
  margin-bottom: 1.25rem;
  line-height: 1.4;
}

.form-grid-2 {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1rem;
  margin-bottom: 0.875rem;
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  margin-top: 1.5rem;
}

@media (max-width: 640px) {
  .form-grid-2 {
    grid-template-columns: 1fr;
    gap: 0.75rem;
  }
}
</style>
