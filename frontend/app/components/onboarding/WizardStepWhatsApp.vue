<template>
  <div class="step-wa">
    <template v-if="phase === 'connect' && createdInstance">
      <WizardWhatsAppConnectPane :instance="createdInstance" @connected="onWhatsAppConnected" />
      <div class="edit-back">
        <BaseButton variant="ghost" size="sm" @click="phase = 'config'">
          Editar gatilhos ou provedor
        </BaseButton>
      </div>
    </template>

    <template v-else>
      <div class="grid-2">
        <BaseInput
          v-model="form.name"
          label="Nome da conexão WhatsApp"
          placeholder="Ex: Comercial — Clínica Prime"
          :error="nameError"
          required
        />
        <BaseDropdown
          v-model="form.provider"
          label="Provedor da Conexão"
          :options="providerOptions"
          :disabled="Boolean(createdInstance && (createdInstance.provider === 'evolution_go' || createdInstance.provider === 'waha'))"
        />
      </div>

      <div class="crm-box">
        <div class="grid-2">
          <BaseDropdown
            :model-value="form.ctwa_min_inbound_messages"
            label="Disparo de LeadSubmitted (CTWA)"
            :options="ctwaLeadOptions"
            @update:model-value="form.ctwa_min_inbound_messages = Number($event) === 3 ? 3 : 1"
          />
          <BaseDropdown
            v-model="form.currency"
            label="Moeda da operação"
            :options="META_CURRENCY_OPTIONS"
          />
        </div>
        <BaseCheckbox
          v-model="form.auto_create_crm_lead"
          label="Criar card automaticamente no CRM quando entrar novo lead rastreado"
        />
        <BaseDropdown
          v-if="form.auto_create_crm_lead && stageOptions.length > 1"
          v-model="form.default_crm_stage_id"
          label="Etapa inicial no Funil CRM"
          :options="stageOptions"
        />
      </div>

      <WhatsAppInstanceTriggersBox
        v-model="triggerRows"
        v-model:average-ticket="averageTicketInput"
        :currency="form.currency"
      />

      <div class="ai-box">
        <BaseCheckbox
          v-model="enableAi"
          label="Ativar IA Comercial do WhatsApp (qualificação e vendas automáticas)"
          :hint="hasOpenAiKey
            ? 'Analisa textos, áudios, imagens e comprovantes a cada 15 min e move cards no CRM.'
            : 'Você poderá adicionar sua chave OpenAI depois em Configurações → Meu perfil.'"
        />
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import type { WhatsAppProvider } from '#shared/types'
import WhatsAppInstanceTriggersBox from '~/components/whatsapp/WhatsAppInstanceTriggersBox.vue'
import WizardWhatsAppConnectPane from '~/components/onboarding/WizardWhatsAppConnectPane.vue'
import { useAuthStore } from '~/stores/useAuthStore'
import { useWhatsAppStore } from '~/stores/useWhatsAppStore'
import { useClientOnboardingWizard } from '~/composables/useClientOnboardingWizard'
import { META_CURRENCY_OPTIONS } from '~/utils/metaCurrencies'
import {
  formatAverageTicketInput,
  parseAverageTicketInput,
  serializeEditableTriggerRows,
  toEditableTriggerRows,
  type EditableWhatsAppTriggerRow,
} from '~/utils/whatsappEventOptions'
import { errorMessage } from '~/utils/errorMessage'

const emit = defineEmits<{ done: [] }>()

const api = useAPI()
const wa = useWhatsApp()
const waStore = useWhatsAppStore()
const authStore = useAuthStore()
const toast = useToast()

const { clientName, createdPixel, createdInstance } = useClientOnboardingWizard()

const phase = ref<'config' | 'connect'>(createdInstance.value ? 'connect' : 'config')
const nameError = ref('')
const saving = ref(false)
const enableAi = ref(false)
const stages = ref<Array<{ id: number; name: string }>>([])
const triggerRows = ref<EditableWhatsAppTriggerRow[]>(toEditableTriggerRows(createdInstance.value))
const averageTicketInput = ref(formatAverageTicketInput(createdInstance.value?.average_ticket))

const hasOpenAiKey = computed(() => authStore.hasOpenAiKey)

const form = reactive({
  name: createdInstance.value?.name || (clientName.value.trim() ? `WhatsApp — ${clientName.value.trim()}` : 'Comercial 01'),
  provider: (createdInstance.value?.provider || 'evolution_go') as WhatsAppProvider,
  currency: createdInstance.value?.currency || 'BRL',
  ctwa_min_inbound_messages: createdInstance.value?.ctwa_min_inbound_messages === 3 ? 3 : 1,
  auto_create_crm_lead: createdInstance.value?.auto_create_crm_lead ?? true,
  default_crm_stage_id: createdInstance.value?.default_crm_stage_id || 0,
})

const providerOptions = [
  { value: 'evolution_go', label: 'Evolution Go — QR Code' },
  { value: 'waha', label: 'WAHA API — QR Code' },
  { value: 'api_premium', label: 'API Premium (Webhook)' },
  { value: 'uazapi', label: 'Uazapi (Webhook)' },
]
const ctwaLeadOptions = [
  { value: 1, label: 'Na 1ª mensagem do lead (Imediato)' },
  { value: 3, label: 'Após 3 mensagens (Engajamento)' },
]
const stageOptions = computed(() => [
  { value: 0, label: 'Primeira etapa do funil (padrão)' },
  ...stages.value.map((s) => ({ value: s.id, label: s.name })),
])

onMounted(async () => {
  await waStore.fetchAll(true).catch(() => {})
  if (!createdInstance.value) {
    if (waStore.server?.has_api_key) form.provider = 'evolution_go'
    else if (waStore.wahaServer?.has_api_key) form.provider = 'waha'
    else form.provider = 'api_premium'
  }
  try {
    const board = await api.get<{ stages?: Array<{ id: number; name: string }> }>('/admin/crm/board')
    stages.value = board?.stages || []
  } catch {
    stages.value = []
  }
})

const onWhatsAppConnected = () => {
  if (createdInstance.value) {
    createdInstance.value = { ...createdInstance.value, connection_status: 'OPEN' }
  }
}

const submit = async () => {
  if (phase.value === 'connect') {
    emit('done')
    return
  }
  nameError.value = ''
  if (!form.name.trim()) {
    nameError.value = 'Informe um nome para a conexão WhatsApp'
    return
  }
  if (!createdPixel.value?.id) {
    toast.error('Pixel obrigatório', 'Salve o Pixel na etapa anterior antes de criar o WhatsApp.')
    return
  }
  const { triggers, missingCustomName } = serializeEditableTriggerRows(triggerRows.value)
  if (missingCustomName) {
    toast.error('Evento personalizado incompleto', 'Digite o nome do evento ou limpe o gatilho correspondente.')
    return
  }

  saving.value = true
  try {
    const payload = {
      name: form.name.trim(),
      provider: form.provider,
      pixel_id: createdPixel.value.id,
      currency: form.currency || 'BRL',
      ctwa_min_inbound_messages: Number(form.ctwa_min_inbound_messages) === 3 ? 3 : 1,
      auto_create_crm_lead: form.auto_create_crm_lead,
      default_crm_stage_id: form.default_crm_stage_id > 0 ? Number(form.default_crm_stage_id) : null,
      event_triggers: triggers,
      average_ticket: parseAverageTicketInput(averageTicketInput.value),
      enabled: true,
    }
    const saved = createdInstance.value
      ? await wa.updateInstance(createdInstance.value.id, payload)
      : await wa.createInstance(payload)
    createdInstance.value = saved

    if (enableAi.value) {
      const curAi = await wa.getAISettings().catch(() => null)
      if (curAi) {
        await wa.updateAISettings({
          enabled: true,
          model: curAi.model || 'gpt-4o-mini',
          base_prompt: curAi.base_prompt,
          qualified_stage_id: curAi.qualified_stage_id,
          sale_stage_id: curAi.sale_stage_id,
        }).catch(() => {})
      }
    }

    toast.success('WhatsApp salvo', 'Agora conecte o aparelho ou copie o link para o cliente.')
    phase.value = 'connect'
  } catch (err: unknown) {
    toast.error('Erro ao salvar WhatsApp', errorMessage(err, 'Não foi possível criar a conexão WhatsApp.'))
  } finally {
    saving.value = false
  }
}

const primaryLabel = computed(() => {
  if (phase.value === 'connect') {
    return createdInstance.value?.connection_status === 'OPEN'
      ? 'Finalizar configuração'
      : 'Concluir (conectar depois)'
  }
  return form.provider === 'evolution_go' || form.provider === 'waha' ? 'Salvar e gerar QR Code' : 'Salvar e gerar Webhook'
})

defineExpose({ submit, saving, primaryLabel })
</script>

<style scoped>
.step-wa { display: flex; flex-direction: column; gap: 1rem; }
.grid-2 { display: grid; grid-template-columns: 1fr 1fr; gap: 0.75rem; }
.crm-box, .ai-box {
  display: flex; flex-direction: column; gap: 0.75rem; padding: 0.85rem;
  background: var(--surface-deep); border: 1px solid var(--border-soft); border-radius: 10px;
}
.edit-back { display: flex; justify-content: center; }
@media (max-width: 640px) { .grid-2 { grid-template-columns: 1fr; } }
</style>
