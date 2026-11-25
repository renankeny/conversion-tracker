<template>
  <section class="form-section">
    <header class="section-header">
      <h4>4. Testar envio de conversão</h4>
      <p class="section-hint">
        Dispara uma conversão síncrona para o Google Ads para verificar se a conexão e o Action ID estão corretos. Por padrão usa <code>validateOnly=true</code>: o Google valida autenticação e payload sem registrar conversão real.
      </p>
    </header>

    <div v-if="!canTest" class="actions-empty">
      Conecte o OAuth, selecione a conta Google Ads e ative ao menos uma Ação de Conversão acima para liberar o teste.
    </div>

    <div v-else class="test-fields">
      <BaseDropdown
        v-model="test.action_id"
        label="Ação de Conversão a testar"
        :options="testActionOptions"
      />

      <div class="grid-2">
        <BaseInput
          v-model="test.gclid"
          label="GCLID (opcional)"
          placeholder="Cj0KCQiA..."
          hint="Se vazio em validate_only, o sistema gera um GCLID sintético para validação."
        />
        <BaseInput
          v-model.number="test.value"
          type="number"
          label="Valor da conversão (BRL)"
          placeholder="1.00"
        />
      </div>

      <BaseCheckbox v-model="test.validate_only">
        <strong>Modo validação (`validate_only`):</strong> valida autenticação e estrutura na API do Google sem gravar conversão nas métricas.
      </BaseCheckbox>

      <div class="test-actions">
        <BaseButton
          type="button"
          variant="primary"
          size="sm"
          :loading="testLoading"
          :disabled="!testActionOptions.length"
          @click="onSendTest"
        >
          Enviar teste
        </BaseButton>
      </div>

      <div v-if="testResult" class="test-result" :class="testResultClass">
        <header class="result-header">
          <span class="result-pill">
            {{ testResult.status_code }} {{ testResultLabel }}
          </span>
          <span class="result-action">
            action_id: <code>{{ testResult.conversion_action_id }}</code>
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
          <summary>Resposta do Google</summary>
          <pre>{{ formatJSON(testResult.response) }}</pre>
        </details>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import type { GoogleAdsAccount, GoogleAdsConversionAction } from '#shared/types'
import { errorMessage } from '~/utils/errorMessage'

const props = defineProps<{
  account?: GoogleAdsAccount | null
  oauthConnected: boolean
  customerId: string
  loginCustomerId: string
  actions: GoogleAdsConversionAction[]
}>()

const googleAdsApi = useGoogleAdsAccounts()
const toast = useToast()

const test = reactive({
  action_id: '',
  gclid: '',
  value: 1,
  validate_only: true,
})
const testLoading = ref(false)
const testResult = ref<{
  status_code: number
  conversion_action_id: string
  request_body: string
  response: string
  error?: string
} | null>(null)

const activeActions = computed(() =>
  props.actions.filter((ca) => ca.enabled && ca.conversion_action_id.trim()),
)

const canTest = computed(() =>
  Boolean(
    props.account?.id
    && props.oauthConnected
    && props.customerId.replace(/\D/g, '')
    && activeActions.value.length,
  ),
)

const testActionOptions = computed(() => {
  const seen = new Set<string>()
  const out: Array<{ value: string; label: string }> = []
  for (const ca of activeActions.value) {
    if (seen.has(ca.conversion_action_id)) continue
    seen.add(ca.conversion_action_id)
    out.push({
      value: ca.conversion_action_id,
      label: ca.name
        ? `${ca.event_type}: ${ca.name} (ID ${ca.conversion_action_id})`
        : `${ca.event_type} (ID ${ca.conversion_action_id})`,
    })
  }
  return out
})

watchEffect(() => {
  if (!testActionOptions.value.some((o) => o.value === test.action_id)) {
    test.action_id = testActionOptions.value[0]?.value || ''
  }
})

const testResultClass = computed(() => {
  if (!testResult.value) return ''
  if (testResult.value.error) return 'result-fail'
  return testResult.value.status_code >= 200 && testResult.value.status_code < 300 ? 'result-ok' : 'result-fail'
})

const testResultLabel = computed(() => {
  if (!testResult.value) return ''
  if (testResult.value.error) return 'Erro'
  return testResult.value.status_code >= 200 && testResult.value.status_code < 300 ? 'OK' : 'Falha'
})

const onSendTest = async () => {
  if (!props.account?.id) return
  const chosen = activeActions.value.find((a) => a.conversion_action_id === test.action_id)
  if (!chosen) return

  testLoading.value = true
  testResult.value = null
  try {
    const res = await googleAdsApi.testEvent(props.account.id, {
      event_type: chosen.event_type,
      conversion_action_id: chosen.conversion_action_id,
      action_type: chosen.action_type || undefined,
      owner_customer_id: chosen.owner_customer_id || undefined,
      customer_id: props.customerId || undefined,
      login_customer_id: props.loginCustomerId || undefined,
      gclid: test.gclid.trim() || undefined,
      value: test.value > 0 ? test.value : undefined,
      validate_only: test.validate_only,
    })
    const first = res.results?.[0]
    if (!first) {
      toast.error('Erro', 'Sem resposta da API do Google Ads.')
      return
    }
    testResult.value = {
      status_code: first.status_code,
      conversion_action_id: res.conversion_action_id,
      request_body: first.request_body,
      response: first.response,
      error: first.error,
    }
    if (first.error) {
      toast.error('Falha no teste', first.error)
    } else {
      toast.success('Sucesso', test.validate_only ? 'Payload validado pelo Google Ads.' : 'Conversão enviada.')
    }
  } catch (error: unknown) {
    toast.error('Erro', errorMessage(error, 'Falha ao testar envio Google Ads'))
  } finally {
    testLoading.value = false
  }
}

const formatJSON = (raw: string): string => {
  if (!raw) return ''
  try { return JSON.stringify(JSON.parse(raw), null, 2) } catch { return raw }
}
</script>

<style scoped>
.form-section { display: flex; flex-direction: column; gap: 0.75rem; padding-bottom: 1.25rem; border-bottom: 1px solid var(--border-faint); }
.section-header h4 { font-family: var(--font-heading); font-size: var(--fs-lg); font-weight: var(--fw-semibold); color: var(--text-strong); margin: 0; }
.section-hint { font-size: var(--fs-base); color: var(--text-2); margin: 0.25rem 0 0; line-height: 1.5; }
.actions-empty { font-size: var(--fs-base); color: var(--text-2); background: var(--surface-soft); border: 1px dashed var(--border); border-radius: 8px; padding: 0.875rem; }
.grid-2 { display: grid; grid-template-columns: 1fr; gap: 0.75rem; }
@media (min-width: 640px) { .grid-2 { grid-template-columns: 1fr 1fr; } }
.test-fields { display: flex; flex-direction: column; gap: 0.875rem; }
.test-actions { display: flex; justify-content: flex-end; }
.test-result { display: flex; flex-direction: column; gap: 0.625rem; padding: 0.875rem; border: 1px solid var(--border); border-radius: 8px; background: var(--surface-soft); font-size: var(--fs-base); }
.test-result.result-ok { border-color: var(--brand-border); background: var(--success-soft); }
.test-result.result-fail { border-color: var(--danger-border); background: var(--danger-soft); }
.result-header { display: flex; justify-content: space-between; align-items: center; gap: 0.5rem; flex-wrap: wrap; }
.result-pill { font-weight: var(--fw-semibold); font-size: var(--fs-xs); padding: 0.18rem 0.55rem; border-radius: 999px; }
.test-result.result-ok .result-pill { background: var(--success); color: var(--brand-text); }
.test-result.result-fail .result-pill { background: var(--danger); color: var(--text-on-solid); }
.result-action { font-size: var(--fs-xs); color: var(--text-2); font-family: var(--font-mono); }
.result-action code { background: var(--surface-2); padding: 0.125rem 0.4rem; border-radius: 4px; }
.result-error { color: var(--danger); font-weight: var(--fw-medium); word-break: break-word; }
.test-result summary { cursor: pointer; font-weight: var(--fw-medium); color: var(--text-1); padding: 0.25rem 0; user-select: none; }
.test-result pre { margin-top: 0.5rem; background: var(--surface-deep); padding: 0.75rem; border-radius: 6px; overflow: auto; max-height: 300px; font-size: var(--fs-sm); line-height: 1.4; color: var(--text-1); white-space: pre-wrap; word-break: break-all; }
</style>
