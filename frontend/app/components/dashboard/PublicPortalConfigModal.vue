<template>
  <BaseModal
    v-model="isOpen"
    title="Portal Público do Cliente"
    description="Defina quais abas e permissões o seu cliente terá ao acessar o link compartilhável deste pixel."
    size="md"
  >
    <div class="portal-cfg-body">
      <div v-if="shareUrl" class="share-url-box">
        <span class="box-label">Link exclusivo do cliente</span>
        <div class="url-row">
          <code class="url-code">{{ shareUrl }}</code>
          <BaseButton variant="outline" size="sm" @click="$emit('copy')">
            <Check v-if="copied" :size="14" />
            <Copy v-else :size="14" />
            <span>{{ copied ? 'Copiado' : 'Copiar' }}</span>
          </BaseButton>
        </div>
      </div>

      <div class="options-stack">
        <div class="option-card">
          <BaseCheckbox
            :model-value="true"
            disabled
            label="Exibir aba Dashboard (sempre ativa)"
          />
          <p class="option-hint">
            Métricas executivas, criativos, palavras-chave, links rastreáveis, funil e vendas recentes.
          </p>
        </div>

        <div class="option-card">
          <BaseCheckbox
            v-model="showSalesGoal"
            label="Exibir barra de Meta de Vendas na Dashboard do cliente"
          />
          <p class="option-hint">
            Quando o pixel possui uma meta ativa na aba Geral, mostra a barra de porcentagem do 0% até atingir a meta para o cliente acompanhar.
          </p>
        </div>

        <div class="option-card">
          <BaseCheckbox
            v-model="showCrm"
            label="Exibir aba CRM na sidebar do cliente"
          />
          <p class="option-hint">
            Mostra o quadro Kanban filtrado estritamente pelos leads deste pixel.
          </p>

          <div v-if="showCrm" class="nested-option">
            <BaseCheckbox
              v-model="allowCrmEdit"
              label="Permitir que o comercial do cliente opere o CRM pelo link"
            />
            <p class="option-hint">
              Quando desmarcado, o CRM fica 100% somente visualização. Quando marcado, o time do cliente pode mover cards, confirmar vendas, adicionar anotações e tags (disparando os eventos das etapas automaticamente), sem poder alterar ou excluir etapas.
            </p>
          </div>
        </div>

        <div class="option-card">
          <BaseCheckbox
            v-model="showConversations"
            label="Exibir aba Conversas na sidebar do cliente"
          />
          <p class="option-hint">
            Auditoria somente leitura de conversas rastreadas do WhatsApp, origem do tráfego e tempos de resposta do comercial (SLA).
          </p>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="modal-footer-row">
        <BaseButton
          v-if="shareUrl"
          variant="ghost"
          size="sm"
          class="revoke-btn"
          :loading="revoking"
          @click="$emit('revoke')"
        >
          Desativar link
        </BaseButton>
        <div v-else />
        <div class="footer-right">
          <BaseButton variant="ghost" size="sm" @click="isOpen = false">Cancelar</BaseButton>
          <BaseButton variant="primary" size="sm" :loading="saving" @click="onSave">Salvar permissões</BaseButton>
        </div>
      </div>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import { Copy, Check } from 'lucide-vue-next'
import type { DashboardConfig } from '#shared/types'

const props = defineProps<{
  modelValue: boolean
  shareUrl: string
  config?: DashboardConfig | null
  saving?: boolean
  revoking?: boolean
  copied?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [val: boolean]
  save: [patch: Pick<DashboardConfig, 'public_show_crm' | 'public_allow_crm_edit' | 'public_show_conversations' | 'public_show_sales_goal'>]
  copy: []
  revoke: []
}>()

const isOpen = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v),
})

const showCrm = ref(true)
const allowCrmEdit = ref(false)
const showConversations = ref(true)
const showSalesGoal = ref(true)

watch(() => props.modelValue, (open) => {
  if (!open) return
  showCrm.value = props.config?.public_show_crm !== false
  allowCrmEdit.value = Boolean(props.config?.public_allow_crm_edit)
  showConversations.value = props.config?.public_show_conversations !== false
  showSalesGoal.value = props.config?.public_show_sales_goal !== false
}, { immediate: true })

const onSave = () => {
  emit('save', {
    public_show_crm: showCrm.value,
    public_allow_crm_edit: showCrm.value ? allowCrmEdit.value : false,
    public_show_conversations: showConversations.value,
    public_show_sales_goal: showSalesGoal.value,
  })
}
</script>

<style scoped>
.portal-cfg-body { display: flex; flex-direction: column; gap: 1rem; }
.share-url-box {
  background: var(--surface-deep); border: 1px solid var(--border-soft);
  border-radius: 10px; padding: 0.75rem 0.875rem; display: flex; flex-direction: column; gap: 0.4rem;
}
.box-label { font-size: var(--fs-xs); font-weight: var(--fw-semibold); color: var(--text-3); text-transform: uppercase; letter-spacing: 0.04em; }
.url-row { display: flex; align-items: center; gap: 0.5rem; flex-wrap: wrap; }
.url-code {
  flex: 1; min-width: 0; font-family: var(--font-mono); font-size: var(--fs-sm);
  color: var(--text-1); background: var(--surface-2); border: 1px solid var(--border);
  border-radius: 6px; padding: 0.375rem 0.625rem; overflow-wrap: anywhere;
}
.options-stack { display: flex; flex-direction: column; gap: 0.625rem; }
.option-card {
  background: var(--surface-deep); border: 1px solid var(--border-soft);
  border-radius: 10px; padding: 0.75rem 0.875rem; display: flex; flex-direction: column; gap: 0.35rem;
}
.option-hint { font-size: var(--fs-sm); color: var(--text-3); margin: 0 0 0 1.65rem; line-height: 1.45; }
.nested-option {
  margin: 0.5rem 0 0 1.65rem; padding-top: 0.625rem;
  border-top: 1px solid var(--border-soft); display: flex; flex-direction: column; gap: 0.35rem;
}
.modal-footer-row { display: flex; align-items: center; justify-content: space-between; width: 100%; gap: 0.75rem; }
.footer-right { display: flex; align-items: center; gap: 0.5rem; }
.revoke-btn { color: var(--danger) !important; }
</style>
