<template>
  <BaseModal
    :model-value="modelValue"
    :title="editing ? 'Editar conexão WhatsApp' : 'Nova conexão WhatsApp'"
    description="Conversas de anúncios CTWA, links com protocolo do tracker ou leads já identificados são rastreadas automaticamente."
    size="md"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <div class="inst-form">
      <div class="grid-2">
        <BaseInput
          v-model="form.name"
          label="Nome de identificação"
          placeholder="Ex: Comercial 01 — Lançamento"
          :error="nameError"
          required
        />
        <BaseDropdown
          v-model="form.provider"
          label="Provedor da Conexão"
          :options="providerOptions"
          :disabled="Boolean(editing && editing.provider === 'evolution_go')"
        />
      </div>

      <div class="grid-2">
        <BaseDropdown
          v-model="form.pixel_id"
          label="Pixel vinculado (Meta CAPI / Google Ads / TikTok)"
          placeholder="Selecione um Pixel..."
          :options="pixelOptions"
        />
        <BaseDropdown
          v-model="form.currency"
          label="Moeda da Operação (Meta / Ads)"
          :options="META_CURRENCY_OPTIONS"
        />
      </div>

      <div v-if="isExternalProvider" class="provider-box">
        <div class="box-head">
          <p class="box-title">Webhook {{ providerLabel }}</p>
          <span class="filter-pill">Ignora API & Grupos automaticamente</span>
        </div>

        <div v-if="editing?.webhook_url" class="wh-copy-row">
          <div class="wh-url-box">
            <span class="wh-url-label">URL do Webhook (cole no painel da {{ providerLabel }}):</span>
            <code class="wh-url-val">{{ editing.webhook_url }}</code>
          </div>
          <BaseButton variant="outline" size="sm" @click="copyWebhook">
            {{ copiedWh ? 'Copiado!' : 'Copiar Webhook' }}
          </BaseButton>
        </div>

        <p v-else class="box-hint">
          Ao salvar, o sistema gerará automaticamente a <strong>URL exclusiva do Webhook</strong> para você copiar e colar lá na sua {{ providerLabel }}.
        </p>
      </div>

      <div class="crm-box">
        <BaseDropdown
          :model-value="form.ctwa_min_inbound_messages"
          label="Disparo de LeadSubmitted (Anúncios CTWA)"
          :options="ctwaLeadModeOptions"
          hint="Na opção de 3 mensagens, o anúncio já é identificado na 1ª mensagem, mas o LeadSubmitted só vai para a Meta quando o lead responder e der continuidade."
          @update:model-value="form.ctwa_min_inbound_messages = Number($event) === 3 ? 3 : 1"
        />
        <BaseCheckbox
          v-model="form.auto_create_crm_lead"
          label="Criar card automaticamente no CRM quando entrar novo lead rastreado"
        />
        <BaseDropdown
          v-if="form.auto_create_crm_lead && stageOptions.length"
          v-model="form.default_crm_stage_id"
          label="Etapa inicial no Funil CRM"
          placeholder="Primeira etapa do funil (padrão)"
          :options="stageOptions"
        />
      </div>

      <WhatsAppInstanceTriggersBox
        v-model:lead-trigger="form.lead_trigger"
        v-model:purchase-trigger="form.purchase_trigger"
        v-model:custom-event-name="form.custom_event_name"
        v-model:custom-event-trigger="form.custom_event_trigger"
      />

      <BaseCheckbox
        v-if="editing"
        v-model="form.enabled"
        label="Conexão ativa (desmarque para pausar o recebimento de eventos)"
      />
    </div>

    <template #footer>
      <BaseButton variant="ghost" @click="emit('update:modelValue', false)">Cancelar</BaseButton>
      <BaseButton variant="primary" :loading="saving" @click="onSubmit">
        {{ editing ? 'Salvar alterações' : (isExternalProvider ? 'Criar e gerar Webhook' : 'Criar e conectar') }}
      </BaseButton>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import type { Pixel, WhatsAppInstance, WhatsAppInstanceInput, WhatsAppProvider } from '#shared/types'
import { META_CURRENCY_OPTIONS } from '~/utils/metaCurrencies'
import WhatsAppInstanceTriggersBox from '~/components/whatsapp/WhatsAppInstanceTriggersBox.vue'

const props = defineProps<{
  modelValue: boolean
  editing: WhatsAppInstance | null
  pixels: Pixel[]
  stages: Array<{ id: number; name: string }>
  saving: boolean
  hasEvolutionServer?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [val: boolean]
  submit: [payload: WhatsAppInstanceInput]
}>()

const toast = useToast()
const nameError = ref('')
const copiedWh = ref(false)

const form = reactive({
  name: '',
  provider: 'evolution_go' as WhatsAppProvider,
  pixel_id: 0 as number,
  currency: 'BRL',
  ctwa_min_inbound_messages: 1 as number,
  auto_create_crm_lead: true,
  default_crm_stage_id: 0 as number,
  lead_trigger: '',
  purchase_trigger: '',
  custom_event_name: '',
  custom_event_trigger: '',
  enabled: true,
})

const providerOptions = [
  { value: 'evolution_go', label: 'Conexão Direta — QR Code' },
  { value: 'api_premium', label: 'API Premium (Webhook)' },
  { value: 'uazapi', label: 'Uazapi (Webhook)' },
]

const ctwaLeadModeOptions = [
  { value: 1, label: 'Na 1ª mensagem recebida do lead (Imediato)' },
  { value: 3, label: 'Após 3 mensagens recebidas do lead (Engajamento)' },
]

const isExternalProvider = computed(() => form.provider === 'api_premium' || form.provider === 'uazapi')
const providerLabel = computed(() => (form.provider === 'api_premium' ? 'API Premium' : 'Uazapi'))
const pixelOptions = computed(() => props.pixels.map((p) => ({ value: p.id, label: p.name })))
const stageOptions = computed(() => [
  { value: 0, label: 'Primeira etapa do funil (padrão)' },
  ...props.stages.map((s) => ({ value: s.id, label: s.name })),
])

const copyWebhook = async () => {
  if (!props.editing?.webhook_url || typeof window === 'undefined') return
  try {
    await navigator.clipboard.writeText(props.editing.webhook_url)
    copiedWh.value = true
    setTimeout(() => { copiedWh.value = false }, 2200)
    toast.success('Webhook copiado!', `Cole no painel da ${providerLabel.value}.`)
  } catch {
    toast.error('Erro ao copiar', props.editing.webhook_url)
  }
}

watch(
  () => [props.modelValue, props.editing] as const,
  ([open, inst]) => {
    if (!open) return
    nameError.value = ''
    copiedWh.value = false
    if (inst) {
      form.name = inst.name
      form.provider = inst.provider || 'evolution_go'
      form.pixel_id = inst.pixel_id
      form.currency = inst.currency || 'BRL'
      form.ctwa_min_inbound_messages = inst.ctwa_min_inbound_messages === 3 ? 3 : 1
      form.auto_create_crm_lead = inst.auto_create_crm_lead
      form.default_crm_stage_id = inst.default_crm_stage_id || 0
      form.lead_trigger = inst.lead_trigger || ''
      form.purchase_trigger = inst.purchase_trigger || ''
      form.custom_event_name = inst.custom_event_name || ''
      form.custom_event_trigger = inst.custom_event_trigger || ''
      form.enabled = inst.enabled
    } else {
      form.name = ''
      form.provider = props.hasEvolutionServer === false ? 'api_premium' : 'evolution_go'
      form.pixel_id = props.pixels[0]?.id || 0
      form.currency = 'BRL'
      form.ctwa_min_inbound_messages = 1
      form.auto_create_crm_lead = true
      form.default_crm_stage_id = 0
      form.lead_trigger = ''
      form.purchase_trigger = ''
      form.custom_event_name = ''
      form.custom_event_trigger = ''
      form.enabled = true
    }
  },
  { immediate: true },
)

const onSubmit = () => {
  nameError.value = ''
  if (!form.name.trim()) { nameError.value = 'Informe um nome para a conexão'; return }
  if (!form.pixel_id) { nameError.value = 'Selecione um Pixel válido'; return }
  emit('submit', {
    name: form.name.trim(),
    provider: form.provider,
    pixel_id: Number(form.pixel_id),
    currency: form.currency || 'BRL',
    ctwa_min_inbound_messages: Number(form.ctwa_min_inbound_messages) === 3 ? 3 : 1,
    auto_create_crm_lead: form.auto_create_crm_lead,
    default_crm_stage_id: form.default_crm_stage_id > 0 ? Number(form.default_crm_stage_id) : null,
    lead_trigger: form.lead_trigger.trim(),
    purchase_trigger: form.purchase_trigger.trim(),
    custom_event_name: form.custom_event_name.trim(),
    custom_event_trigger: form.custom_event_trigger.trim(),
    enabled: form.enabled,
  })
}
</script>

<style scoped>
.inst-form { display: flex; flex-direction: column; gap: 1rem; }
.crm-box, .provider-box { display: flex; flex-direction: column; gap: 0.75rem; padding: 0.875rem; background: var(--surface-deep); border: 1px solid var(--border-soft); border-radius: 10px; }
.box-head { display: flex; align-items: center; justify-content: space-between; gap: 0.5rem; flex-wrap: wrap; }
.box-title { font-size: var(--fs-md); font-weight: var(--fw-semibold); color: var(--text-1); margin: 0; }
.filter-pill { font-size: var(--fs-2xs); font-weight: var(--fw-semibold); color: var(--success); background: var(--surface-soft); border: 1px solid var(--border-soft); padding: 0.2rem 0.55rem; border-radius: 999px; }
.wh-copy-row { display: flex; align-items: center; justify-content: space-between; gap: 0.75rem; padding: 0.65rem 0.75rem; background: var(--surface-2); border: 1px solid var(--border); border-radius: 8px; }
.wh-url-box { min-width: 0; flex: 1; display: flex; flex-direction: column; gap: 0.15rem; }
.wh-url-label { font-size: var(--fs-2xs); color: var(--text-3); font-weight: var(--fw-medium); }
.wh-url-val { font-family: var(--font-mono); font-size: var(--fs-xs); color: var(--brand); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.box-hint { font-size: var(--fs-sm); color: var(--text-3); margin: -0.2rem 0 0; line-height: 1.45; }
.grid-2 { display: grid; grid-template-columns: 1fr 1fr; gap: 0.75rem; }
@media (max-width: 640px) { .grid-2 { grid-template-columns: 1fr; } }
</style>
