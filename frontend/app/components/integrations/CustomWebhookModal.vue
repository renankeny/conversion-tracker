<template>
  <BaseModal
    :model-value="modelValue"
    :title="editingItem ? 'Editar Integração' : 'Nova Integração'"
    size="xl"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <div class="form-group">
      <label>Nome da Plataforma</label>
      <input v-model="form.name" type="text" class="base-input" placeholder="Ex: Kiwify, PerfectPay..." />
    </div>

    <div v-if="isOwner" class="form-group" style="margin-top: 0.75rem;">
      <BaseCheckbox
        v-model="form.is_global"
        label="Disponibilizar Globalmente (Todas as Empresas)"
      />
    </div>

    <div class="ai-container" :class="{ 'ai-loading': generatingAI }" style="margin-top: 0.75rem;">
      <div v-if="generatingAI" class="ai-overlay">
        <div class="spinner" />
        <span>A IA está analisando a estrutura do seu Payload...</span>
      </div>

      <div class="form-group">
        <label>Cole um Payload de Exemplo (JSON)</label>
        <textarea v-model="exampleJson" class="json-textarea" placeholder="Cole o JSON de teste de webhook da plataforma aqui..." />
      </div>

      <div v-if="exampleJson && !showAdvanced && !aiSuccess" class="ai-actions">
        <BaseButton v-if="hasApiKey" size="lg" variant="primary" class="full-width-btn pulse-btn" :loading="generatingAI" @click="generateMappingWithAI">
          ✨ Mapear e Integrar Automaticamente
        </BaseButton>
        <BaseButton v-else size="lg" variant="outline" class="full-width-btn" @click="router.push('/settings')">
          Configurar Chave da OpenAI para Automapeamento
        </BaseButton>
      </div>

      <div v-if="aiSuccess && !showAdvanced" class="ai-success">
        <div class="success-icon">✅</div>
        <h3>Mapeamento Concluído com Sucesso!</h3>
        <p>A Inteligência Artificial já preencheu tudo por debaixo dos panos.</p>
      </div>
    </div>

    <div class="advanced-toggle">
      <BaseButton size="sm" variant="ghost" @click="showAdvanced = !showAdvanced">
        {{ showAdvanced ? 'Esconder Mapeamento Manual' : 'Modo Avançado (Mapeamento Manual)' }}
      </BaseButton>
    </div>

    <CustomWebhookMappingForm
      v-if="showAdvanced"
      :mapping-config="form.mapping_config"
      :example-json="exampleJson"
    />

    <template #footer>
      <BaseButton variant="ghost" @click="$emit('update:modelValue', false)">Cancelar</BaseButton>
      <BaseButton variant="primary" :loading="saving" @click="onSave">Salvar Integração</BaseButton>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import CustomWebhookMappingForm from '~/components/integrations/CustomWebhookMappingForm.vue'
import { useAuthStore } from '~/stores/useAuthStore'
import type { CustomMappingConfig, CustomWebhook } from '#shared/types'
import { errorMessage } from '~/utils/errorMessage'

const props = defineProps<{
  modelValue: boolean
  editingItem?: CustomWebhook | null
  saving: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  save: [payload: { id: string | null; name: string; is_global: boolean; mapping_config: CustomMappingConfig }]
}>()

const auth = useAuthStore()
const toast = useToast()
const router = useRouter()

const hasApiKey = computed(() => auth.hasOpenAiKey)
const isOwner = computed(() => auth.isOwner)

const defaultMapping = (): CustomMappingConfig => ({
  first_name: '', last_name: '', email: '', phone: '', transaction_id: '', amount: '', status: '',
  utm_source: '', utm_medium: '', utm_campaign: '', pixel: '',
  status_map: { paid: 'Purchase', approved: 'Purchase', waiting_payment: 'InitiateCheckout' },
})

const form = ref({
  name: '',
  is_global: false,
  mapping_config: defaultMapping(),
})
const exampleJson = ref('')
const showAdvanced = ref(false)
const generatingAI = ref(false)
const aiSuccess = ref(false)

watch(
  () => [props.modelValue, props.editingItem] as const,
  ([open, item]) => {
    if (!open) return
    showAdvanced.value = !!item
    aiSuccess.value = false
    if (item) {
      form.value = {
        name: item.name,
        is_global: item.account_id === null,
        mapping_config: item.mapping_config ? { ...defaultMapping(), ...item.mapping_config } : defaultMapping(),
      }
      exampleJson.value = item.mapping_config?._example_payload || ''
    } else {
      form.value = { name: '', is_global: false, mapping_config: defaultMapping() }
      exampleJson.value = ''
    }
  },
  { immediate: true }
)

const generateMappingWithAI = async () => {
  if (!exampleJson.value) return
  generatingAI.value = true
  try {
    const api = useAPI()
    const data = await api.post<{ mapping: Partial<CustomMappingConfig> }>('/admin/openai/webhook-mapping', {
      payload: exampleJson.value,
    })
    const aiResult = data.mapping || {}
    form.value.mapping_config = {
      ...defaultMapping(),
      ...aiResult,
      status_map: aiResult.status_map || {},
    }
    aiSuccess.value = true
    toast.success('Sucesso', 'JSON Mapeado pela Inteligência Artificial!')
  } catch (error: unknown) {
    toast.error('Erro ao gerar com IA', errorMessage(error, 'Falha ao mapear payload'))
  } finally {
    generatingAI.value = false
  }
}

const onSave = () => {
  const payloadToSave: CustomMappingConfig = { ...form.value.mapping_config }
  if (exampleJson.value) payloadToSave._example_payload = exampleJson.value
  else delete payloadToSave._example_payload

  emit('save', {
    id: props.editingItem ? String(props.editingItem.id) : null,
    name: form.value.name,
    is_global: form.value.is_global,
    mapping_config: payloadToSave,
  })
}
</script>

<style scoped>
.json-textarea { width: 100%; height: 200px; background: var(--surface-2); color: var(--text-1); border: 1px solid var(--border); border-radius: 8px; padding: 1rem; font-family: var(--font-mono); resize: vertical; }
.base-input { width: 100%; padding: 0.5rem; border-radius: 6px; border: 1px solid var(--border); background: var(--surface-2); color: var(--text-1); }
.form-group { display: flex; flex-direction: column; gap: 0.25rem; }
.form-group label { font-size: var(--fs-md); font-weight: var(--fw-medium); }
.ai-container { position: relative; margin-bottom: 1rem; }
.ai-loading .json-textarea { filter: blur(2px); pointer-events: none; }
.ai-overlay { position: absolute; inset: 0; background: var(--surface-overlay); border-radius: 8px; z-index: 10; display: flex; flex-direction: column; align-items: center; justify-content: center; color: var(--brand); font-weight: var(--fw-bold); gap: 1rem; }
.spinner { width: 40px; height: 40px; border: 4px solid var(--border); border-left-color: var(--brand); border-radius: 50%; animation: spin 1s linear infinite; }
@keyframes spin { 100% { transform: rotate(360deg); } }
.ai-success { display: flex; flex-direction: column; align-items: center; justify-content: center; padding: 2rem; background: var(--success-soft); border: 1px solid var(--success); border-radius: 8px; margin-top: 1rem; text-align: center; }
.ai-success .success-icon { font-size: var(--fs-3xl); margin-bottom: 0.5rem; }
.ai-success h3 { color: var(--success); margin-bottom: 0.25rem; font-size: var(--fs-2xl); }
.ai-success p { color: var(--text-2); font-size: var(--fs-md); }
.full-width-btn { width: 100%; font-size: var(--fs-xl); padding: 1rem; margin-top: 1rem; }
.pulse-btn { animation: pulse 2s infinite; background: linear-gradient(45deg, var(--brand), var(--accent-purple-strong)); border: none; }
@keyframes pulse {
  0% { box-shadow: 0 0 0 0 var(--accent-purple-soft); }
  70% { box-shadow: 0 0 0 10px transparent; }
  100% { box-shadow: 0 0 0 0 transparent; }
}
.advanced-toggle { display: flex; justify-content: flex-end; margin-bottom: 1rem; }
</style>
