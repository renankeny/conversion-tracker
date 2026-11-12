<template>
  <BaseModal
    :model-value="modelValue"
    :title="isCreating ? 'Nova Etapa' : 'Configurar Etapa'"
    size="md"
    @update:model-value="onModalUpdate"
  >
    <div class="form-group">
      <BaseInput
        v-model="stage.name"
        label="Nome da Etapa"
        placeholder="Ex: Proposta Enviada"
      />
    </div>

    <div class="form-group">
      <label class="field-label">Cor de Destaque</label>
      <div class="color-picker-row">
        <input v-model="stage.color" type="color" class="color-picker" />
        <div class="color-presets">
          <button
            v-for="c in colorPresets"
            :key="c"
            type="button"
            class="color-preset-dot"
            :class="{ active: stage.color === c }"
            :style="{ backgroundColor: c }"
            @click="stage.color = c"
          ></button>
        </div>
      </div>
    </div>

    <div class="form-group">
      <BaseDropdown
        v-model="stage.trigger_event"
        label="Evento Web / Site (Trigger Automático)"
        placeholder="Nenhum"
        :options="CRM_WEB_TRIGGER_EVENTS"
        hint="Disparado quando um visitante do site entrar nesta etapa."
      />
    </div>

    <div class="form-group">
      <BaseDropdown
        v-model="stage.trigger_event_ctwa"
        label="Evento WhatsApp / CTWA (Business Messaging)"
        placeholder="Nenhum"
        :options="CRM_CTWA_TRIGGER_EVENTS"
        hint="Disparado via Meta Business Messaging CAPI para leads de anúncios de WhatsApp."
      />
    </div>

    <div class="form-group stage-extra-config">
      <BaseAdAccountSelect
        :model-value="stage.google_ads_account_id"
        label="Forçar Conta Google Ads (Opcional)"
        placeholder="Nenhuma (Usar padrão do Pixel)"
        :options="[
          { value: null, label: 'Nenhuma (Usar padrão do Pixel)', channel: 'none' },
          ...gadsAccounts.map((acc) => ({
            value: acc.id,
            label: acc.name || acc.customer_id,
            sublabel: acc.customer_id || acc.oauth_email || '',
            avatarUrl: acc.picture_url || acc.oauth_picture_url || '',
            channel: 'google_ads' as const,
          }))
        ]"
        hint="Substitui a conta configurada no Pixel apenas para esta etapa."
        @update:model-value="emit('change-gads-account', $event === null || $event === '' ? null : Number($event))"
      />
    </div>

    <div v-if="stage.google_ads_account_id" class="form-group">
      <BaseDropdown
        v-model="stage.google_ads_action_id"
        label="Ação de Conversão (Google Ads)"
        :placeholder="isLoadingGadsActions ? 'Carregando conversões...' : 'Automática (Pelo nome do evento)'"
        :disabled="isLoadingGadsActions"
        :options="[
          { value: '', label: 'Automática (Pelo nome do evento)' },
          ...gadsActions.map((action) => ({
            value: action.conversion_action_id,
            label: `${action.name} (${action.event_type})`,
          }))
        ]"
      />
    </div>

    <div class="modal-actions space-between">
      <BaseButton
        v-if="!isCreating"
        variant="danger"
        :loading="loading"
        :disabled="!canDeleteStage"
        @click="emit('delete')"
      >
        Excluir Etapa
      </BaseButton>
      <div v-else></div>
      <div class="actions-right">
        <BaseButton variant="ghost" @click="emit('close')">Cancelar</BaseButton>
        <BaseButton variant="primary" :loading="loading" @click="emit('save')">Salvar Etapa</BaseButton>
      </div>
    </div>
  </BaseModal>
</template>

<script setup lang="ts">
import type { CrmStageForm, GoogleAdsAccount } from '#shared/types'
import { CRM_WEB_TRIGGER_EVENTS, CRM_CTWA_TRIGGER_EVENTS } from '~/utils/crmEventOptions'

withDefaults(defineProps<{
  modelValue: boolean
  isCreating: boolean
  loading: boolean
  isLoadingGadsActions?: boolean
  canDeleteStage?: boolean
  stage: CrmStageForm
  gadsAccounts: GoogleAdsAccount[]
  gadsActions: Array<{ conversion_action_id: string; name: string; event_type: string }>
}>(), {
  isLoadingGadsActions: false,
  canDeleteStage: true,
})

const emit = defineEmits<{
  (e: 'update:modelValue', val: boolean): void
  (e: 'change-gads-account', accountId: number | null): void
  (e: 'close'): void
  (e: 'save'): void
  (e: 'delete'): void
}>()

const onModalUpdate = (val: boolean) => {
  if (!val) {
    emit('close')
    return
  }
  emit('update:modelValue', val)
}

const colorPresets = [
  '#3B82F6',
  '#10B981',
  '#F59E0B',
  '#EF4444',
  '#8B5CF6',
  '#EC4899',
  '#A2F07C',
]
</script>

<style scoped>
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

.color-preset-dot:hover {
  transform: scale(1.15);
}

.color-preset-dot.active {
  border-color: var(--text-strong);
  transform: scale(1.1);
}

.stage-extra-config {
  margin-top: 0.75rem;
  padding-top: 0.75rem;
  border-top: 1px solid var(--border-soft);
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  margin-top: 1.5rem;
}

.modal-actions.space-between {
  justify-content: space-between;
}

.actions-right {
  display: flex;
  gap: 0.75rem;
}
</style>
