<template>
  <section class="form-section">
    <header class="section-header">
      <h4>3. Testar conexão e Conversions API</h4>
      <p class="section-hint">
        Valide a saúde do token OAuth na Graph API ou dispare um evento de teste para o Gerenciador de Eventos usando seu <code>Test Event Code</code>.
      </p>
    </header>

    <div v-if="!account || !oauthConnected" class="actions-empty">
      Conecte o OAuth no passo 1 para liberar o diagnóstico de conexão e o disparo de teste CAPI.
    </div>

    <div v-else class="test-fields">
      <div class="grid-2">
        <BaseDropdown
          v-model="test.pixel_id"
          label="Modo / Pixel de destino"
          :options="pixelOptions"
        />

        <BaseDropdown
          v-model="test.event_name"
          label="Evento de teste"
          :options="eventOptions"
          :disabled="!test.pixel_id"
        />
      </div>

      <div v-if="test.pixel_id" class="grid-2">
        <BaseInput
          v-model="test.test_event_code"
          label="Test Event Code (Gerenciador de Eventos)"
          placeholder="Ex: TEST12345"
          hint="Copie na aba Eventos de Teste da Meta para não registrar conversão real."
        />
        <BaseInput
          v-model.number="test.value"
          type="number"
          label="Valor simulado (BRL)"
          placeholder="97.00"
        />
      </div>

      <div class="test-actions">
        <BaseButton
          type="button"
          variant="primary"
          size="sm"
          :loading="testLoading"
          @click="onSendTest"
        >
          {{ test.pixel_id ? 'Disparar evento de teste' : 'Testar conexão Graph API' }}
        </BaseButton>
      </div>

      <div v-if="testResult" class="test-result" :class="isResultOk ? 'result-ok' : 'result-fail'">
        <header class="result-header">
          <span class="result-pill">
            {{ testResult.status_code }} {{ isResultOk ? 'OK' : 'Falha' }}
          </span>
          <span class="result-meta">
            {{ testResult.mode === 'capi_event' ? `Pixel ${testResult.pixel_id}` : 'Diagnóstico Graph API' }}
          </span>
        </header>
        <div v-if="testResult.error" class="result-error">
          {{ testResult.error }}
        </div>
        <details v-if="testResult.request_body">
          <summary>Payload enviado</summary>
          <pre>{{ formatJSON(testResult.request_body) }}</pre>
        </details>
        <details v-if="testResult.response" open>
          <summary>Resposta da Meta</summary>
          <pre>{{ formatJSON(testResult.response) }}</pre>
        </details>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import type { MetaAccount, MetaAccountTestResponse, MetaRemotePixel } from '#shared/types'
import { errorMessage } from '~/utils/errorMessage'

const props = defineProps<{
  account?: MetaAccount | null
  oauthConnected: boolean
  adAccountId: string
  remotePixels: MetaRemotePixel[]
}>()

const metaApi = useMetaAccounts()
const toast = useToast()

const test = reactive({
  pixel_id: '',
  event_name: 'Lead',
  test_event_code: '',
  value: 97,
})
const testLoading = ref(false)
const testResult = ref<MetaAccountTestResponse | null>(null)

const pixelOptions = computed(() => {
  const opts = [
    { value: '', label: 'Apenas verificar conexão OAuth e Conta de Anúncios' },
  ]
  for (const px of props.remotePixels) {
    opts.push({
      value: px.id,
      label: `CAPI Pixel: ${px.name} (${px.id})`,
    })
  }
  return opts
})

const eventOptions = [
  { value: 'Lead', label: 'Lead' },
  { value: 'Purchase', label: 'Purchase (Venda)' },
  { value: 'InitiateCheckout', label: 'InitiateCheckout' },
  { value: 'CompleteRegistration', label: 'CompleteRegistration' },
]

watch(
  () => props.remotePixels,
  (list) => {
    if (test.pixel_id && !list.some((p) => p.id === test.pixel_id)) {
      test.pixel_id = list[0]?.id || ''
    } else if (!test.pixel_id && list.length) {
      test.pixel_id = list[0]!.id
    }
  },
  { immediate: true },
)

const isResultOk = computed(() => {
  if (!testResult.value) return false
  return !testResult.value.error && testResult.value.status_code >= 200 && testResult.value.status_code < 300
})

const onSendTest = async () => {
  if (!props.account) return
  testLoading.value = true
  testResult.value = null
  try {
    const res = await metaApi.testEvent(props.account.id, {
      ad_account_id: props.adAccountId || undefined,
      pixel_id: test.pixel_id || undefined,
      event_name: test.event_name,
      test_event_code: test.test_event_code.trim() || undefined,
      value: test.value > 0 ? test.value : undefined,
      currency: 'BRL',
    })
    testResult.value = res
    if (res.error) {
      toast.error('Falha no teste Meta', res.error)
    } else {
      toast.success(
        'Teste concluído',
        res.mode === 'capi_event' ? 'Evento recebido pela Conversions API.' : 'Conexão OAuth e Graph API validadas.',
      )
    }
  } catch (error: unknown) {
    toast.error('Erro no teste', errorMessage(error, 'Não foi possível testar a conta Meta'))
  } finally {
    testLoading.value = false
  }
}

const formatJSON = (raw?: string): string => {
  if (!raw) return ''
  try { return JSON.stringify(JSON.parse(raw), null, 2) } catch { return raw }
}
</script>

<style scoped>
.form-section { display: flex; flex-direction: column; gap: 0.75rem; padding-bottom: 1.25rem; border-bottom: 1px solid var(--border-faint); }
.section-header h4 { font-family: var(--font-heading); font-size: var(--fs-lg); font-weight: var(--fw-semibold); color: var(--text-strong); margin: 0; }
.section-hint { font-size: var(--fs-base); color: var(--text-2); margin: 0.25rem 0 0; line-height: 1.5; }
.actions-empty { font-size: var(--fs-base); color: var(--text-2); background: var(--surface-soft); border: 1px dashed var(--border); border-radius: 8px; padding: 0.875rem; }
.test-fields { display: flex; flex-direction: column; gap: 0.875rem; }
.grid-2 { display: grid; grid-template-columns: 1fr; gap: 0.75rem; }
@media (min-width: 640px) { .grid-2 { grid-template-columns: 1fr 1fr; } }
.test-actions { display: flex; justify-content: flex-end; }
.test-result { display: flex; flex-direction: column; gap: 0.625rem; padding: 0.875rem; border: 1px solid var(--border); border-radius: 8px; background: var(--surface-soft); font-size: var(--fs-base); }
.test-result.result-ok { border-color: var(--brand-border); background: var(--success-soft); }
.test-result.result-fail { border-color: var(--danger-border); background: var(--danger-soft); }
.result-header { display: flex; justify-content: space-between; align-items: center; gap: 0.5rem; flex-wrap: wrap; }
.result-pill { font-weight: var(--fw-semibold); font-size: var(--fs-xs); padding: 0.18rem 0.55rem; border-radius: 999px; }
.test-result.result-ok .result-pill { background: var(--success); color: var(--brand-text); }
.test-result.result-fail .result-pill { background: var(--danger); color: var(--text-on-solid); }
.result-meta { font-size: var(--fs-xs); font-family: var(--font-mono); color: var(--text-2); }
.result-error { color: var(--danger); font-weight: var(--fw-medium); word-break: break-word; }
.test-result summary { cursor: pointer; font-weight: var(--fw-medium); color: var(--text-1); padding: 0.2rem 0; user-select: none; }
.test-result pre { margin-top: 0.45rem; background: var(--surface-deep); padding: 0.75rem; border-radius: 6px; overflow: auto; max-height: 280px; font-size: var(--fs-sm); line-height: 1.4; color: var(--text-1); white-space: pre-wrap; word-break: break-all; }
</style>
