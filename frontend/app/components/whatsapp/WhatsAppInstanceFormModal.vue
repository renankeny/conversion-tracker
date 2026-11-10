<template>
  <BaseModal
    :model-value="modelValue"
    :title="editing ? 'Editar conexão WhatsApp' : 'Nova conexão WhatsApp'"
    description="Apenas conversas originadas por anúncios CTWA ou links com protocolo do tracker são rastreadas."
    size="md"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <div class="inst-form">
      <BaseInput
        v-model="form.name"
        label="Nome de identificação"
        placeholder="Ex: Comercial 01 — Lançamento"
        :error="nameError"
        required
      />

      <BaseDropdown
        v-model="form.pixel_id"
        label="Pixel vinculado (Meta CAPI / Google Ads)"
        placeholder="Selecione um Pixel..."
        :options="pixelOptions"
        hint="Conversões de CTWA e CRM desta conexão serão enviadas para este Pixel."
      />

      <div class="crm-box">
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

      <div class="triggers-box">
        <p class="box-title">Gatilhos opcionais por mensagem na conversa</p>
        <p class="box-hint">
          Se o vendedor enviar uma mensagem contendo o termo exato abaixo, o evento será disparado automaticamente.
        </p>
        <div class="grid-2">
          <BaseInput
            v-model="form.lead_trigger"
            label="Termo para Lead Qualificado"
            placeholder="Ex: proposta enviada"
          />
          <BaseInput
            v-model="form.purchase_trigger"
            label="Termo para Venda (Purchase)"
            placeholder="Ex: pagamento confirmado"
          />
        </div>
      </div>

      <BaseCheckbox
        v-if="editing"
        v-model="form.enabled"
        label="Conexão ativa (desmarque para pausar o recebimento de eventos)"
      />
    </div>

    <template #footer>
      <BaseButton variant="ghost" @click="emit('update:modelValue', false)">
        Cancelar
      </BaseButton>
      <BaseButton variant="primary" :loading="saving" @click="onSubmit">
        {{ editing ? 'Salvar alterações' : 'Criar e conectar' }}
      </BaseButton>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import type { Pixel, WhatsAppInstance, WhatsAppInstanceInput } from '#shared/types'

const props = defineProps<{
  modelValue: boolean
  editing: WhatsAppInstance | null
  pixels: Pixel[]
  stages: Array<{ id: number; name: string }>
  saving: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [val: boolean]
  submit: [payload: WhatsAppInstanceInput]
}>()

const nameError = ref('')
const form = reactive({
  name: '',
  pixel_id: 0 as number,
  auto_create_crm_lead: true,
  default_crm_stage_id: 0 as number,
  lead_trigger: '',
  purchase_trigger: '',
  enabled: true,
})

const pixelOptions = computed(() =>
  props.pixels.map((p) => ({ value: p.id, label: p.name })),
)

const stageOptions = computed(() => [
  { value: 0, label: 'Primeira etapa do funil (padrão)' },
  ...props.stages.map((s) => ({ value: s.id, label: s.name })),
])

watch(
  () => [props.modelValue, props.editing] as const,
  ([open, inst]) => {
    if (!open) return
    nameError.value = ''
    if (inst) {
      form.name = inst.name
      form.pixel_id = inst.pixel_id
      form.auto_create_crm_lead = inst.auto_create_crm_lead
      form.default_crm_stage_id = inst.default_crm_stage_id || 0
      form.lead_trigger = inst.lead_trigger || ''
      form.purchase_trigger = inst.purchase_trigger || ''
      form.enabled = inst.enabled
    } else {
      form.name = ''
      form.pixel_id = props.pixels[0]?.id || 0
      form.auto_create_crm_lead = true
      form.default_crm_stage_id = 0
      form.lead_trigger = ''
      form.purchase_trigger = ''
      form.enabled = true
    }
  },
  { immediate: true },
)

const onSubmit = () => {
  nameError.value = ''
  if (!form.name.trim()) {
    nameError.value = 'Informe um nome para a conexão'
    return
  }
  if (!form.pixel_id) {
    nameError.value = 'Selecione um Pixel válido'
    return
  }
  emit('submit', {
    name: form.name.trim(),
    pixel_id: Number(form.pixel_id),
    auto_create_crm_lead: form.auto_create_crm_lead,
    default_crm_stage_id: form.default_crm_stage_id > 0 ? Number(form.default_crm_stage_id) : null,
    lead_trigger: form.lead_trigger.trim(),
    purchase_trigger: form.purchase_trigger.trim(),
    enabled: form.enabled,
  })
}
</script>

<style scoped>
.inst-form { display: flex; flex-direction: column; gap: 1.125rem; }
.crm-box, .triggers-box {
  display: flex; flex-direction: column; gap: 0.75rem;
  padding: 0.875rem; background: var(--surface-deep);
  border: 1px solid var(--border-soft); border-radius: 10px;
}
.box-title { font-size: var(--fs-md); font-weight: var(--fw-semibold); color: var(--text-1); margin: 0; }
.box-hint { font-size: var(--fs-sm); color: var(--text-3); margin: -0.35rem 0 0; line-height: 1.45; }
.grid-2 { display: grid; grid-template-columns: 1fr 1fr; gap: 0.75rem; }
@media (max-width: 640px) { .grid-2 { grid-template-columns: 1fr; } }
</style>

