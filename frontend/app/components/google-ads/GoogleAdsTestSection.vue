<template>
  <section class="form-section">
    <header class="section-header">
      <h4>Testar envio</h4>
      <p class="section-hint">
        Manda uma conversao sincrona pro Google Ads pra checar se a
        conexao + Action ID estao corretos. Por padrao usa
        <code>validateOnly=true</code> &mdash; o Google valida auth e
        payload mas NAO registra a conversao.
      </p>
    </header>

    <div v-if="!account.connected" class="actions-empty">
      Conecte a conta via OAuth e cadastre ao menos uma Conversion Action
      ativa pra liberar o teste.
    </div>

    <div v-else class="test-fields">
      <BaseDropdown
        v-model="test.event_type"
        label="Conversion Action a disparar"
        :options="testActionOptions"
      />

      <div class="grid-2">
        <BaseInput
          v-model="test.gclid"
          label="GCLID (opcional)"
          placeholder="Cj0KCQiA..."
          hint="Deixe vazio em modo validate-only — o backend gera um GCLID sintetico."
        />
        <BaseInput
          v-model.number="test.value"
          type="number"
          label="Valor da conversao"
          placeholder="1.00"
        />
      </div>

      <BaseCheckbox v-model="test.validate_only">
        <strong>validate_only</strong> — quando ligado, o Google checa
        auth + payload sem gravar conversão. Desligue só depois de
        confirmar que tudo passa.
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
          <summary>Request enviado</summary>
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
  account: GoogleAdsAccount
  actions: GoogleAdsConversionAction[]
}>()

const googleAdsApi = useGoogleAdsAccounts()
const toast = useToast()

const test = reactive({
  event_type: 'Purchase',
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

const testActionOptions = computed(() =>
  props.actions
    .filter((ca) => ca.enabled && ca.conversion_action_id)
    .map((ca) => ({
      value: ca.event_type,
      label: ca.name ? `${ca.event_type} — ${ca.name}` : ca.event_type,
    }))
)

watchEffect(() => {
  if (!testActionOptions.value.find((o) => o.value === test.event_type)) {
    test.event_type = testActionOptions.value[0]?.value || ''
  }
})

const testResultClass = computed(() => {
  if (!testResult.value) return ''
  if (testResult.value.error) return 'result-fail'
  return testResult.value.status_code >= 200 && testResult.value.status_code < 300 ? 'result-ok' : 'result-fail'
})

const testResultLabel = computed(() => {
  if (!testResult.value) return ''
  if (testResult.value.error) return 'erro'
  return testResult.value.status_code >= 200 && testResult.value.status_code < 300 ? 'OK' : 'falha'
})

const onSendTest = async () => {
  testLoading.value = true
  testResult.value = null
  try {
    const res = await googleAdsApi.testEvent(props.account.id, {
      event_type: test.event_type,
      gclid: test.gclid.trim() || undefined,
      value: test.value > 0 ? test.value : undefined,
      validate_only: test.validate_only,
    })
    const first = res.results?.[0]
    if (!first) {
      toast.error('Erro', 'Sem resposta do Google.')
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
      toast.success('Sucesso', test.validate_only ? 'Payload validado pelo Google.' : 'Conversao enviada.')
    }
  } catch (error: unknown) {
    toast.error('Erro', errorMessage(error, 'Falha ao chamar test-event'))
  } finally {
    testLoading.value = false
  }
}

const formatJSON = (raw: string): string => {
  if (!raw) return ''
  try {
    return JSON.stringify(JSON.parse(raw), null, 2)
  } catch {
    return raw
  }
}
</script>

<style scoped>
.form-section { display: flex; flex-direction: column; gap: 0.75rem; padding-bottom: 1.25rem; border-bottom: 1px solid var(--border-faint); }
.section-header h4 { font-family: var(--font-sans); font-size: var(--fs-lg); font-weight: var(--fw-semibold); color: var(--text-strong); margin: 0; }
.section-hint { font-size: var(--fs-base); color: var(--text-2); margin: 0.25rem 0 0; line-height: 1.5; }
.actions-empty { font-size: var(--fs-base); color: var(--text-2); background: var(--surface-soft); border: 1px dashed var(--border); border-radius: 8px; padding: 0.875rem; }
.grid-2 { display: grid; grid-template-columns: 1fr; gap: 0.75rem; }
@media (min-width: 640px) { .grid-2 { grid-template-columns: 1fr 1fr; } }
.enabled-row { display: flex; align-items: flex-start; gap: 0.625rem; font-size: var(--fs-md); color: var(--text-2); cursor: pointer; user-select: none; }
.enabled-row strong { color: var(--text-1); }
.test-fields { display: flex; flex-direction: column; gap: 0.875rem; }
.test-actions { display: flex; justify-content: flex-end; }
.test-result { display: flex; flex-direction: column; gap: 0.625rem; padding: 0.875rem; border: 1px solid var(--border); border-radius: 8px; background: var(--surface-soft); font-size: var(--fs-base); }
.test-result.result-ok { border-color: var(--success); background: var(--success-soft); }
.test-result.result-fail { border-color: var(--danger-border); background: var(--danger-soft); }
.result-header { display: flex; justify-content: space-between; align-items: center; gap: 0.5rem; flex-wrap: wrap; }
.result-pill { font-weight: var(--fw-semibold); font-size: var(--fs-sm); padding: 0.2rem 0.5rem; border-radius: 999px; background: var(--surface-2); color: var(--text-1); }
.test-result.result-ok .result-pill { background: var(--success); color: var(--text-on-solid); }
.test-result.result-fail .result-pill { background: var(--danger); color: var(--text-on-solid); }
.result-action { font-size: var(--fs-sm); color: var(--text-2); }
.result-action code { background: var(--surface-2); padding: 0.125rem 0.4rem; border-radius: 4px; }
.result-error { color: var(--danger); font-weight: var(--fw-medium); word-break: break-word; }
.test-result details { margin-top: 0.25rem; }
.test-result summary { cursor: pointer; font-weight: var(--fw-medium); color: var(--text-1); padding: 0.25rem 0; user-select: none; }
.test-result pre { margin-top: 0.5rem; background: var(--surface-deep); padding: 0.75rem; border-radius: 6px; overflow: auto; max-height: 320px; font-size: var(--fs-sm); line-height: 1.4; color: var(--text-1); white-space: pre-wrap; word-break: break-all; }
</style>
