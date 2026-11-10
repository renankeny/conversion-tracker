<template>
  <BaseModal
    :model-value="modelValue"
    size="lg"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <template #header>
      <div class="lead-modal-header-custom">
        <div class="lead-modal-identity">
          <div class="lead-modal-avatar">
            {{ (leadEdit.first_name || leadEdit.phone || 'L').slice(0, 2).toUpperCase() }}
          </div>
          <div>
            <h2 class="lead-modal-title">
              {{ leadEdit.first_name || leadEdit.last_name ? `${leadEdit.first_name} ${leadEdit.last_name}`.trim() : 'Lead sem nome' }}
            </h2>
            <div class="lead-modal-sub">
              <span v-if="leadEdit.phone">{{ leadEdit.phone }}</span>
              <span v-if="leadEdit.phone && leadEdit.email">•</span>
              <span v-if="leadEdit.email">{{ leadEdit.email }}</span>
            </div>
          </div>
        </div>

        <div class="lead-modal-quick-actions">
          <a
            v-if="leadEdit.phone"
            :href="getWhatsAppLink(leadEdit.phone, leadEdit.first_name)"
            target="_blank"
            rel="noopener noreferrer"
            class="btn-wpp-lead-modal"
            title="Abrir no WhatsApp"
          >
            <MessageCircle :size="15" />
            <span>WhatsApp</span>
          </a>
          <BaseButton
            v-if="trackedUser"
            variant="outline"
            size="sm"
            class="journey-btn"
            title="Ver Linha do Tempo e Visitas do Visitante"
            @click="emit('open-journey')"
          >
            <Compass :size="14" class="journey-icon" />
            Jornada {{ brandName }}
          </BaseButton>
        </div>
      </div>
    </template>

    <div v-if="leadId" class="lead-modal-body-grid">
      <div class="lead-col-main">
        <div class="info-card-box">
          <h4 class="box-subtitle">Oportunidade de Venda</h4>
          <div class="form-grid-2">
            <BaseInput
              :model-value="leadEdit.value || ''"
              type="number"
              label="Valor da Negociação (R$)"
              placeholder="0.00"
              @update:model-value="leadEdit.value = $event === '' ? 0 : Number($event)"
            />
            <BaseDropdown
              v-model="leadEdit.stage_id"
              label="Etapa do Funil"
              :options="stages.map((st) => ({ value: st.id, label: st.name }))"
            />
          </div>
        </div>

        <div class="info-card-box">
          <h4 class="box-subtitle">Dados do Contato</h4>
          <div class="form-grid-2">
            <BaseInput v-model="leadEdit.first_name" label="Nome" placeholder="Primeiro nome" />
            <BaseInput v-model="leadEdit.last_name" label="Sobrenome" placeholder="Sobrenome" />
          </div>
          <div class="form-grid-2 mt-half">
            <BaseInput v-model="leadEdit.email" type="email" label="E-mail" placeholder="cliente@email.com" />
            <BaseInput v-model="leadEdit.phone" label="Telefone / WhatsApp" placeholder="(11) 99999-9999" />
          </div>
        </div>

        <CrmLeadJourneyTab
          :tracked-user="trackedUser"
          :tags="leadEdit.tags"
          :all-tags="allTags"
          :can-manage-tags="canDeleteLeads"
          :selected-tag-to-add="selectedTagToAdd"
          @update:selected-tag-to-add="emit('update:selectedTagToAdd', $event)"
          @create-tag="emit('create-tag')"
          @add-tag="emit('add-tag')"
          @remove-tag="emit('remove-tag', $event)"
          @delete-account-tag="emit('delete-account-tag', $event)"
        />

        <div class="lead-main-actions">
          <BaseButton
            v-if="canDeleteLeads"
            variant="ghost"
            size="sm"
            class="delete-lead-btn"
            @click="emit('delete-lead')"
          >
            Excluir este Lead
          </BaseButton>
          <div v-else></div>
          <BaseButton variant="primary" :loading="isSavingLead" @click="emit('save-lead')">
            Salvar Alterações
          </BaseButton>
        </div>
      </div>

      <CrmLeadRightPanel
        :lead-id="leadId"
        :tracked-user-id="trackedUser?.id"
        :notes="leadEdit.notes"
        :new-note-content="newNoteContent"
        :is-adding-note="isAddingNote"
        @update:new-note-content="emit('update:newNoteContent', $event)"
        @add-note="emit('add-note')"
        @delete-note="emit('delete-note', $event)"
      />
    </div>
  </BaseModal>
</template>

<script setup lang="ts">
import { MessageCircle, Compass } from 'lucide-vue-next'
import type { CrmLeadEditState, CrmStage, CrmTag, TrackedUser } from '#shared/types'
import CrmLeadJourneyTab from '~/components/crm/CrmLeadJourneyTab.vue'
import CrmLeadRightPanel from '~/components/crm/CrmLeadRightPanel.vue'
import { useCrmHelpers } from '~/composables/useCrmHelpers'

withDefaults(defineProps<{
  modelValue: boolean
  leadId: number | null
  leadEdit: CrmLeadEditState
  trackedUser: TrackedUser | null
  stages: CrmStage[]
  allTags: CrmTag[]
  selectedTagToAdd: string
  newNoteContent: string
  isSavingLead: boolean
  isAddingNote: boolean
  canDeleteLeads?: boolean
}>(), {
  canDeleteLeads: true,
})

const emit = defineEmits<{
  (e: 'update:modelValue', val: boolean): void
  (e: 'update:selectedTagToAdd', val: string): void
  (e: 'update:newNoteContent', val: string): void
  (e: 'open-journey'): void
  (e: 'create-tag'): void
  (e: 'add-tag'): void
  (e: 'remove-tag', tagId: number): void
  (e: 'delete-account-tag', tagId: number): void
  (e: 'delete-lead'): void
  (e: 'save-lead'): void
  (e: 'add-note'): void
  (e: 'delete-note', noteId: number): void
}>()

const { brandName } = useBranding()
const { getWhatsAppLink } = useCrmHelpers()
</script>

<style scoped>
.lead-modal-header-custom {
  display: flex; justify-content: space-between; align-items: center;
  width: 100%; flex-wrap: wrap; gap: 0.75rem;
}
.lead-modal-identity { display: flex; align-items: center; gap: 0.75rem; }
.lead-modal-avatar {
  width: 40px; height: 40px; border-radius: 8px;
  background: var(--brand-soft); color: var(--brand);
  display: flex; align-items: center; justify-content: center;
  font-size: var(--fs-lg); font-weight: var(--fw-bold);
}
.lead-modal-title { font-size: var(--fs-xl); font-weight: var(--fw-bold); color: var(--text-strong); margin: 0; }
.lead-modal-sub {
  font-size: var(--fs-sm); color: var(--text-3);
  display: flex; align-items: center; gap: 0.35rem; margin-top: 2px;
}
.lead-modal-quick-actions { display: flex; align-items: center; gap: 0.5rem; }
.btn-wpp-lead-modal {
  display: inline-flex; align-items: center; gap: 0.375rem;
  background: var(--success-soft); color: var(--success);
  border: 1px solid color-mix(in srgb, var(--success) 30%, transparent);
  padding: 0.35rem 0.75rem;
  border-radius: 6px; font-size: var(--fs-sm); font-weight: var(--fw-semibold);
  text-decoration: none; transition: all 0.15s;
}
.btn-wpp-lead-modal:hover { background: var(--success); color: var(--brand-text); border-color: var(--success); }
.journey-btn { font-size: var(--fs-sm) !important; color: var(--brand) !important; border-color: var(--brand-border) !important; }
.journey-icon { margin-right: 4px; }
.lead-modal-body-grid { display: grid; grid-template-columns: 1.15fr 1.35fr; gap: 1.5rem; }
.lead-col-main { display: flex; flex-direction: column; }
.info-card-box {
  background: var(--surface-deep); border: 1px solid var(--border);
  border-radius: 8px; padding: 0.875rem; margin-bottom: 0.875rem;
}
.box-subtitle {
  font-size: var(--fs-base); font-weight: var(--fw-bold); color: var(--text-1);
  margin: 0 0 0.625rem 0; text-transform: uppercase; letter-spacing: 0.03em;
}
.form-grid-2 { display: grid; grid-template-columns: 1fr 1fr; gap: 1rem; }
.mt-half { margin-top: 0.625rem; }
.lead-main-actions {
  display: flex; justify-content: space-between; align-items: center;
  margin-top: 0.5rem; padding-top: 0.75rem; border-top: 1px solid var(--border-soft);
}
.delete-lead-btn { color: var(--danger) !important; font-size: var(--fs-base) !important; padding: 0 !important; }
@media (max-width: 900px) {
  .lead-modal-body-grid { grid-template-columns: 1fr; }
}
@media (max-width: 640px) {
  .form-grid-2 { grid-template-columns: 1fr; gap: 0.75rem; }
}
</style>
