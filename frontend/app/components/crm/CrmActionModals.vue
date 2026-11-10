<template>
  <ConfirmModal
    :model-value="showDeleteLead"
    title="Excluir Lead?"
    message="Tem certeza que deseja excluir este Lead? Todas as anotações e vínculos serão perdidos e esta ação não pode ser desfeita."
    confirm-label="Excluir Lead"
    variant="danger"
    @update:model-value="onDeleteLeadModalChange"
    @confirm="emit('confirm-delete-lead')"
    @cancel="emit('cancel-delete-lead')"
  />

  <ConfirmModal
    :model-value="showDeleteNote"
    title="Excluir Anotação?"
    message="Tem certeza que deseja excluir esta anotação? Esta ação não pode ser desfeita."
    confirm-label="Excluir"
    variant="danger"
    @update:model-value="emit('update:showDeleteNote', $event)"
    @confirm="emit('confirm-delete-note')"
  />

  <ConfirmModal
    :model-value="showDeleteStage"
    title="Excluir Etapa?"
    message="Tem certeza que deseja excluir esta etapa? Os leads dela serão movidos automaticamente para a primeira coluna do funil."
    confirm-label="Excluir Etapa"
    variant="danger"
    :loading="isDeletingStage"
    @update:model-value="emit('update:showDeleteStage', $event)"
    @confirm="emit('confirm-delete-stage')"
  />

  <BaseModal
    :model-value="showPurchase"
    title="Confirmar Venda"
    size="sm"
    @update:model-value="onPurchaseModalChange"
  >
    <p class="modal-desc">Essa etapa dispara o evento de <strong>Purchase</strong>. Defina o valor da conversão:</p>
    <div class="form-group" @keydown.enter.prevent="onPurchaseEnter">
      <BaseInput
        :model-value="purchaseValue ?? ''"
        type="number"
        label="Valor (R$)"
        placeholder="0.00"
        @update:model-value="emit('update:purchaseValue', $event === '' ? null : Number($event))"
      />
    </div>
    <div class="modal-actions">
      <BaseButton variant="ghost" :disabled="isConfirmingPurchase" @click="emit('cancel-purchase')">
        Cancelar
      </BaseButton>
      <BaseButton
        variant="primary"
        :loading="isConfirmingPurchase"
        :disabled="!purchaseValue || purchaseValue <= 0 || isConfirmingPurchase"
        @click="emit('confirm-purchase')"
      >
        Salvar e Mover
      </BaseButton>
    </div>
  </BaseModal>

  <BaseModal
    :model-value="showCreateTag"
    title="Nova Tag"
    size="sm"
    @update:model-value="onCreateTagModalChange"
  >
    <div class="form-group" @keydown.enter.prevent="newTag.name.trim() ? emit('confirm-create-tag') : undefined">
      <BaseInput
        v-model="newTag.name"
        label="Nome"
        placeholder="Ex: Quente, Prioritário..."
      />
    </div>
    <div class="form-group">
      <label class="field-label">Cor</label>
      <div class="color-picker-row">
        <input v-model="newTag.color" type="color" class="color-picker" />
        <div class="color-presets">
          <button
            v-for="c in tagColorPresets"
            :key="c"
            type="button"
            class="color-preset-dot"
            :class="{ active: newTag.color === c }"
            :style="{ backgroundColor: c }"
            @click="newTag.color = c"
          ></button>
        </div>
      </div>
    </div>
    <div class="modal-actions">
      <BaseButton variant="ghost" @click="emit('cancel-create-tag')">Cancelar</BaseButton>
      <BaseButton
        variant="primary"
        :loading="isCreatingTag"
        :disabled="!newTag.name.trim() || isCreatingTag"
        @click="emit('confirm-create-tag')"
      >
        Criar Tag
      </BaseButton>
    </div>
  </BaseModal>
</template>

<script setup lang="ts">
const props = defineProps<{
  showDeleteLead: boolean
  showDeleteNote: boolean
  showDeleteStage?: boolean
  isDeletingStage?: boolean
  showPurchase: boolean
  purchaseValue: number | null
  isConfirmingPurchase?: boolean
  showCreateTag: boolean
  isCreatingTag: boolean
  newTag: { name: string; color: string }
}>()

const emit = defineEmits<{
  (e: 'update:showDeleteLead', val: boolean): void
  (e: 'confirm-delete-lead'): void
  (e: 'cancel-delete-lead'): void
  (e: 'update:showDeleteNote', val: boolean): void
  (e: 'confirm-delete-note'): void
  (e: 'update:showDeleteStage', val: boolean): void
  (e: 'confirm-delete-stage'): void
  (e: 'update:showPurchase', val: boolean): void
  (e: 'update:purchaseValue', val: number | null): void
  (e: 'cancel-purchase'): void
  (e: 'confirm-purchase'): void
  (e: 'update:showCreateTag', val: boolean): void
  (e: 'cancel-create-tag'): void
  (e: 'confirm-create-tag'): void
}>()

const tagColorPresets = ['#3B82F6', '#10B981', '#F59E0B', '#EF4444', '#8B5CF6', '#EC4899']

const onDeleteLeadModalChange = (val: boolean) => {
  if (!val) {
    emit('cancel-delete-lead')
    return
  }
  emit('update:showDeleteLead', val)
}

const onCreateTagModalChange = (val: boolean) => {
  if (!val) {
    emit('cancel-create-tag')
    return
  }
  emit('update:showCreateTag', val)
}

const onPurchaseModalChange = (val: boolean) => {
  if (!val) {
    emit('cancel-purchase')
    return
  }
  emit('update:showPurchase', val)
}

const onPurchaseEnter = () => {
  if (props.purchaseValue && props.purchaseValue > 0 && !props.isConfirmingPurchase) {
    emit('confirm-purchase')
  }
}
</script>

<style scoped>
.modal-desc {
  color: var(--text-3);
  font-size: var(--fs-base);
  margin-bottom: 1.25rem;
  line-height: 1.4;
}
.form-group {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
  margin-bottom: 0.875rem;
}
.field-label {
  font-size: var(--fs-base);
  font-weight: var(--fw-medium);
  color: var(--text-2);
}
.color-picker-row {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}
.color-picker {
  padding: 0.2rem;
  width: 44px;
  height: 36px;
  border-radius: 6px;
  border: 1px solid var(--border);
  background: var(--surface-1);
  cursor: pointer;
}
.color-presets {
  display: flex;
  align-items: center;
  gap: 0.4rem;
}
.color-preset-dot {
  width: 22px;
  height: 22px;
  border-radius: 50%;
  border: 2px solid transparent;
  cursor: pointer;
  transition: transform 0.15s, border-color 0.15s;
}
.color-preset-dot.active {
  border-color: var(--text-strong);
  transform: scale(1.1);
}
.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  margin-top: 1.5rem;
}
</style>
